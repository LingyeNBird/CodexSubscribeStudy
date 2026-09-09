package study

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
)

type Server struct {
	store     *Store
	assets    fs.FS
	mu        sync.Mutex
	cached    Result
	cachedAt  time.Time
	tokens    float64
	lastToken time.Time
	sem       chan struct{}
}

func NewServer(store *Store, assets fs.FS) *Server {
	return &Server{store: store, assets: assets, tokens: 60, lastToken: time.Now(), sem: make(chan struct{}, 16)}
}

func (s *Server) result() (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.cachedAt) < 30*time.Second {
		return s.cached, nil
	}
	value, err := s.store.Aggregate()
	if err != nil {
		return Result{}, err
	}
	s.cached, s.cachedAt = value, time.Now()
	return value, nil
}

func (s *Server) allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.tokens += now.Sub(s.lastToken).Seconds() / 2
	if s.tokens > 60 {
		s.tokens = 60
	}
	s.lastToken = now
	if s.tokens < 1 {
		return false
	}
	s.tokens--
	return true
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status int, code string) {
	respond(w, status, map[string]string{"error": code})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	default:
		fail(w, http.StatusServiceUnavailable, "busy")
		return
	}

	switch r.URL.Path {
	case "/api/survey/submissions", "/api/survey/statistics":
		s.serveSurvey(w, r)
		return
	case "/healthz":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fail(w, http.StatusMethodNotAllowed, "method")
			return
		}
		respond(w, http.StatusOK, map[string]string{"status": "ok", "protocol": Protocol})
		return
	case "/api/protocol":
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(protocol.Method)
		return
	case "/api/studies", "/api/studies/" + StudyID:
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method")
			return
		}
		result, err := s.result()
		if err != nil {
			fail(w, http.StatusServiceUnavailable, "storage_unavailable")
			return
		}
		if r.URL.Path == "/api/studies" {
			respond(w, http.StatusOK, map[string]any{"studies": []Result{result}})
		} else {
			respond(w, http.StatusOK, result)
		}
		return
	case "/api/reports":
		if r.Method != http.MethodPost {
			fail(w, http.StatusMethodNotAllowed, "method")
			return
		}
		if !s.allow() {
			w.Header().Set("Retry-After", "120")
			fail(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
		contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || contentType != "application/json" {
			fail(w, http.StatusUnsupportedMediaType, "json_required")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fail(w, http.StatusRequestEntityTooLarge, "body_too_large")
			return
		}
		report, err := Decode(body, r.Header.Get("X-Study-Signature"), r.URL.Path)
		if err != nil {
			fail(w, http.StatusBadRequest, "invalid_submission")
			return
		}
		duplicate, err := s.store.Put(report, body)
		if errors.Is(err, ErrConflict) {
			fail(w, http.StatusConflict, "revision_conflict")
			return
		}
		if errors.Is(err, ErrCapacity) {
			fail(w, http.StatusServiceUnavailable, "contribution_capacity")
			return
		}
		if err != nil {
			fail(w, http.StatusServiceUnavailable, "storage_unavailable")
			return
		}
		if !duplicate {
			s.mu.Lock()
			s.cachedAt = time.Time{}
			s.mu.Unlock()
		}
		respond(w, http.StatusOK, map[string]any{"accepted": true, "revision": report.Revision, "duplicate": duplicate})
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	data, err := fs.ReadFile(s.assets, name)
	if err != nil {
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		name = "index.html"
		data, err = fs.ReadFile(s.assets, name)
	}
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "frontend_build_required")
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		switch path.Ext(name) {
		case ".woff2":
			contentType = "font/woff2"
		case ".woff":
			contentType = "font/woff"
		case ".ttf":
			contentType = "font/ttf"
		}
	}
	w.Header().Set("Content-Type", contentType)
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}
