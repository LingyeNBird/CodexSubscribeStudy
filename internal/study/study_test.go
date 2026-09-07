package study

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func openTest(t *testing.T, maximum int) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "study.db"), maximum)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func call(handler http.Handler, endpoint, method string, body []byte, signature string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, endpoint, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if signature != "" {
		request.Header.Set("X-Study-Signature", signature)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

// A single-request, single-interval fixture has exactly flat evidence.
func exampleReport(t *testing.T) (Report, []byte, string) {
	t.Helper()
	summary := &Summary{Requests: 1, GPT6Requests: 1, RawUSD: 1, GPT6RawUSD: 1, QuotaPoints: 1,
		Intervals: 1, Groups: 1, Quality: map[string]int64{}, LogEvidence: make([][]float64, 3),
		GPT6Quota: make([]float64, len(points)), Information: make([][]float64, 4)}
	for _, key := range Quality {
		summary.Quality[key] = 0
	}
	for i := range summary.LogEvidence {
		summary.LogEvidence[i] = make([]float64, len(points))
	}
	for i := range summary.GPT6Quota {
		summary.GPT6Quota[i] = 1
	}
	for i := range summary.Information {
		summary.Information[i] = make([]float64, 4)
	}
	r := Report{Protocol: Protocol, StudyID: StudyID, Method: Method, MethodDigest: protocol.Digest(),
		Revision: 1, BatchID: "11111111-1111-4111-8111-111111111111", Summary: summary}
	r, body, sig := signReport(t, r, "/api/reports", 80)
	if _, err := Decode(body, sig, "/api/reports"); err != nil {
		t.Fatal(err)
	}
	return r, body, sig
}
func signReport(t *testing.T, r Report, endpoint string, seed byte) (Report, []byte, string) {
	t.Helper()
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
	r.PublicKey = base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey))
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, append([]byte("CodexSubscribeStudy\nPOST\n"+endpoint+"\n"), data...)))
	return r, data, signature
}
func TestSignatureGridAndOneRequestAdmission(t *testing.T) {
	r, body, sig := exampleReport(t)
	if r.Summary.Requests != 1 || r.Summary.Intervals != 1 {
		t.Fatal("tiny fixture")
	}
	var method struct {
		GridSHA256     string `json:"grid_sha256"`
		CandidateCount int    `json:"candidate_count"`
	}
	if json.Unmarshal(protocol.Method, &method) != nil {
		t.Fatal("method")
	}
	if len(points) != 1311 || method.CandidateCount != 1311 || method.GridSHA256 != GridDigest() {
		t.Fatal("grid mismatch")
	}
	if _, e := Decode(body, sig, "/wrong"); e == nil {
		t.Fatal("signature not bound to route")
	}
	changed := append([]byte{}, body...)
	changed[len(changed)/2] ^= 1
	if _, e := Decode(changed, sig, "/api/reports"); e == nil {
		t.Fatal("tamper")
	}
}
func TestNoMinimumContributorsOrRank(t *testing.T) {
	s := openTest(t, 10)
	r, body, _ := exampleReport(t)
	if _, e := s.Put(r, body); e != nil {
		t.Fatal(e)
	}
	result, e := s.Aggregate()
	if e != nil {
		t.Fatal(e)
	}
	if result.Totals.Contributors != 1 || result.Totals.Requests != 1 || result.Causes[0].Support != nil {
		t.Fatal("single contribution was rejected or invented evidence")
	}
	raw, _ := json.Marshal(result)
	for _, name := range []string{"auxiliary", "capacity_context", "particle", "constant_capacity"} {
		if bytes.Contains(raw, []byte(name)) {
			t.Fatal("capacity estimate field in public result", name)
		}
	}
}
func TestReplaceAppendAndRetain(t *testing.T) {
	s := openTest(t, 10)
	r, _, _ := exampleReport(t)
	r, b, _ := signReport(t, r, "/api/reports", 11)
	if _, e := s.Put(r, b); e != nil {
		t.Fatal(e)
	}
	if duplicate, e := s.Put(r, b); e != nil || !duplicate {
		t.Fatal("duplicate", e)
	}
	old, oldBody := r, b
	r.Revision = 2
	r.Summary.Requests = 2
	r.Summary.GPT6Requests = 2
	r, b, _ = signReport(t, r, "/api/reports", 11)
	if _, e := s.Put(r, b); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Put(old, oldBody); !errors.Is(e, ErrConflict) {
		t.Fatal("old revision accepted", e)
	}
	result, _ := s.Aggregate()
	if result.Totals.Requests != 2 || result.Totals.Batches != 1 {
		t.Fatal("replace")
	}
	r.Revision = 3
	r.BatchID = "22222222-2222-4222-8222-222222222222"
	r, b, _ = signReport(t, r, "/api/reports", 11)
	if _, e := s.Put(r, b); e != nil {
		t.Fatal(e)
	}
	result, _ = s.Aggregate()
	if result.Totals.Requests != 4 || result.Totals.Batches != 2 {
		t.Fatal("history discarded")
	}
	if e := s.Backup(filepath.Join(t.TempDir(), "backup.db")); e != nil {
		t.Fatal(e)
	}
}

func TestJointGridNotLocalPercentageAverage(t *testing.T) {
	a := newAccumulator()
	r, _, _ := exampleReport(t)
	s := *r.Summary
	s.Intervals = 2
	s.Groups = 1
	s.Contrasts = 1
	// Each contributor gives one rank-one direction. All four jointly identify
	// beta=(1,1,2,1), without any contributor being independently rank four.
	target := [4]float64{1, 1, 2, 1}
	for component := 0; component < 4; component++ {
		for j, p := range points {
			cost := -10 * math.Pow(p.Factors[component]-target[component], 2)
			null := -10 * math.Pow(1-target[component], 2)
			for k := 0; k < 3; k++ {
				s.LogEvidence[k][j] = cost - null
			}
		}
		for i := 0; i < 4; i++ {
			for j := 0; j < 4; j++ {
				s.Information[i][j] = 0
			}
		}
		s.Information[component][component] = 1
		if e := s.Validate(); e != nil {
			t.Fatal(e)
		}
		_ = a.add(string(rune('a'+component)), s, 1)
	}
	result := a.finish()
	if result.InformationRank != 4 || result.Causes[2].Support == nil || *result.Causes[2].Support < .8 {
		t.Fatal("joint information not pooled", result.Causes)
	}
	if len(result.Parameters) != 4 {
		t.Fatal("parameters")
	}
	flat := make([]float64, len(points))
	_, mass := causes(flat, true)
	for _, m := range mass {
		if math.Abs(m-1./7) > 1e-10 {
			t.Fatal("prior applied per grid point instead of family")
		}
	}
}
func TestStrictSchemaAndNoPII(t *testing.T) {
	cases := map[string]func(*Report){
		"negative count":              func(r *Report) { r.Summary.Requests = -1 },
		"count mismatch":              func(r *Report) { r.Summary.OtherRequests = 2 },
		"bad dimensions":              func(r *Report) { r.Summary.LogEvidence = r.Summary.LogEvidence[:2] },
		"nonzero baseline":            func(r *Report) { r.Summary.LogEvidence[0][nullPoint] = 1 },
		"impossible attribution":      func(r *Report) { r.Summary.GPT6Quota[0] = 999 },
		"indefinite matrix":           func(r *Report) { r.Summary.Information[0][0] = -1 },
		"unknown quality":             func(r *Report) { r.Summary.Quality["ip"] = 1 },
		"invalid uuid":                func(r *Report) { r.BatchID = "account-123" },
		"wrong digest":                func(r *Report) { r.MethodDigest = "bad" },
		"spurious singleton evidence": func(r *Report) { r.Summary.LogEvidence[1][0] = 2 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r, _, _ := exampleReport(t)
			change(&r)
			_, b, sig := signReport(t, r, "/api/reports", 6)
			if _, e := Decode(b, sig, "/api/reports"); e == nil {
				t.Fatal("accepted")
			}
		})
	}
	r, _, _ := exampleReport(t)
	r, b, _ := signReport(t, r, "/api/reports", 6)
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{6}, 32))
	mutations := [][]byte{bytes.Replace(b, []byte(`"requests":`), []byte(`"Requests":`), 1), append([]byte(`{"ip":"127.0.0.1",`), b[1:]...), bytes.Replace(b, []byte(`"gateway_only":false,`), nil, 1)}
	for _, bad := range mutations {
		sig := base64.StdEncoding.EncodeToString(ed25519.Sign(private, append([]byte("CodexSubscribeStudy\nPOST\n/api/reports\n"), bad...)))
		if _, e := Decode(bad, sig, "/api/reports"); e == nil {
			t.Fatal("unknown or missing field accepted")
		}
	}
	_ = r
}
func TestPublicHTTPNoRawIdentity(t *testing.T) {
	s := openTest(t, 10)
	server := NewServer(s, fstest.MapFS{"index.html": {Data: []byte("hello")}})
	r, body, sig := exampleReport(t)
	res := call(server, "/api/reports", "POST", body, sig)
	if res.Code != 200 {
		t.Fatal(res.Code, res.Body.String())
	}
	res = call(server, "/api/studies/"+StudyID, "GET", nil, "")
	if res.Code != 200 {
		t.Fatal(res.Code)
	}
	for _, private := range []string{r.PublicKey, r.BatchID, "log_evidence", "account_id"} {
		if bytes.Contains(res.Body.Bytes(), []byte(private)) {
			t.Fatal("raw identity or evidence disclosed")
		}
	}
}

// Independently enumerate exp(likelihood)/family-count to check prior-once
// pooling, rather than calling the implementation under test for expectations.
func TestPoolingMatchesDirectEnumeration(t *testing.T) {
	counts := [7]int{}
	for _, p := range points {
		counts[p.Family]++
	}
	for _, copies := range []int{1, 5, 30} {
		curve := make([]float64, len(points))
		expected := [7]float64{}
		denom := 0.
		for i, p := range points {
			loss := 0.
			for j, v := range p.Factors {
				target := 1.
				if j == 2 {
					target = 2
				}
				loss += math.Pow(v-target, 2)
			}
			curve[i] = -float64(copies) * loss
			value := math.Exp(curve[i]) / float64(counts[p.Family]) / 7
			expected[p.Family] += value
			denom += value
		}
		_, mass := causes(curve, true)
		for i, v := range mass {
			if math.Abs(v-expected[i]/denom) > 1e-12 {
				t.Fatal("pooling differs", copies, i)
			}
		}
	}
}

func TestRejectsEveryCapacityEstimateField(t *testing.T) {
	_, body, _ := exampleReport(t)
	for _, field := range []string{"capacity_context", "auxiliary_evidence", "auxiliary_groups", "particle_capacity", "constant_capacity", "estimated_capacity", "effective_usd_per_percent"} {
		t.Run(field, func(t *testing.T) {
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			raw["summary"].(map[string]any)[field] = 1
			b, _ := json.Marshal(raw)
			key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{80}, 32))
			sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, append([]byte("CodexSubscribeStudy\nPOST\n/api/reports\n"), b...)))
			if _, e := Decode(b, sig, "/api/reports"); e == nil {
				t.Fatal("estimate accepted")
			}
		})
	}
}

func TestZeroRequestStatisticsAndConflictingRevision(t *testing.T) {
	store := openTest(t, 10)
	r, _, _ := exampleReport(t)
	r.Summary.Requests = 0
	r.Summary.GPT6Requests = 0
	r.Summary.RawUSD = 0
	r.Summary.GPT6RawUSD = 0
	r.Summary.Intervals = 0
	r.Summary.Groups = 0
	r.Summary.QuotaPoints = 0
	for i := range r.Summary.GPT6Quota {
		r.Summary.GPT6Quota[i] = 0
	}
	r, b, sig := signReport(t, r, "/api/reports", 8)
	if _, e := Decode(b, sig, "/api/reports"); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Put(r, b); e != nil {
		t.Fatal(e)
	}
}
