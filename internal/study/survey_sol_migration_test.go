package study

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestSurveySolMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	old := []byte(`{"status":["降智"],"answers":{"models":["GPT-5.6 Sora","GPT-5.6 Terra"]},"submittedAt":"2026-09-09T00:00:00Z","extra":"preserved"}`)
	unchanged := []byte(`{"status":["正常"],"answers":{}}`)
	if err := store.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(surveyBucket)
		if err := b.Put([]byte("old"), old); err != nil {
			return err
		}
		if err := b.Put([]byte("unchanged"), unchanged); err != nil {
			return err
		}
		if err := b.SetSequence(42); err != nil {
			return err
		}
		meta := tx.Bucket(surveyCacheBucket)
		if err := meta.Put(surveyCacheStateKey, []byte(`{"obsolete":true}`)); err != nil {
			return err
		}
		return meta.Put(surveyStorageVersionKey, []byte("2"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	var first []byte
	for attempt := 0; attempt < 2; attempt++ {
		store, err = Open(path, 10)
		if err != nil {
			t.Fatal(err)
		}
		err = store.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(surveyBucket)
			got := b.Get([]byte("old"))
			var actual, expected any
			if err := json.Unmarshal(got, &actual); err != nil {
				return err
			}
			if err := json.Unmarshal(bytes.ReplaceAll(old, []byte("GPT-5.6 Sora"), []byte("GPT-5.6 Sol")), &expected); err != nil {
				return err
			}
			actualJSON, _ := json.Marshal(actual)
			expectedJSON, _ := json.Marshal(expected)
			if !bytes.Equal(actualJSON, expectedJSON) {
				t.Errorf("migration changed unrelated fields: %s", got)
			}
			if !bytes.Equal(b.Get([]byte("unchanged")), unchanged) || b.Sequence() != 42 || b.Stats().KeyN != 2 {
				t.Error("migration changed unrelated records or sequence")
			}
			if tx.Bucket(surveyCacheBucket).Get(surveyCacheStateKey) != nil {
				t.Error("stale statistics survived migration")
			}
			if attempt == 0 {
				first = bytes.Clone(got)
			} else if !bytes.Equal(first, got) {
				t.Error("migration is not idempotent")
			}
			return nil
		})
		closeErr := store.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
}
