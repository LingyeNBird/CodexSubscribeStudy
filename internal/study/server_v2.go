package study

import (
	"errors"
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"io"
	"mime"
	"net/http"
	"time"
)

func (s *Server) resultV2() (ResultV2, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.cachedV2At) < 30*time.Second {
		return s.cachedV2, nil
	}
	value, err := s.store.AggregateV2()
	if err != nil {
		return ResultV2{}, err
	}
	s.cachedV2, s.cachedV2At = value, time.Now()
	return value, nil
}
func (s *Server) serveV2(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v2/protocol":
		if r.Method != "GET" {
			fail(w, 405, "method")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(protocol.MethodV2)
		return
	case "/api/v2/studies", "/api/v2/studies/" + StudyID:
		if r.Method != "GET" {
			fail(w, 405, "method")
			return
		}
		result, err := s.resultV2()
		if err != nil {
			fail(w, 503, "storage_unavailable")
			return
		}
		if r.URL.Path == "/api/v2/studies" {
			respond(w, 200, map[string]any{"studies": []ResultV2{result}})
		} else {
			respond(w, 200, result)
		}
		return
	case "/api/v2/reports", "/api/v2/withdraw":
		if r.Method != "POST" {
			fail(w, 405, "method")
			return
		}
		if !s.allow() {
			w.Header().Set("Retry-After", "120")
			fail(w, 429, "rate_limited")
			return
		}
		ctype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || ctype != "application/json" {
			fail(w, 415, "json_required")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyV2)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fail(w, 413, "body_too_large")
			return
		}
		report, err := DecodeV2(body, r.Header.Get("X-Study-Signature"), r.URL.Path)
		if err != nil {
			fail(w, 400, "invalid_submission")
			return
		}
		duplicate, err := s.store.PutV2(report, body, r.URL.Path == "/api/v2/withdraw")
		if errors.Is(err, ErrStale) || errors.Is(err, ErrConflict) {
			fail(w, 409, "revision_conflict")
			return
		}
		if errors.Is(err, ErrCapacity) {
			fail(w, 503, "contribution_capacity")
			return
		}
		if err != nil {
			fail(w, 503, "storage_unavailable")
			return
		}
		if !duplicate {
			s.mu.Lock()
			s.cachedAt = time.Time{}
			s.cachedV2At = time.Time{}
			s.mu.Unlock()
		}
		respond(w, 200, map[string]any{"accepted": true, "revision": report.Revision, "duplicate": duplicate})
		return
	}
	fail(w, 404, "not_found")
}
