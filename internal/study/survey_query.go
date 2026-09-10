package study

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
)

type surveyPrerequisite struct {
	Factor  string   `json:"factor"`
	Options []string `json:"options"`
}
type surveyQuery struct {
	Prerequisites []surveyPrerequisite `json:"prerequisites"`
}

func validateSurveyQuery(query surveyQuery) error {
	if len(query.Prerequisites) == 0 || len(query.Prerequisites) > 12 {
		return errors.New("invalid prerequisite count")
	}
	seen := map[string]bool{}
	total := 0
	for _, condition := range query.Prerequisites {
		options, ok := surveyCatalog.Choices[condition.Factor]
		if !ok || seen[condition.Factor] || len(condition.Options) == 0 {
			return errors.New("invalid prerequisite")
		}
		seen[condition.Factor] = true
		total += len(condition.Options)
		if total > 32 {
			return errors.New("too many prerequisite options")
		}
		selected := map[string]bool{}
		for _, option := range condition.Options {
			if !slices.Contains(options, option) || selected[option] {
				return errors.New("invalid prerequisite option")
			}
			selected[option] = true
		}
	}
	return nil
}
func matchesPrerequisites(record storedSurveyRecord, conditions []surveyPrerequisite) bool {
	values := record.SurveySubmission.normalized()
	for _, condition := range conditions {
		matched := false
		for _, option := range condition.Options {
			if slices.Contains(values[condition.Factor], option) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
func (s *Server) conditionalSurveyStatistics(conditions []surveyPrerequisite) (surveySummary, error) {
	state := newSurveyAggregate()
	err := s.store.db.View(func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT id,payload FROM survey_submissions ORDER BY id")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uint64
			var body []byte
			if err := rows.Scan(&id, &body); err != nil {
				return err
			}
			var record storedSurveyRecord
			if err := json.Unmarshal(body, &record); err != nil {
				return err
			}
			if matchesPrerequisites(record, conditions) {
				if err := state.add(id, record); err != nil {
					return err
				}
			}
		}
		return rows.Err()
	})
	if err != nil {
		return surveySummary{}, err
	}
	state.Range.ComputedAt = s.store.now().UTC()
	return state.summary(), nil
}
func (s *Server) serveSurveyQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, "method")
		return
	}
	var query surveyQuery
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&query); err != nil || validateSurveyQuery(query) != nil {
		fail(w, http.StatusBadRequest, "invalid_query")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		fail(w, http.StatusBadRequest, "invalid_query")
		return
	}
	result, err := s.conditionalSurveyStatistics(query.Prerequisites)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "storage_unavailable")
		return
	}
	respond(w, http.StatusOK, result)
}
