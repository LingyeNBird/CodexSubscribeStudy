package study

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func example(t *testing.T) (Report, []byte, string) {
	t.Helper()
	raw, err := os.ReadFile("../../protocol/testdata/synthetic-report.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Body      string `json:"body_base64"`
		Signature string `json:"signature"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	body, err := base64.StdEncoding.DecodeString(fixture.Body)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Decode(body, fixture.Signature, "/api/v1/reports")
	if err != nil {
		t.Fatal("Python fixture invalid", err)
	}
	return report, body, fixture.Signature
}
func sign(t *testing.T, r Report, path string, seed byte) (Report, []byte, string) {
	t.Helper()
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
	r.PublicKey = base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	body, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, append([]byte("CodexSubscribeStudy/1\nPOST\n"+path+"\n"), body...)))
	return r, body, sig
}
func openTest(t *testing.T, capacity int) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "study.db"), capacity)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func call(server http.Handler, path, method string, body []byte, sig string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Study-Signature", sig)
	req.Header.Set("X-Forwarded-For", "sensitive-ip-not-to-be-stored")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}
func TestPythonContractAndMethodDigest(t *testing.T) {
	report, _, _ := example(t)
	if report.MethodDigest != protocol.Digest() || len(report.Summary.Support) != 7 || !report.Summary.Eligible {
		t.Fatal("contract")
	}
	if report.Summary.Support[2] < .9 {
		t.Fatal("fixture not expected cache-read scenario")
	}
}
func TestSignaturesScopeAndTampering(t *testing.T) {
	_, body, sig := example(t)
	for _, path := range []string{"/api/v1/withdraw", "/wrong"} {
		if _, err := Decode(body, sig, path); err == nil {
			t.Fatal("path signature accepted")
		}
	}
	changed := bytes.Replace(body, []byte(`"revision":1`), []byte(`"revision":2`), 1)
	if _, err := Decode(changed, sig, "/api/v1/reports"); err == nil {
		t.Fatal("tampered")
	}
	if _, err := Decode(body, "bad", "/api/v1/reports"); err == nil {
		t.Fatal("bad signature")
	}
}
func TestStrictSchemaDoesNotAcceptPrivateFieldsOrAmbiguousJSON(t *testing.T) {
	r, body, _ := example(t)
	mutations := [][]byte{
		bytes.Replace(body, []byte(`"revision":1`), []byte(`"revision":1,"revision":2`), 1),
		bytes.Replace(body, []byte(`"requests":5000`), []byte(`"requests":5000,"account_id":123`), 1),
		append(body, []byte(" {}")...), []byte(strings.Repeat("[", 10) + strings.Repeat("]", 10)),
	}
	key := ed25519.NewKeyFromSeed(func() []byte {
		b := make([]byte, 32)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	}())
	for i, b := range mutations {
		sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, append([]byte("CodexSubscribeStudy/1\nPOST\n/api/v1/reports\n"), b...)))
		if _, e := Decode(b, sig, "/api/v1/reports"); e == nil {
			t.Fatalf("accepted mutation %d for %s", i, r.PublicKey)
		}
	}
}
func TestSummaryValidation(t *testing.T) {
	cases := map[string]func(*Summary){
		"tiny": func(s *Summary) { s.Requests = 10 }, "nonfinite": func(s *Summary) { s.RawUSD = math.Inf(1) }, "money precision": func(s *Summary) { s.RawUSD = 100.1234 },
		"bad status": func(s *Summary) { s.Status = "account@example.org" }, "missing dimension": func(s *Summary) { s.ScoreCov[6] = nil },
		"nonPSD": func(s *Summary) { s.ScoreCov[1][1] = -2 }, "asymmetric": func(s *Summary) { s.ScoreCov[1][2] += 1 },
		"unsupported confidence": func(s *Summary) { s.Support[0] = 1 }, "privacy exclusions": func(s *Summary) { s.Exclusions["name"] = 1 },
		"no consent coverage": func(s *Summary) { s.GatewayOnly = false }, "bad family factors": func(s *Summary) { s.Factors[2][0] = 2 },
		"invalid rank": func(s *Summary) { s.DesignRank = 5 }, "overflow counts": func(s *Summary) { s.GPT6Requests = math.MaxInt64 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r, _, _ := example(t)
			change(r.Summary)
			if r.Summary.Validate() == nil {
				t.Fatal("invalid summary accepted")
			}
		})
	}
}
func TestLatestSnapshotReplacesAndWithdrawalCannotResurrect(t *testing.T) {
	s := openTest(t, 20)
	r, body, _ := example(t)
	if duplicate, e := s.Put(r, body, false); e != nil || duplicate {
		t.Fatal(e)
	}
	if duplicate, e := s.Put(r, body, false); e != nil || !duplicate {
		t.Fatal("idempotence", e)
	}
	if _, e := s.Put(r, append(body, ' '), false); !errors.Is(e, ErrConflict) {
		t.Fatal("same revision collision", e)
	}
	r.Revision = 2
	r.Summary.RawUSD += 100
	_, body, _ = sign(t, r, "/api/v1/reports", 1)
	r, body, _ = sign(t, r, "/api/v1/reports", 1)
	if _, e := s.Put(r, body, false); e != nil {
		t.Fatal(e)
	}
	// Same identity updates replace its snapshot rather than adding request counts.
	r.Revision = 3
	_, body, _ = sign(t, r, "/api/v1/reports", 1)
	if _, e := s.Put(r, body, false); e != nil {
		t.Fatal(e)
	}
	rows, _, e := s.Snapshot()
	if e != nil || len(rows) != 2 {
		t.Fatal("snapshot count", len(rows), e)
	}
	old := r
	oldBody := body
	r.Revision = 4
	r.Summary = nil
	_, body, _ = sign(t, r, "/api/v1/withdraw", 1)
	if _, e = s.Put(r, body, true); e != nil {
		t.Fatal(e)
	}
	if duplicate, e := s.Put(r, body, true); e != nil || !duplicate {
		t.Fatal("withdraw idempotence", e)
	}
	if _, e = s.Put(old, oldBody, false); !errors.Is(e, ErrStale) {
		t.Fatal("resurrected", e)
	}
	rows, _, _ = s.Snapshot()
	if len(rows) != 1 {
		t.Fatal("withdraw not removed")
	}
}
func TestCapacityAndDurableRetentionAndPersistentRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "study.db")
	s, err := Open(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	r, _, _ := example(t)
	for i := byte(1); i <= 2; i++ {
		r, b, _ := sign(t, r, "/api/v1/reports", i)
		if _, e := s.Put(r, b, false); e != nil {
			t.Fatal(e)
		}
	}
	other, b, _ := sign(t, r, "/api/v1/reports", 3)
	if _, e := s.Put(other, b, false); !errors.Is(e, ErrCapacity) {
		t.Fatal(e)
	}
	other.Summary = nil
	other.Revision = 2
	if _, e := s.Put(other, b, true); !errors.Is(e, ErrCapacity) {
		t.Fatal("tombstone capacity", e)
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if e := s.Backup(backup); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, err = Open(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, _, _ := s.Snapshot()
	if len(rows) != 2 {
		t.Fatal("restart")
	}
	s.now = func() time.Time { return time.Now().Add(121 * 24 * time.Hour) }
	rows, _, _ = s.Snapshot()
	if len(rows) != 2 {
		t.Fatal("inactivity must not erase legacy archive")
	}
	if err = s.Prune(); err != nil {
		t.Fatal(err)
	}
	r, b, _ = sign(t, r, "/api/v1/reports", 1)
	if duplicate, e := s.Put(r, b, false); e != nil || !duplicate {
		t.Fatal("durable duplicate must be idempotent", e)
	}
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte("sensitive-ip-not-to-be-stored")) {
		t.Fatal("IP written")
	}
}
func TestHTTPPublicTotalsNoIndividualData(t *testing.T) {
	s := openTest(t, 10)
	assets := fstest.MapFS{"index.html": {Data: []byte("<h1>study</h1>")}}
	server := NewServer(s, assets)
	r, _, _ := example(t)
	for i := byte(1); i <= 3; i++ {
		_, body, sig := sign(t, r, "/api/v1/reports", i)
		rec := call(server, "/api/v1/reports", "POST", body, sig)
		if rec.Code != 200 {
			t.Fatal(rec.Code, rec.Body.String())
		}
	}
	rec := call(server, "/api/v1/studies/"+StudyID, "GET", nil, "")
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	var result Result
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result.Totals.Contributors != 3 || result.Totals.Requests != r.Summary.Requests*3 || result.Causes[2].Support == nil {
		t.Fatal("aggregate", rec.Body.String())
	}
	for _, key := range []string{"public_key", "sensitive-ip", r.PublicKey, "account_id"} {
		if strings.Contains(rec.Body.String(), key) {
			t.Fatal("private metadata in public data")
		}
	}
	if rec.Header().Get("Set-Cookie") != "" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("privacy headers")
	}
	if call(server, "/studies/anything", "GET", nil, "").Code != 200 {
		t.Fatal("SPA fallback")
	}
	if call(server, "/api/v1/reports/person", "GET", nil, "").Code != 404 {
		t.Fatal("no individual route")
	}
	if call(server, "/assets/missing.js", "GET", nil, "").Code != 404 {
		t.Fatal("bad asset")
	}
}
func TestHTTPGuardsAndRateLimit(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{})
	_, body, sig := example(t)
	tests := []struct {
		path, method string
		body         []byte
		sig          string
		want         int
	}{
		{"/api/v1/reports", "GET", nil, "", 405}, {"/api/v1/reports", "POST", body, "bad", 400},
		{"/api/v1/reports", "POST", bytes.Repeat([]byte{'a'}, MaxBody+1), sig, 413},
		{"/api/v1/nonesuch", "GET", nil, "", 404},
	}
	for _, tc := range tests {
		if got := call(server, tc.path, tc.method, tc.body, tc.sig).Code; got != tc.want {
			t.Fatalf("got %d wanted %d", got, tc.want)
		}
	}
	req := httptest.NewRequest("POST", "/api/v1/reports", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != 415 {
		t.Fatal("content type")
	}
	server.mu.Lock()
	server.tokens = 0
	server.lastToken = time.Now()
	server.mu.Unlock()
	if call(server, "/api/v1/reports", "POST", body, sig).Code != 429 {
		t.Fatal("rate limit")
	}
}
func TestAggregationInsufficientHeterogeneousAndDeterministic(t *testing.T) {
	r, _, _ := example(t)
	if Aggregate(nil, 0).State != "no_data" {
		t.Fatal("empty")
	}
	if Aggregate([]Summary{*r.Summary, *r.Summary}, 0).Causes[0].Support != nil {
		t.Fatal("tiny confidence")
	}
	summaries := []Summary{*r.Summary, *r.Summary, *r.Summary}
	a, b := Aggregate(summaries, 0), Aggregate(summaries, 0)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Fatal("not deterministic")
	}
	sum := 0.
	for _, c := range a.Causes {
		sum += *c.Support
	}
	if math.Abs(sum-1) > 1e-8 {
		t.Fatal(sum)
	}
	for i := range summaries {
		clone, _, _ := example(t)
		summaries[i] = *clone.Summary
		for j := 0; j < 7; j++ {
			summaries[i].ScoreMean[j] = -1
			for k := 0; k < 7; k++ {
				summaries[i].ScoreCov[j][k] = 0
			}
		}
		summaries[i].ScoreMean[0] = 0
		summaries[i].ScoreMean[2] = 1
	}
	summaries[2].ScoreMean[2] = -1
	summaries[2].ScoreMean[5] = 2
	if Aggregate(summaries, 0).State != "heterogeneous" {
		t.Fatal("dissent hidden")
	}
}
func TestConcurrentSubmissionsDoNotAddDuplicateSnapshots(t *testing.T) {
	store := openTest(t, 50)
	r, _, _ := example(t)
	var wg sync.WaitGroup
	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(revision int) {
			defer wg.Done()
			copy := r
			copy.Revision = uint64(revision)
			body, _ := json.Marshal(copy)
			_, e := store.Put(copy, body, false)
			if e != nil && !errors.Is(e, ErrStale) {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	rows, _, e := store.Snapshot()
	if e != nil || len(rows) != 1 {
		t.Fatal(fmt.Sprint(e), len(rows))
	}
}
