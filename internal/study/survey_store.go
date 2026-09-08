package study

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	bolt "go.etcd.io/bbolt"
)

var surveyBucket = []byte("survey-submissions")

func initSurvey(tx *bolt.Tx) error {
	// Remove obsolete advisory identifiers without touching questionnaire records.
	for _, name := range []string{"survey-identities", "survey-meta"} {
		if err := tx.DeleteBucket([]byte(name)); err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
	}
	bucket, err := tx.CreateBucketIfNotExists(surveyBucket)
	if err != nil {
		return err
	}
	type update struct{ key, body []byte }
	var updates []update
	if err := bucket.ForEach(func(key, body []byte) error {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(body, &record); err != nil {
			return err
		}
		raw := record["status"]
		if len(raw) == 0 || raw[0] != '"' {
			return nil
		}
		var old string
		if err := json.Unmarshal(raw, &old); err != nil {
			return err
		}
		var statuses []string
		switch old {
		case "正常", "降智", "封号":
			statuses = []string{old}
		case "降智并封号":
			statuses = []string{"降智", "封号"}
		default:
			return errors.New("invalid stored survey status")
		}
		record["status"], _ = json.Marshal(statuses)
		encoded, err := json.Marshal(record)
		if err != nil {
			return err
		}
		updates = append(updates, update{append([]byte(nil), key...), encoded})
		return nil
	}); err != nil {
		return err
	}
	for _, item := range updates {
		if err := bucket.Put(item.key, item.body); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) putSurvey(submission SurveySubmission) error {
	body, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		if bucket.Stats().KeyN >= 100000 {
			return ErrCapacity
		}
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], id)
		return bucket.Put(key[:], body)
	})
}
