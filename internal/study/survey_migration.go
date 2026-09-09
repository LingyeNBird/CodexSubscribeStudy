package study

import (
	"encoding/json"
	"errors"

	bolt "go.etcd.io/bbolt"
)

var surveyStorageVersionKey = []byte("storage-version")

const surveyStorageVersion = "4"

func migrateSurvey(tx *bolt.Tx) error {
	for _, name := range []string{"survey-identities", "survey-meta"} {
		if err := tx.DeleteBucket([]byte(name)); err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
	}
	bucket := tx.Bucket(surveyBucket)
	meta := tx.Bucket(surveyCacheBucket)
	if string(meta.Get(surveyStorageVersionKey)) == surveyStorageVersion {
		return nil
	}
	type update struct{ key, body []byte }
	var updates []update
	if err := bucket.ForEach(func(key, body []byte) error {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(body, &record); err != nil {
			return err
		}
		changed := false
		if raw := record["status"]; len(raw) > 0 && raw[0] == '"' {
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
			changed = true
		}
		var answers map[string]json.RawMessage
		if raw := record["answers"]; len(raw) > 0 {
			if err := json.Unmarshal(raw, &answers); err != nil {
				return err
			}
		}
		answersChanged := false
		if raw := answers["official"]; len(raw) > 0 {
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			for index, value := range values {
				if value == "Codex CI" {
					values[index] = "Codex CLI"
					answersChanged = true
				}
			}
			if answersChanged {
				answers["official"], _ = json.Marshal(values)
			}
		}
		if raw, exists := answers["ciMode"]; exists {
			answers["cliMode"] = raw
			delete(answers, "ciMode")
			answersChanged = true
		}
		if raw := answers["models"]; len(raw) > 0 {
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			for index, value := range values {
				if value == "GPT-5.6 Sora" {
					values[index] = "GPT-5.6 Sol"
					answersChanged = true
				}
			}
			if answersChanged {
				answers["models"], _ = json.Marshal(values)
			}
		}
		if raw := answers["network"]; len(raw) > 0 {
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			for index, value := range values {
				if value == "宽带" {
					values[index] = "家宽"
					answersChanged = true
				}
			}
			if answersChanged {
				answers["network"], _ = json.Marshal(values)
			}
		}
		if raw, exists := answers["quality"]; exists {
			record["legacyIPQuality"] = raw
			delete(answers, "quality")
			answersChanged = true
		}
		if answersChanged {
			record["answers"], _ = json.Marshal(answers)
			changed = true
		}
		if !changed {
			return nil
		}
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
	// Any migrated answer changes the inputs of a previously persisted aggregate.
	if err := meta.Delete(surveyCacheStateKey); err != nil {
		return err
	}
	return meta.Put(surveyStorageVersionKey, []byte(surveyStorageVersion))
}
