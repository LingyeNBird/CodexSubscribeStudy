package study

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

// withFakeFeedbackUpstream points feedbackURL at a local server for the
// duration of the test, recording every accepted call.
func withFakeFeedbackUpstream(t *testing.T, status int) *int32 {
	t.Helper()
	var calls int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["project"] != feedbackProject || body["content"] == "" || !feedbackClientIDPattern.MatchString(body["clientId"]) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(upstream.Close)
	original := feedbackURL
	feedbackURL = upstream.URL
	t.Cleanup(func() { feedbackURL = original })
	return &calls
}

func feedbackCall(s *Server, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	s.ServeHTTP(response, request)
	return response
}

func TestFeedbackSendsImmediatelyWhenCooldownHasElapsed(t *testing.T) {
	calls := withFakeFeedbackUpstream(t, http.StatusAccepted)
	server := NewServer(openTest(t, 10), fstest.MapFS{})

	response := feedbackCall(server, `{"content":"看起来不错","clientId":"abcdefghijklmnopqrstuvwx"}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body)
	}
	var payload struct {
		Queued bool `json:"queued"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Queued {
		t.Fatal("first submission should send immediately, not queue")
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("expected exactly one upstream call, got %d", got)
	}
}

func TestFeedbackQueuesWhenSubmittedAgainWithinCooldown(t *testing.T) {
	calls := withFakeFeedbackUpstream(t, http.StatusAccepted)
	server := NewServer(openTest(t, 10), fstest.MapFS{})

	first := feedbackCall(server, `{"content":"第一条","clientId":"abcdefghijklmnopqrstuvwx"}`)
	if first.Code != http.StatusAccepted {
		t.Fatalf("first submission rejected: %d", first.Code)
	}

	second := feedbackCall(server, `{"content":"第二条","clientId":"abcdefghijklmnopqrstuvwx"}`)
	if second.Code != http.StatusAccepted {
		t.Fatalf("second submission rejected: %d", second.Code)
	}
	var payload struct {
		Queued bool `json:"queued"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Queued {
		t.Fatal("second submission within the cooldown should be queued, not sent immediately")
	}
	// The queued item must not reach the upstream before the cooldown ends.
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("queued item was sent before the cooldown elapsed: %d calls", got)
	}
}

func TestFeedbackRejectsOversizedContent(t *testing.T) {
	withFakeFeedbackUpstream(t, http.StatusAccepted)
	server := NewServer(openTest(t, 10), fstest.MapFS{})

	oversized := strings.Repeat("字", feedbackContentLimit+1)
	body, err := json.Marshal(map[string]string{"content": oversized, "clientId": "abcdefghijklmnopqrstuvwx"})
	if err != nil {
		t.Fatal(err)
	}
	response := feedbackCall(server, string(body))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized content should be rejected: %d", response.Code)
	}
}

func TestFeedbackRejectsEmptyContent(t *testing.T) {
	withFakeFeedbackUpstream(t, http.StatusAccepted)
	server := NewServer(openTest(t, 10), fstest.MapFS{})

	response := feedbackCall(server, `{"content":"   ","clientId":"abcdefghijklmnopqrstuvwx"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("blank content should be rejected: %d", response.Code)
	}
}

func TestFeedbackReplacesMalformedClientID(t *testing.T) {
	var seenClientID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		seenClientID = body["clientId"]
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()
	original := feedbackURL
	feedbackURL = upstream.URL
	defer func() { feedbackURL = original }()

	server := NewServer(openTest(t, 10), fstest.MapFS{})
	response := feedbackCall(server, `{"content":"你好","clientId":"not valid!"}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("submission should still succeed: %d", response.Code)
	}
	if !feedbackClientIDPattern.MatchString(seenClientID) {
		t.Fatalf("malformed client id was forwarded to upstream: %q", seenClientID)
	}
}

func TestFeedbackRetriesQueuedItemAfterUpstreamRejection(t *testing.T) {
	original := feedbackCooldown
	feedbackCooldown = 30 * time.Millisecond
	t.Cleanup(func() { feedbackCooldown = original })

	var calls int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 2 {
			// The queued item's first retry is rejected, simulating the
			// upstream still enforcing its window past our own tracking
			// (this is exactly what was observed against the real API).
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()
	originalURL := feedbackURL
	feedbackURL = upstream.URL
	defer func() { feedbackURL = originalURL }()

	server := NewServer(openTest(t, 10), fstest.MapFS{})
	if response := feedbackCall(server, `{"content":"第一条","clientId":"abcdefghijklmnopqrstuvwx"}`); response.Code != http.StatusAccepted {
		t.Fatalf("first submission rejected: %d", response.Code)
	}
	if response := feedbackCall(server, `{"content":"第二条","clientId":"abcdefghijklmnopqrstuvwx"}`); response.Code != http.StatusAccepted {
		t.Fatalf("second submission rejected: %d", response.Code)
	}

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&calls) < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected the rejected item to be retried and delivered (3 upstream calls), got %d", got)
	}
}

func TestFeedbackGivesUpAfterMaxAttempts(t *testing.T) {
	original := feedbackCooldown
	feedbackCooldown = 10 * time.Millisecond
	t.Cleanup(func() { feedbackCooldown = original })
	calls := withFakeFeedbackUpstream(t, http.StatusTooManyRequests)

	server := NewServer(openTest(t, 10), fstest.MapFS{})
	// Consumes the immediate slot; its failure is reported straight back and
	// is not part of what this test is checking.
	feedbackCall(server, `{"content":"占位","clientId":"abcdefghijklmnopqrstuvwx"}`)
	if response := feedbackCall(server, `{"content":"重试内容","clientId":"abcdefghijklmnopqrstuvwx"}`); response.Code != http.StatusAccepted {
		t.Fatalf("queued submission should still be accepted up front: %d", response.Code)
	}

	expected := int32(1 + feedbackMaxAttempts)
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(calls) < expected && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(calls); got != expected {
		t.Fatalf("expected %d total upstream calls (1 immediate + %d retries), got %d", expected, feedbackMaxAttempts, got)
	}
	// Give the worker long enough that it would have retried again had it
	// not correctly given up after the attempt cap.
	time.Sleep(10 * feedbackCooldown)
	if got := atomic.LoadInt32(calls); got != expected {
		t.Fatalf("worker kept retrying past the attempt cap: %d calls", got)
	}
}

func TestFeedbackQueueFullIsRejected(t *testing.T) {
	withFakeFeedbackUpstream(t, http.StatusAccepted)
	server := NewServer(openTest(t, 10), fstest.MapFS{})

	// Consume the immediate slot, then fill the queue to its cap.
	if response := feedbackCall(server, `{"content":"占用槽位","clientId":"abcdefghijklmnopqrstuvwx"}`); response.Code != http.StatusAccepted {
		t.Fatalf("setup submission failed: %d", response.Code)
	}
	var last *httptest.ResponseRecorder
	for i := 0; i < feedbackQueueMax+1; i++ {
		last = feedbackCall(server, `{"content":"排队内容","clientId":"abcdefghijklmnopqrstuvwx"}`)
	}
	if last.Code != http.StatusServiceUnavailable {
		t.Fatalf("submission beyond queue capacity should be rejected: %d", last.Code)
	}
}
