package study

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	bolt "go.etcd.io/bbolt"
)

func TestSurveyCountsMigrationPreservesRawRecords(t *testing.T) {
	for _, size := range []int{0, 9} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "survey.db")
			store, err := Open(path, 10)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { store.Close() }()
			for i := 0; i < size; i++ {
				if err := store.putSurvey(cacheQuestion(i)); err != nil {
					t.Fatal(err)
				}
			}
			server := NewServer(store, fstest.MapFS{})
			before, err := server.surveyStatistics()
			if err != nil {
				t.Fatal(err)
			}
			original := map[string][]byte{}
			var sequence uint64
			if err := store.db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(surveyBucket)
				sequence = bucket.Sequence()
				if err := bucket.ForEach(func(key, value []byte) error {
					original[string(key)] = bytes.Clone(value)
					return nil
				}); err != nil {
					return err
				}
				meta := tx.Bucket(surveyCacheBucket)
				var legacy map[string]json.RawMessage
				if err := json.Unmarshal(meta.Get(surveyCacheStateKey), &legacy); err != nil {
					return err
				}
				legacy["version"] = json.RawMessage(`1`)
				legacy["result"] = json.RawMessage(`{"total":999999,"associations":[]}`)
				raw, err := json.Marshal(legacy)
				if err != nil {
					return err
				}
				return meta.Put(surveyCacheStateKey, raw)
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = Open(path, 10)
			if err != nil {
				t.Fatal(err)
			}
			server = NewServer(store, fstest.MapFS{})
			after, err := server.surveyStatistics()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("upgrade changed counts, timestamps, range, or metadata")
			}
			if err := store.db.View(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(surveyBucket)
				if bucket.Sequence() != sequence || bucket.Stats().KeyN != len(original) {
					return fmt.Errorf("raw record sequence or size changed")
				}
				for key, value := range original {
					if !bytes.Equal(bucket.Get([]byte(key)), value) {
						return fmt.Errorf("raw record bytes changed")
					}
				}
				var cache map[string]json.RawMessage
				if err := json.Unmarshal(tx.Bucket(surveyCacheBucket).Get(surveyCacheStateKey), &cache); err != nil {
					return err
				}
				if string(cache["version"]) != "2" || cache["result"] != nil {
					return fmt.Errorf("obsolete derived cache was not migrated")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.putSurvey(cacheQuestion(7)); err != nil {
				t.Fatal(err)
			}
			appended, err := server.surveyStatistics()
			if err != nil {
				t.Fatal(err)
			}
			expected := before.Statuses
			expected[7]++
			if appended.Statuses != expected || appended.Range.LastSubmissionID != sequence+1 {
				t.Fatal("append after migration lost or duplicated history")
			}
			if err := store.db.View(func(tx *bolt.Tx) error {
				for key, value := range original {
					if !bytes.Equal(tx.Bucket(surveyBucket).Get([]byte(key)), value) {
						return fmt.Errorf("incremental refresh changed historical bytes")
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
