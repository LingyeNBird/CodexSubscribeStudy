package study

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	bolt "go.etcd.io/bbolt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func surveyCall(s *Server, method, path, body, peer, forwarded string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = peer
	r.Header.Set("Content-Type", "application/json")
	if forwarded != "" {
		r.Header.Set("X-Forwarded-For", forwarded)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestSurveyPersistenceAndRepeatedSubmission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, fstest.MapFS{})
	check := func(w *httptest.ResponseRecorder, code int) {
		t.Helper()
		if w.Code != code {
			t.Fatalf("%d: %s", w.Code, w.Body.String())
		}
	}
	peer := "192.0.2.1:1000"
	body := `{"status":["降智","封号"],"answers":{"plans":["Plus"],"usage":["反代"],"proxy":["其他"]},"details":{"proxyOther":"private-tool","duration":"24","durationUnit":"小时","degradationTime":"2026-09-01","banTime":"2026-09-02T10:15"}}`
	for range 2 {
		check(surveyCall(server, "POST", "/api/survey/submissions", body, peer, "198.51.100.1"), 201)
	}
	check(surveyCall(server, "POST", "/api/survey/submissions", `{"status":["正常"],"answers":{"plans":["Free 免费"]}}`, "192.0.2.2:1000", ""), 201)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server = NewServer(store, fstest.MapFS{})
	w := surveyCall(server, "GET", "/api/survey/statistics", "", peer, "")
	check(w, 200)
	var stats surveyStatistics
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 || stats.Both != 2 || stats.Normal != 1 || stats.Degraded != 2 || stats.Banned != 2 {
		t.Fatalf("incorrect totals %+v", stats)
	}
	for _, g := range stats.Associations {
		if g.ID == "proxy" {
			if g.Total != 2 || g.Rows[2].Degraded.Phi != nil {
				t.Fatal("ineligible normal counted as unselected")
			}
		}
	}
	for _, factor := range stats.Factors {
		if factor.ID == "duration" && factor.Groups["degraded"].Rows[1].Count != 2 {
			t.Fatal("duration normalization")
		}
	}
	for _, secret := range []string{"private-tool", "192.0.2.1", "degradationTime"} {
		if bytes.Contains(w.Body.Bytes(), []byte(secret)) {
			t.Fatal("private record escaped aggregation")
		}
	}
}
func TestSurveyRejectsInvalidAndHiddenAnswers(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	bodies := []string{
		`{"status":[],"answers":{"plans":["Plus"]}}`,
		`{"status":["正常","风控（限流）"],"answers":{"plans":["Plus"]}}`,
		`{"status":["降智","降智"],"answers":{"plans":["Plus"]}}`,
		`{"status":["降智并封号"],"answers":{"plans":["Plus"]}}`,
		`{"status":"正常","answers":{"plans":["Plus"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"email":"a@example.com"}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"models":["GPT-6 Astra"]}}`,
		`{"status":["正常"],"answers":{"plans":["Free 免费"],"activation":["代充"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"proxy":["CPA"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代","反代"]}}`,
		`{"status":["正常"],"answers":{"plans":[null]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"details":{"duration":"NaN","durationUnit":"天"}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"details":{"people":"2"}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"details":{"banTime":"2026-09-01"}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"details":{"concurrency":"-1"}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]}} {}`,
	}
	for _, body := range bodies {
		if w := surveyCall(server, "POST", "/api/survey/submissions", body, "192.0.2.1:42", ""); w.Code != 400 {
			t.Fatalf("accepted %s: %d", body, w.Code)
		}
	}
	stats, err := server.surveyStatistics()
	if err != nil || stats.Total != 0 {
		t.Fatal("invalid submissions persisted")
	}
}

func TestSurveyStatusMigrationAndRateLimiting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	err = store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		for _, status := range []string{"正常", "降智", "封号", "降智并封号"} {
			body, err := json.Marshal(map[string]any{"status": status, "answers": map[string][]string{"plans": {"Plus"}}})
			if err != nil {
				return err
			}
			id, err := bucket.NextSequence()
			if err != nil {
				return err
			}
			var key [8]byte
			binary.BigEndian.PutUint64(key[:], id)
			if err := bucket.Put(key[:], body); err != nil {
				return err
			}
		}
		return tx.Bucket(surveyCacheBucket).Delete(surveyStorageVersionKey)
	})
	if err != nil {
		store.Close()
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
	server := NewServer(store, fstest.MapFS{})
	for _, body := range []string{
		`{"status":["风控（限流）"],"answers":{"plans":["Free 免费"]}}`,
		`{"status":["降智","封号","风控（限流）"],"answers":{"plans":["Plus"]}}`,
	} {
		if w := surveyCall(server, "POST", "/api/survey/submissions", body, "192.0.2.1:80", ""); w.Code != 201 {
			t.Fatal(w.Body.String())
		}
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 6 || stats.Normal != 1 || stats.Degraded != 3 || stats.Banned != 3 || stats.Limited != 2 || stats.Both != 2 {
		t.Fatalf("wrong totals: %+v", stats)
	}
	sum := 0
	for _, status := range stats.Statuses {
		sum += status.Count
	}
	if sum != 6 || stats.Statuses[4].Count != 1 || stats.Statuses[7].Count != 1 {
		t.Fatal("incorrect combination counts")
	}
	for _, factor := range stats.Factors {
		if factor.ID == "plans" && factor.Groups["limited"].Total != 2 {
			t.Fatal("missing rate-limited distribution")
		}
	}
	for _, group := range stats.Associations {
		if group.ID == "plans" && (group.Rows[0].Limited.Selected.Events != 1 || group.Rows[0].Limited.Unselected.Events != 1) {
			t.Fatal("incorrect rate-limited association")
		}
	}
}

func TestSurveyDirectIPAndStability(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{})
	for _, body := range []string{
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["直登"],"network":["家宽"],"ipStability":["固定 IP"]},"details":{"exitCountry":"JP"},"ipRisk":0}`,
		`{"status":["封号"],"answers":{"plans":["Plus"],"usage":["反代"],"network":["机场"],"ipStability":["IP 乱飞"]},"details":{"exitCountry":"unknown"},"ipRisk":90}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["直登"]}}`,
	} {
		if w := surveyCall(server, "POST", "/api/survey/submissions", body, "192.0.2.1:80", ""); w.Code != 201 {
			t.Fatal(w.Body.String())
		}
	}
	for _, body := range []string{
		`{"status":["正常"],"answers":{"plans":["Plus"],"ipStability":["固定 IP"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["直登"],"ipStability":["其他"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["直登"],"proxy":["CPA"]}}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"details":{"exitCountry":"JP"}}`,
	} {
		if w := surveyCall(server, "POST", "/api/survey/submissions", body, "192.0.2.1:80", ""); w.Code != 400 {
			t.Fatal("inapplicable IP answer accepted")
		}
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, group := range stats.Associations {
		if group.ID == "ipStability" {
			found = true
			if group.Total != 2 || group.Rows[1].Banned.Selected.Events != 1 || group.Rows[1].Banned.Unselected.Events != 0 {
				t.Fatal("wrong stability comparison population")
			}
		}
		if group.ID == "ipRisk" && group.Total != 2 {
			t.Fatal("direct IP omitted from risk statistics")
		}
	}
	if !found {
		t.Fatal("missing IP stability statistics")
	}
}
