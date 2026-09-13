package study

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	feedbackProject = "codex-subscribe-study"

	// feedbackContentLimit is this app's own UX limit. The upstream API's hard
	// cap is 1000 characters; 800 leaves headroom rather than chasing it.
	feedbackContentLimit = 800
	feedbackBodyLimit    = 4 << 10

	// feedbackQueueMax bounds memory under a burst or a scripted abuser; once
	// full, further submissions are rejected rather than queued indefinitely,
	// so a backlog cannot delay every later visitor's feedback by hours.
	feedbackQueueMax = 50
	// A queued send can still be rejected even after the locally-tracked
	// cooldown has elapsed — observed in practice as a 429 from the upstream
	// on the very first retry, likely because its window is not perfectly
	// aligned with a flat "5 minutes since our last attempt". Retrying rather
	// than dropping the item is what makes the queue actually deliver
	// feedback instead of silently losing it; the cap keeps a permanently
	// failing upstream from wedging the queue forever.
	feedbackMaxAttempts = 5
)

// feedbackURL is a var, not a const, so tests can redirect it to a local
// httptest server instead of calling the real endpoint.
var feedbackURL = "https://feedbackbash.nightunderfly.online/api/v1/feedback"

// feedbackCooldown is a var, not a const, so tests can shrink it instead of
// waiting on the real five minutes. The upstream API allows at most one
// submission every five minutes per source IP; since every visitor's
// feedback is relayed from this server's own outbound IP, that quota is
// shared across all of them, not personal to one visitor — so it is enforced
// here as a single global cooldown with a queue for whatever arrives while
// it is in effect.
var feedbackCooldown = 5 * time.Minute

var feedbackClientIDPattern = regexp.MustCompile(`^[a-z0-9]{16,32}$`)

const feedbackClientIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// randomFeedbackClientID stands in whenever the browser's own id is missing
// or malformed. It only needs to satisfy the upstream's format, not be
// unpredictable, since it is nothing more than a grouping key.
func randomFeedbackClientID() string {
	const length = 24
	raw := make([]byte, length)
	_, _ = rand.Read(raw)
	id := make([]byte, length)
	for i, b := range raw {
		id[i] = feedbackClientIDAlphabet[int(b)%len(feedbackClientIDAlphabet)]
	}
	return string(id)
}

type pendingFeedback struct {
	content  string
	clientID string
	attempts int
}

// feedbackQueue serializes outbound submissions to the upstream API so that
// this server, as a single shared caller, never exceeds its rate limit
// regardless of how many visitors submit feedback concurrently.
type feedbackQueue struct {
	mu       sync.Mutex
	lastSent time.Time
	pending  []pendingFeedback
	wake     chan struct{}
	client   *http.Client
}

func newFeedbackQueue() *feedbackQueue {
	q := &feedbackQueue{
		wake:   make(chan struct{}, 1),
		client: &http.Client{Timeout: 10 * time.Second},
	}
	go q.run()
	return q
}

// submit decides whether an item can go out immediately (the shared cooldown
// has elapsed and nothing is already waiting) or must be queued for the
// background worker. It reports false only when the queue is already full.
func (q *feedbackQueue) submit(item pendingFeedback) (accepted, queued bool) {
	q.mu.Lock()
	if len(q.pending) == 0 && time.Since(q.lastSent) >= feedbackCooldown {
		q.lastSent = time.Now()
		q.mu.Unlock()
		return q.send(item) == nil, false
	}
	if len(q.pending) >= feedbackQueueMax {
		q.mu.Unlock()
		return false, false
	}
	q.pending = append(q.pending, item)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return true, true
}

func (q *feedbackQueue) run() {
	for {
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.mu.Unlock()
			<-q.wake
			continue
		}
		wait := feedbackCooldown - time.Since(q.lastSent)
		q.mu.Unlock()
		if wait > 0 {
			time.Sleep(wait)
		}
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.mu.Unlock()
			continue
		}
		item := q.pending[0]
		q.pending = q.pending[1:]
		q.lastSent = time.Now()
		item.attempts++
		q.mu.Unlock()
		if err := q.send(item); err != nil {
			if item.attempts < feedbackMaxAttempts {
				log.Printf("feedback: queued submission failed (attempt %d/%d), will retry: %v", item.attempts, feedbackMaxAttempts, err)
				q.mu.Lock()
				q.pending = append([]pendingFeedback{item}, q.pending...)
				q.mu.Unlock()
			} else {
				// Best-effort relay: the visitor was already told this
				// succeeded, so there is no one left to report a final
				// failure to beyond the server's own log.
				log.Printf("feedback: giving up on queued submission after %d attempts: %v", item.attempts, err)
			}
		}
	}
}

func (q *feedbackQueue) send(item pendingFeedback) error {
	payload, err := json.Marshal(map[string]string{
		"project":  feedbackProject,
		"content":  item.content,
		"clientId": item.clientID,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, feedbackURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := q.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusAccepted {
		return &feedbackUpstreamError{status: response.StatusCode}
	}
	return nil
}

type feedbackUpstreamError struct{ status int }

func (e *feedbackUpstreamError) Error() string {
	return "unexpected upstream status " + http.StatusText(e.status)
}

type feedbackRequest struct {
	Content  string `json:"content"`
	ClientID string `json:"clientId"`
}

func (s *Server) serveFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, feedbackBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req feedbackRequest
	if err := decoder.Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "invalid_feedback")
		return
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		fail(w, http.StatusBadRequest, "invalid_feedback")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" || utf8.RuneCountInString(content) > feedbackContentLimit {
		fail(w, http.StatusBadRequest, "invalid_feedback")
		return
	}
	clientID := req.ClientID
	if !feedbackClientIDPattern.MatchString(clientID) {
		clientID = randomFeedbackClientID()
	}
	accepted, queued := s.feedback.submit(pendingFeedback{content: content, clientID: clientID})
	if !accepted {
		fail(w, http.StatusServiceUnavailable, "feedback_unavailable")
		return
	}
	respond(w, http.StatusAccepted, map[string]bool{"queued": queued})
}
