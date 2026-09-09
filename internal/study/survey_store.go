package study

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"time"
)

type storedSurveyRecord struct {
	SurveySubmission
	SubmittedAt     *time.Time      `json:"submittedAt,omitempty"`
	LegacyIPQuality json.RawMessage `json:"legacyIPQuality,omitempty"`
}

func surveyStatusMask(q SurveySubmission) int {
	mask := 0
	if q.degraded() {
		mask |= 1
	}
	if q.banned() {
		mask |= 2
	}
	if q.limited() {
		mask |= 4
	}
	return mask
}

func insertSurveyRecord(tx *sql.Tx, id uint64, body []byte) error {
	if id > math.MaxInt64 {
		return errors.New("survey ID exceeds SQLite integer range")
	}
	var record storedSurveyRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return err
	}
	var submittedAt any
	if record.SubmittedAt != nil {
		submittedAt = record.SubmittedAt.UTC().Format("2006-01-02T15:04:05.000000000Z")
	}
	var err error
	if id == 0 {
		_, err = tx.Exec("INSERT INTO survey_submissions(submitted_at,status_mask,payload) VALUES(?,?,?)", submittedAt, surveyStatusMask(record.SurveySubmission), string(body))
	} else {
		_, err = tx.Exec("INSERT INTO survey_submissions(id,submitted_at,status_mask,payload) VALUES(?,?,?,?)", int64(id), submittedAt, surveyStatusMask(record.SurveySubmission), string(body))
	}
	return err
}

func (s *Store) putSurvey(submission SurveySubmission) error {
	submittedAt := s.now().UTC()
	body, err := json.Marshal(storedSurveyRecord{SurveySubmission: submission, SubmittedAt: &submittedAt})
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&count); err != nil {
			return err
		}
		if count >= 100000 {
			return ErrCapacity
		}
		return insertSurveyRecord(tx, 0, body)
	})
}
