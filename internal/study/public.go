package study

import (
	"database/sql"
	"net/http"
)

// The public pivot panel mirrors the administrator panel's data feed for
// visitors who are not signed in. It is safe to publish because the study
// never collects usernames, emails, or other identifying fields; the only
// thing withheld here is the investigator's own tags and notes, which are
// working annotations rather than survey data.

type publicSubmission struct {
	surveyRecordCore
}

type publicSubmissions struct {
	Total     int                `json:"total"`
	Returned  int                `json:"returned"`
	Truncated bool               `json:"truncated"`
	Records   []publicSubmission `json:"records"`
}

// listPublicSurveyRecords is listSurveyRecords without the annotations join,
// so investigator tags and notes never enter the response at all.
func (s *Store) listPublicSurveyRecords(limit int) (publicSubmissions, error) {
	result := publicSubmissions{Records: []publicSubmission{}}
	err := s.db.View(func(tx *sql.Tx) error {
		cores, total, err := scanSurveyRecordCores(tx, limit)
		if err != nil {
			return err
		}
		result.Total = total
		for _, core := range cores {
			result.Records = append(result.Records, publicSubmission{surveyRecordCore: core})
		}
		return nil
	})
	if err != nil {
		return publicSubmissions{}, err
	}
	result.Returned = len(result.Records)
	result.Truncated = result.Total > result.Returned
	return result, nil
}

// servePivot serves the public, unauthenticated pivot-analysis data feed. It
// is deliberately kept out of search indexes: the data is public to anyone
// with the link, but there is no reason to have it crawled.
func (s *Server) servePivot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if !s.allowPivot() {
		w.Header().Set("Retry-After", "5")
		fail(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	switch r.URL.Path {
	case "/api/pivot/catalog":
		s.servePivotCatalog(w, r)
	case "/api/pivot/submissions":
		s.servePivotSubmissions(w, r)
	default:
		fail(w, http.StatusNotFound, "not_found")
	}
}

func (s *Server) servePivotCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(surveyCatalogJSON)
}

func (s *Server) servePivotSubmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	result, err := s.store.listPublicSurveyRecords(surveyRecordLimit)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "storage_unavailable")
		return
	}
	respond(w, http.StatusOK, result)
}
