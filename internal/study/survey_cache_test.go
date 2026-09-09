package study

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func testSurveyCount(counts [8]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func cacheQuestion(index int) SurveySubmission {
	states := [][]string{{"正常"}, {"降智"}, {"封号"}, {"降智", "封号"}, {"风控（限流）"}, {"降智", "风控（限流）"}, {"封号", "风控（限流）"}, {"降智", "封号", "风控（限流）"}}
	q := SurveySubmission{Status: states[index%8], Answers: map[string][]string{"plans": {"Plus"}, "usage": {"直登", "反代"}, "network": {"家宽"}}, Details: map[string]string{}}
	if index%3 == 0 {
		q.Answers["plans"] = []string{"Free 免费"}
		q.Answers["network"] = []string{"机场"}
	}
	if index%2 == 0 {
		q.Answers["tools"] = []string{"Pi", "Cursor"}
		q.Details["toolMode:Pi"] = "反代"
		q.Details["toolMode:Cursor"] = "反代"
	}
	if index%5 != 0 {
		risk := index % 101
		q.IPRisk = &risk
	}
	if index%3 != 0 {
		q.UsagePattern = make([]*int, 24)
		for hour := range q.UsagePattern {
			value := (index + hour) % 25
			q.UsagePattern[hour] = &value
		}
	}
	if q.degraded() {
		q.Answers["discovery"] = []string{"画鹈鹕"}
	}
	if q.limited() {
		q.Answers["limitedDiscovery"] = []string{"服务不可用"}
	}
	return q
}

func TestSurveyIncrementalMatchesRawEvidence(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	var records []SurveySubmission
	for range 4 {
		for range 13 {
			q := cacheQuestion(len(records))
			if err := validateSurvey(q); err != nil {
				t.Fatal(err)
			}
			if err := store.putSurvey(q); err != nil {
				t.Fatal(err)
			}
			records = append(records, q)
		}
		result, err := server.surveyStatistics()
		if err != nil {
			t.Fatal(err)
		}
		if testSurveyCount(result.Statuses) != len(records) || result.Range.LastSubmissionID != uint64(len(records)) {
			t.Fatal("incremental total or cutoff")
		}
		for _, factor := range result.Factors {
			var expectedApplicable [8]int
			expectedOptions := make(map[string][8]int)
			for _, q := range records {
				mask := 0
				for _, status := range q.Status {
					switch status {
					case "降智":
						mask |= 1
					case "封号":
						mask |= 2
					case "风控（限流）":
						mask |= 4
					}
				}
				values := q.normalized()[factor.ID]
				if len(values) == 0 {
					continue
				}
				expectedApplicable[mask]++
				for index, value := range values {
					if slices.Contains(values[:index], value) {
						continue
					}
					counts := expectedOptions[value]
					counts[mask]++
					expectedOptions[value] = counts
				}
			}
			if factor.Applicable != expectedApplicable {
				t.Fatalf("%s applicability: %v != %v", factor.ID, factor.Applicable, expectedApplicable)
			}
			for _, option := range factor.Options {
				if option.Counts != expectedOptions[option.Label] {
					t.Fatalf("%s %s counts: %v != %v", factor.ID, option.Label, option.Counts, expectedOptions[option.Label])
				}
			}
		}
		sums := [24]int{}
		n := 0
		for _, q := range records {
			if q.UsagePattern != nil {
				n++
				for hour, value := range q.UsagePattern {
					sums[hour] += *value
				}
			}
		}
		if result.UsagePattern.Total != n {
			t.Fatal("usage denominator")
		}
		for hour, sum := range sums {
			if result.UsagePattern.Sums[hour] != sum {
				t.Fatal("usage sums lost")
			}
		}
	}
}

func TestSurveyCacheSurvivesRestartAndSkipsOldRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, fstest.MapFS{})
	if err := store.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	first, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	// A cached historical record cannot be aggregated again; persisted counts must be reused.
	if err := store.db.Update(func(tx *sql.Tx) error {
		raw, err := readSurveyAggregate(tx)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(raw)
		if !valid {
			return fmt.Errorf("missing cache")
		}
		state.Version = 1
		legacy, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE survey_statistics SET payload=? WHERE id=1", string(legacy)); err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE survey_submissions SET payload='{"usagePattern":[]}' WHERE id=1`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server = NewServer(store, fstest.MapFS{})
	hit, err := server.surveyStatistics()
	if err != nil {
		t.Fatal("cache rescanned history", err)
	}
	if !reflect.DeepEqual(first, hit) {
		t.Fatal("cache hit changed result")
	}
	if err := store.putSurvey(cacheQuestion(7)); err != nil {
		t.Fatal(err)
	}
	delta, err := server.surveyStatistics()
	if err != nil {
		t.Fatal("incremental refresh rescanned history", err)
	}
	if testSurveyCount(delta.Statuses) != 2 || delta.Statuses[0] != 1 || delta.Statuses[7] != 1 || delta.Range.LastSubmissionID != 2 {
		t.Fatal("lost persistent base counts")
	}
}

func TestSurveyCacheRefreshIsAtomicAndConcurrent(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	first, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if testSurveyCount(first.Statuses) != 0 || first.Range.LastSubmissionID != 0 || first.Range.FirstSubmittedAt != nil {
		t.Fatal("empty range")
	}
	if err := store.putSurvey(cacheQuestion(1)); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO survey_submissions(status_mask,payload) VALUES(0,'{"usagePattern":[]}')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.surveyStatistics(); err == nil {
		t.Fatal("invalid incremental record accepted")
	}
	if err := store.db.View(func(tx *sql.Tx) error {
		raw, err := readSurveyAggregate(tx)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(raw)
		if !valid || testSurveyCount(state.Counts.Statuses) != 0 || state.Range.LastSubmissionID != 0 {
			return fmt.Errorf("failed refresh advanced progress")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *sql.Tx) error {
		body, _ := json.Marshal(cacheQuestion(2))
		_, err := tx.Exec("UPDATE survey_submissions SET status_mask=2,payload=? WHERE id=2", string(body))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 40)
	for i := range 20 {
		wg.Add(2)
		go func(i int) { defer wg.Done(); failures <- store.putSurvey(cacheQuestion(i)) }(i)
		go func() { defer wg.Done(); _, err := server.surveyStatistics(); failures <- err }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if testSurveyCount(result.Statuses) != 22 || result.Range.LastSubmissionID != 22 {
		t.Fatal("concurrent refresh lost or duplicated records")
	}
}

func TestSurveyServerTimeAndLegacyRiskMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"network":["宽带"],"quality":["优秀"]}}`
	legacyPath := createLegacySurveys(t, "", map[uint64][]byte{1: []byte(legacy)}, 1)
	if _, err := store.ImportBbolt(legacyPath); err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	store.now = func() time.Time { return now }
	server := NewServer(store, fstest.MapFS{})
	submit := func(body string) int {
		r := httptest.NewRequest("POST", "/api/survey/submissions", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w.Code
	}
	for _, risk := range []int{0, 100} {
		if code := submit(fmt.Sprintf(`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"network":["机场"]},"ipRisk":%d}`, risk)); code != 201 {
			t.Fatal(code)
		}
	}
	for _, body := range []string{
		`{"status":["正常"],"answers":{"plans":["Plus"]},"ipRisk":10}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"]},"ipRisk":101}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"]},"ipRisk":-1}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"submittedAt":"2000-01-01T00:00:00Z"}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"quality":["优秀"]}}`,
	} {
		if submit(body) != 400 {
			t.Fatal("invalid risk or client timestamp accepted")
		}
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Range.UnknownTimeCount != 1 || stats.Range.FirstSubmittedAt == nil || !stats.Range.FirstSubmittedAt.Equal(now) || !stats.Range.LastSubmittedAt.Equal(now) {
		t.Fatal("server time / legacy unknown time")
	}
	for _, group := range stats.Factors {
		if group.ID == "ipRisk" && testSurveyCount(group.Applicable) != 2 {
			t.Fatal("legacy rating fabricated a numeric score")
		}
		if group.ID == "network" && (testSurveyCount(group.Options[0].Counts) != 1 || testSurveyCount(group.Options[2].Counts) != 2) {
			t.Fatal("network migration")
		}
	}
	if err := store.db.View(func(tx *sql.Tx) error {
		var body []byte
		if err := tx.QueryRow("SELECT payload FROM survey_submissions WHERE id=1").Scan(&body); err != nil {
			return err
		}
		var record storedSurveyRecord
		if err := json.Unmarshal(body, &record); err != nil {
			return err
		}
		if record.SubmittedAt != nil || string(record.LegacyIPQuality) != `["优秀"]` {
			return fmt.Errorf("legacy data lost or timestamp fabricated")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSurveyCacheInvalidationRebuildsEvidence(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	if err := store.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := server.surveyStatistics(); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *sql.Tx) error {
		raw, err := readSurveyAggregate(tx)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(raw)
		if !valid {
			return fmt.Errorf("missing cache")
		}
		state.CatalogDigest = "previous-catalog"
		body, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE survey_statistics SET payload=? WHERE id=1", string(body)); err != nil {
			return err
		}
		replacement, err := json.Marshal(cacheQuestion(7))
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE survey_submissions SET status_mask=7,payload=? WHERE id=1", string(replacement))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Statuses != [8]int{0, 0, 0, 0, 0, 0, 0, 1} {
		t.Fatal("stale cached counts reused after invalidation")
	}
}
