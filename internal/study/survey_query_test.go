package study

import (
	"encoding/json"
	"net/http"
	"testing"
	"testing/fstest"
)

func querySurvey(t *testing.T, server *Server, body string) *surveySummary {
	t.Helper()
	response := surveyCall(server, http.MethodPost, "/api/survey/statistics/query", body, "192.0.2.1:80", "")
	if response.Code != http.StatusOK {
		t.Fatalf("query failed: %d %s", response.Code, response.Body.String())
	}
	var result surveySummary
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}
func TestConditionalSurveyStatisticsCombinesPrerequisites(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	records := []SurveySubmission{
		{Status: []string{"风控（限流）"}, Answers: map[string][]string{"plans": {"Plus"}}, Details: map[string]string{"country": "PH", "concurrency": "8"}},
		{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Plus"}}, Details: map[string]string{"country": "PH", "concurrency": "1"}},
		{Status: []string{"风控（限流）"}, Answers: map[string][]string{"plans": {"Pro 5X"}}, Details: map[string]string{"country": "PH", "concurrency": "1"}},
		{Status: []string{"封号"}, Answers: map[string][]string{"plans": {"Plus"}}, Details: map[string]string{"country": "JP", "concurrency": "8"}},
		{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Free 免费"}}, Details: map[string]string{"country": "US", "concurrency": "unknown"}},
	}
	for _, record := range records {
		if err := validateSurvey(record); err != nil {
			t.Fatal(err)
		}
		if err := store.putSurvey(record); err != nil {
			t.Fatal(err)
		}
	}
	philippines := querySurvey(t, server, `{"prerequisites":[{"factor":"country","options":["菲律宾"]}]}`)
	if philippines.Statuses != [8]int{1, 0, 0, 0, 2, 0, 0, 0} {
		t.Fatal("country premise selected wrong rows", philippines.Statuses)
	}
	concurrency := philippines.Factors[0]
	for _, factor := range philippines.Factors {
		if factor.ID == "concurrency" {
			concurrency = factor
		}
	}
	if testSurveyCount(concurrency.Applicable) != 3 {
		t.Fatal("conditional concurrency denominator")
	}
	counts := map[string][8]int{}
	for _, option := range concurrency.Options {
		counts[option.Label] = option.Counts
	}
	if counts["6–10"][4] != 1 || counts["1"][0] != 1 || counts["1"][4] != 1 {
		t.Fatal("conditional derived bucket counts", counts)
	}
	orCountries := querySurvey(t, server, `{"prerequisites":[{"factor":"country","options":["菲律宾","日本"]}]}`)
	if testSurveyCount(orCountries.Statuses) != 4 || orCountries.Statuses[2] != 1 {
		t.Fatal("same-factor options must use OR")
	}
	andPlan := querySurvey(t, server, `{"prerequisites":[{"factor":"country","options":["菲律宾"]},{"factor":"plans","options":["Plus"]}]}`)
	if testSurveyCount(andPlan.Statuses) != 2 || andPlan.Statuses[0] != 1 || andPlan.Statuses[4] != 1 {
		t.Fatal("different factors must use AND")
	}
	empty := querySurvey(t, server, `{"prerequisites":[{"factor":"country","options":["玻利维亚"]}]}`)
	if testSurveyCount(empty.Statuses) != 0 || empty.Range.FirstSubmissionID != 0 || empty.Range.LastSubmissionID != 0 {
		t.Fatal("empty premise fabricated range")
	}
}
func TestConditionalSurveyStatisticsRejectsInvalidQueries(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{})
	for _, body := range []string{`{}`, `{"prerequisites":[{"factor":"unknown","options":["x"]}]}`, `{"prerequisites":[{"factor":"country","options":[]}]}`, `{"prerequisites":[{"factor":"country","options":["菲律宾"]},{"factor":"country","options":["日本"]}]}`, `{"prerequisites":[{"factor":"country","options":["菲律宾","菲律宾"]}]}`, `{"prerequisites":[],"extra":true}`, `{"prerequisites":[]} trailing`} {
		if response := surveyCall(server, http.MethodPost, "/api/survey/statistics/query", body, "192.0.2.1:80", ""); response.Code != http.StatusBadRequest {
			t.Fatalf("accepted %s: %d", body, response.Code)
		}
	}
	if response := surveyCall(server, http.MethodGet, "/api/survey/statistics/query", "", "192.0.2.1:80", ""); response.Code != http.StatusMethodNotAllowed {
		t.Fatal("GET query accepted")
	}
}
