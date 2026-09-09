package study

import (
	"encoding/binary"
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var surveyBucket = []byte("survey-submissions")
var surveyCacheBucket = []byte("survey-statistics")

type storedSurveyRecord struct {
	SurveySubmission
	SubmittedAt     *time.Time      `json:"submittedAt,omitempty"`
	LegacyIPQuality json.RawMessage `json:"legacyIPQuality,omitempty"`
}

func initSurvey(tx *bolt.Tx) error {
	for _, name := range [][]byte{surveyBucket, surveyCacheBucket} {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return err
		}
	}
	return migrateSurvey(tx)
}

func (s *Store) putSurvey(submission SurveySubmission) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		if bucket.Stats().KeyN >= 100000 {
			return ErrCapacity
		}
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		submittedAt := s.now().UTC()
		body, err := json.Marshal(storedSurveyRecord{SurveySubmission: submission, SubmittedAt: &submittedAt})
		if err != nil {
			return err
		}
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], id)
		return bucket.Put(key[:], body)
	})
}
