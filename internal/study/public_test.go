package study

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestPublicPivotExposesSubmissionsWithoutAnnotations(t *testing.T) {
	store := openTest(t, 100)
	server := NewServer(store, fstest.MapFS{})

	risk := 20
	if err := store.putSurvey(SurveySubmission{
		Status:  []string{"降智"},
		Answers: map[string][]string{"plans": {"Plus"}, "usage": {"直登"}},
		Details: map[string]string{"country": "SG", "concurrency": "15"},
		IPRisk:  &risk,
	}); err != nil {
		t.Fatal(err)
	}
	note := "跟进中"
	if _, err := store.annotate(annotationsRequest{IDs: []uint64{1}, AddTags: []string{"可疑"}, Note: &note}); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/pivot/submissions", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public read rejected: %d", response.Code)
	}
	if header := response.Header().Get("X-Robots-Tag"); header != "noindex, nofollow" {
		t.Fatalf("public response is indexable: %q", header)
	}

	var payload publicSubmissions
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 1 || len(payload.Records) != 1 {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	record := payload.Records[0]
	if got := record.Raw["country"]; len(got) != 1 || got[0] != "SG" {
		t.Fatalf("submitted data not exposed: %+v", record.Raw)
	}
	if record.IPRisk == nil || *record.IPRisk != risk {
		t.Fatalf("ip risk not exposed: %+v", record.IPRisk)
	}

	// The response JSON must not carry a tags/note field at all, not merely
	// an empty one, since publicSubmission has no such field to populate.
	var raw map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	rawRecord := raw["records"].([]any)[0].(map[string]any)
	if _, ok := rawRecord["tags"]; ok {
		t.Fatalf("investigator tags leaked into public response: %v", rawRecord)
	}
	if _, ok := rawRecord["note"]; ok {
		t.Fatalf("investigator note leaked into public response: %v", rawRecord)
	}
}

func TestPublicPivotCatalogMatchesAdminCatalog(t *testing.T) {
	store := openTest(t, 100)
	server := NewServer(store, fstest.MapFS{})
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/pivot/catalog", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public catalog rejected: %d", response.Code)
	}
	if response.Body.String() != string(surveyCatalogJSON) {
		t.Fatal("public catalog does not match the questionnaire definition")
	}
}

func TestPublicPivotWorksWithoutAdminCredentials(t *testing.T) {
	store := openTest(t, 100)
	server := NewServer(store, fstest.MapFS{}) // no WithAdmin option
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/pivot/submissions", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("pivot feed should not require admin configuration: %d", response.Code)
	}
}

func TestPublicPivotRateLimited(t *testing.T) {
	store := openTest(t, 100)
	server := NewServer(store, fstest.MapFS{})
	server.pivotTokens = 1
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/pivot/submissions", nil))
		if i == 0 && response.Code != http.StatusOK {
			t.Fatalf("first request should be allowed: %d", response.Code)
		}
		if i == 1 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("second request should be throttled: %d", response.Code)
		}
	}
}
