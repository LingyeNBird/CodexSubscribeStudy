package study

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	adminCookieName  = "study_admin"
	adminCookiePath  = "/api/admin"
	adminSessionTTL  = 2 * time.Hour
	adminSessionMax  = 32
	adminLoginWindow = time.Minute
	adminLoginLimit  = 10
	adminRecordLimit = 5000
	adminBodyLimit   = 4 << 10
)

// AdminConfig is the credential file for the read-only administrator panel. It
// lives outside version control and is mounted read-only in production.
type AdminConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoadAdminConfig reports (nil, nil) when the file is absent so that a missing
// configuration disables the panel instead of opening it with a default secret.
func LoadAdminConfig(path string) (*AdminConfig, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var config AdminConfig
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s contains trailing content", path)
	}
	config.Username = strings.TrimSpace(config.Username)
	if config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("%s needs a non-empty username and password", path)
	}
	return &config, nil
}

type adminAuth struct {
	config   AdminConfig
	mu       sync.Mutex
	sessions map[string]time.Time
	window   time.Time
	attempts int
}

func newAdminAuth(config AdminConfig) *adminAuth {
	return &adminAuth{config: config, sessions: map[string]time.Time{}}
}

// throttled limits password guessing without touching stored credentials.
func (a *adminAuth) throttled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if now := time.Now(); now.Sub(a.window) >= adminLoginWindow {
		a.window, a.attempts = now, 0
	}
	return a.attempts >= adminLoginLimit
}

// verify always evaluates both comparisons so that a wrong username and a wrong
// password cost the same work and return the same answer.
func (a *adminAuth) verify(username, password string) bool {
	user := subtle.ConstantTimeCompare([]byte(username), []byte(a.config.Username)) == 1
	secret := subtle.ConstantTimeCompare([]byte(password), []byte(a.config.Password)) == 1
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attempts++
	if user && secret {
		a.attempts = 0
	}
	return user && secret
}

func (a *adminAuth) open() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for key, expiry := range a.sessions {
		if now.After(expiry) {
			delete(a.sessions, key)
		}
	}
	for len(a.sessions) >= adminSessionMax {
		for key := range a.sessions {
			delete(a.sessions, key)
			break
		}
	}
	a.sessions[token] = now.Add(adminSessionTTL)
	return token, nil
}

// refresh validates a session token and slides its expiry forward.
func (a *adminAuth) refresh(token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	expiry, ok := a.sessions[token]
	if !ok {
		return false
	}
	now := time.Now()
	if now.After(expiry) {
		delete(a.sessions, token)
		return false
	}
	a.sessions[token] = now.Add(adminSessionTTL)
	return true
}

func (a *adminAuth) revoke(token string) {
	if token == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, token)
}

func (s *Server) adminToken(r *http.Request) string {
	cookie, err := r.Cookie(adminCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Server) adminGranted(r *http.Request) bool {
	return s.admin != nil && s.admin.refresh(s.adminToken(r))
}

// serveAdmin hides the whole panel while no credentials are configured and
// marks every response as non-indexable.
func (s *Server) serveAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if s.admin == nil {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	switch r.URL.Path {
	case "/api/admin/session":
		s.serveAdminSession(w, r)
	case "/api/admin/catalog":
		s.serveAdminCatalog(w, r)
	case "/api/admin/submissions":
		s.serveAdminSubmissions(w, r)
	case "/api/admin/annotations":
		s.serveAdminAnnotations(w, r)
	default:
		fail(w, http.StatusNotFound, "not_found")
	}
}

func (s *Server) serveAdminSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		respond(w, http.StatusOK, map[string]bool{"configured": true, "authenticated": s.adminGranted(r)})
	case http.MethodPost:
		if s.admin.throttled() {
			w.Header().Set("Retry-After", "60")
			fail(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, adminBodyLimit)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decoder.Decode(&credentials); err != nil {
			fail(w, http.StatusBadRequest, "invalid_credentials")
			return
		}
		if !s.admin.verify(credentials.Username, credentials.Password) {
			fail(w, http.StatusUnauthorized, "invalid_credentials")
			return
		}
		token, err := s.admin.open()
		if err != nil {
			fail(w, http.StatusServiceUnavailable, "session_unavailable")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     adminCookieName,
			Value:    token,
			Path:     adminCookiePath,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
			MaxAge:   int(adminSessionTTL / time.Second),
		})
		respond(w, http.StatusOK, map[string]bool{"authenticated": true})
	case http.MethodDelete:
		s.admin.revoke(s.adminToken(r))
		http.SetCookie(w, &http.Cookie{
			Name:     adminCookieName,
			Value:    "",
			Path:     adminCookiePath,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
			MaxAge:   -1,
		})
		respond(w, http.StatusOK, map[string]bool{"authenticated": false})
	default:
		fail(w, http.StatusMethodNotAllowed, "method")
	}
}

// serveAdminCatalog returns the questionnaire definition so the panel lists
// questions and options exactly as the statistics pipeline sees them.
func (s *Server) serveAdminCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	if !s.adminGranted(r) {
		fail(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(surveyCatalogJSON)
}

type adminSubmission struct {
	ID              uint64              `json:"id"`
	SubmittedAt     *time.Time          `json:"submittedAt"`
	Status          []string            `json:"status"`
	Answers         map[string][]string `json:"answers"`
	Details         map[string]string   `json:"details"`
	Raw             map[string][]string `json:"raw"`
	Normalized      map[string][]string `json:"normalized"`
	UsagePattern    []*int              `json:"usagePattern,omitempty"`
	IPRisk          *int                `json:"ipRisk,omitempty"`
	LegacyIPQuality json.RawMessage     `json:"legacyIPQuality,omitempty"`
	Tags            []string            `json:"tags"`
	Note            string              `json:"note"`
}

// rawAnswers reports what a respondent actually submitted, before the bucketing
// that statistics needs. The panel shows these values, because merging "SG" and
// "TR" into "其他国家或地区" loses the detail an administrator is looking for.
func (q SurveySubmission) rawAnswers() map[string][]string {
	result := make(map[string][]string, len(q.Answers)+6)
	for key, value := range q.Answers {
		result[key] = value
	}
	if q.IPRisk != nil {
		result["ipRisk"] = []string{strconv.Itoa(*q.IPRisk)}
	}
	for _, key := range []string{"country", "exitCountry"} {
		if code := q.Details[key]; code != "" {
			result[key] = []string{code}
		}
	}
	if raw := q.Details["duration"]; raw != "" {
		if unit := q.Details["durationUnit"]; unit != "" {
			raw += " " + unit
		}
		result["duration"] = []string{raw}
	}
	if raw := q.Details["people"]; raw != "" {
		result["people"] = []string{raw}
	}
	if raw := q.Details["concurrency"]; raw != "" {
		result["concurrency"] = []string{raw}
	}
	for key, value := range q.Details {
		if strings.HasPrefix(key, "toolMode:") && !slices.Contains(result["thirdMode"], value) {
			result["thirdMode"] = append(result["thirdMode"], value)
		}
	}
	return result
}

type adminSubmissions struct {
	Total     int               `json:"total"`
	Returned  int               `json:"returned"`
	Truncated bool              `json:"truncated"`
	Records   []adminSubmission `json:"records"`
}

var surveyTimeLayouts = []string{"2006-01-02T15:04:05.000000000Z", time.RFC3339Nano, time.RFC3339}

func parseSurveyTime(value string) *time.Time {
	for _, layout := range surveyTimeLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}

// listSurveyRecords returns the newest submissions first, capped so that a
// large database cannot turn the admin view into an unbounded response.
func (s *Store) listSurveyRecords(limit int) (adminSubmissions, error) {
	result := adminSubmissions{Records: []adminSubmission{}}
	err := s.db.View(func(tx *sql.Tx) error {
		annotations, err := s.surveyAnnotations(tx)
		if err != nil {
			return err
		}
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&result.Total); err != nil {
			return err
		}
		rows, err := tx.Query("SELECT id, submitted_at, payload FROM survey_submissions ORDER BY id DESC LIMIT ?", limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var (
				id      int64
				stamp   sql.NullString
				payload string
			)
			if err := rows.Scan(&id, &stamp, &payload); err != nil {
				return err
			}
			var record storedSurveyRecord
			if err := json.Unmarshal([]byte(payload), &record); err != nil {
				return err
			}
			entry := adminSubmission{
				ID:              uint64(id),
				SubmittedAt:     record.SubmittedAt,
				Status:          record.Status,
				Answers:         record.Answers,
				Details:         record.Details,
				Raw:             record.SurveySubmission.rawAnswers(),
				Normalized:      record.SurveySubmission.normalized(),
				UsagePattern:    record.UsagePattern,
				IPRisk:          record.IPRisk,
				LegacyIPQuality: record.LegacyIPQuality,
				Tags:            []string{},
			}
			if stamp.Valid {
				if parsed := parseSurveyTime(stamp.String); parsed != nil {
					entry.SubmittedAt = parsed
				}
			}
			if entry.Status == nil {
				entry.Status = []string{}
			}
			if entry.Answers == nil {
				entry.Answers = map[string][]string{}
			}
			if entry.Details == nil {
				entry.Details = map[string]string{}
			}
			if annotation, ok := annotations[entry.ID]; ok {
				entry.Tags, entry.Note = annotation.Tags, annotation.Note
			}
			result.Records = append(result.Records, entry)
		}
		return rows.Err()
	})
	if err != nil {
		return adminSubmissions{}, err
	}
	result.Returned = len(result.Records)
	result.Truncated = result.Total > result.Returned
	return result, nil
}

func (s *Server) serveAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	if !s.adminGranted(r) {
		fail(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	result, err := s.store.listSurveyRecords(adminRecordLimit)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "storage_unavailable")
		return
	}
	respond(w, http.StatusOK, result)
}
