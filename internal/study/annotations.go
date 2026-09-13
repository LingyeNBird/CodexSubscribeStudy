package study

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"
)

// Annotations belong to the administrator panel only. They never touch the
// submitted payload, so the statistics pipeline and the public endpoints cannot
// be affected by what an operator writes here.
const (
	annotationTagLength  = 24
	annotationTagLimit   = 12
	annotationNoteLength = 2000
	annotationBatchLimit = 500
	annotationBodyLimit  = 64 << 10
)

type surveyAnnotation struct {
	Tags []string `json:"tags"`
	Note string   `json:"note"`
}

type annotationsRequest struct {
	IDs        []uint64 `json:"ids"`
	AddTags    []string `json:"addTags"`
	RemoveTags []string `json:"removeTags"`
	Note       *string  `json:"note"`
}

func trimTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !slices.Contains(result, tag) {
			result = append(result, tag)
		}
	}
	return result
}

func (r *annotationsRequest) normalize() {
	r.IDs = slices.Compact(slices.Sorted(slices.Values(r.IDs)))
	r.AddTags = trimTags(r.AddTags)
	r.RemoveTags = trimTags(r.RemoveTags)
	r.RemoveTags = slices.DeleteFunc(r.RemoveTags, func(tag string) bool {
		return slices.Contains(r.AddTags, tag)
	})
	// Surrounding whitespace is not content, but nothing longer is dropped:
	// truncating here would lose text the caller cannot detect.
	if r.Note != nil {
		note := strings.TrimSpace(*r.Note)
		r.Note = &note
	}
}

func (r annotationsRequest) validate() error {
	switch {
	case len(r.IDs) == 0:
		return errors.New("at least one submission id is required")
	case len(r.IDs) > annotationBatchLimit:
		return errors.New("too many submissions in one request")
	case len(r.AddTags)+len(r.RemoveTags) > annotationTagLimit:
		return errors.New("too many tags in one request")
	case r.Note == nil && len(r.AddTags) == 0 && len(r.RemoveTags) == 0:
		return errors.New("no annotation change requested")
	}
	for _, id := range r.IDs {
		if id == 0 || id > math.MaxInt64 {
			return errors.New("invalid submission id")
		}
	}
	for _, tag := range append(slices.Clone(r.AddTags), r.RemoveTags...) {
		if utf8.RuneCountInString(tag) > annotationTagLength {
			return errors.New("tag is too long")
		}
	}
	if r.Note != nil && utf8.RuneCountInString(*r.Note) > annotationNoteLength {
		return errors.New("note is too long")
	}
	return nil
}

// applyTagChanges keeps the existing order so tags do not jump around.
func applyTagChanges(current, add, remove []string) []string {
	result := make([]string, 0, len(current)+len(add))
	for _, tag := range current {
		if !slices.Contains(remove, tag) && !slices.Contains(result, tag) {
			result = append(result, tag)
		}
	}
	for _, tag := range add {
		if !slices.Contains(remove, tag) && !slices.Contains(result, tag) {
			result = append(result, tag)
		}
	}
	return result
}

func (s *Store) surveyAnnotations(tx *sql.Tx) (map[uint64]surveyAnnotation, error) {
	result := map[uint64]surveyAnnotation{}
	rows, err := tx.Query("SELECT submission_id, tags, note FROM survey_annotations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id    int64
			raw   string
			entry surveyAnnotation
		)
		if err := rows.Scan(&id, &raw, &entry.Note); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &entry.Tags); err != nil {
			return nil, err
		}
		if entry.Tags == nil {
			entry.Tags = []string{}
		}
		result[uint64(id)] = entry
	}
	return result, rows.Err()
}

func (s *Store) annotate(request annotationsRequest) (int, error) {
	updated := 0
	err := s.db.Update(func(tx *sql.Tx) error {
		updated = 0
		stamp := s.now().UTC().Format("2006-01-02T15:04:05.000000000Z")
		for _, id := range request.IDs {
			var present int
			err := tx.QueryRow("SELECT 1 FROM survey_submissions WHERE id = ?", int64(id)).Scan(&present)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			current := surveyAnnotation{Tags: []string{}}
			var raw string
			err = tx.QueryRow("SELECT tags, note FROM survey_annotations WHERE submission_id = ?", int64(id)).Scan(&raw, &current.Note)
			if err == nil {
				if err := json.Unmarshal([]byte(raw), &current.Tags); err != nil {
					return err
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			tags := applyTagChanges(current.Tags, request.AddTags, request.RemoveTags)
			note := current.Note
			if request.Note != nil {
				note = *request.Note
			}
			if len(tags) == 0 && note == "" {
				// An emptied annotation is removed rather than kept as a blank row.
				if _, err := tx.Exec("DELETE FROM survey_annotations WHERE submission_id = ?", int64(id)); err != nil {
					return err
				}
				updated++
				continue
			}
			encoded, err := json.Marshal(tags)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO survey_annotations(submission_id,tags,note,updated_at) VALUES(?,?,?,?)
ON CONFLICT(submission_id) DO UPDATE SET tags=excluded.tags, note=excluded.note, updated_at=excluded.updated_at`,
				int64(id), string(encoded), note, stamp); err != nil {
				return err
			}
			updated++
		}
		return nil
	})
	return updated, err
}

func (s *Server) serveAdminAnnotations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	if !s.adminGranted(r) {
		fail(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, annotationBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request annotationsRequest
	if err := decoder.Decode(&request); err != nil {
		fail(w, http.StatusBadRequest, "invalid_annotation")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		fail(w, http.StatusBadRequest, "invalid_annotation")
		return
	}
	request.normalize()
	if err := request.validate(); err != nil {
		fail(w, http.StatusBadRequest, "invalid_annotation")
		return
	}
	updated, err := s.store.annotate(request)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "storage_unavailable")
		return
	}
	respond(w, http.StatusOK, map[string]int{"updated": updated})
}
