package study

import (
	"bytes"
	"encoding/json"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSurveyCLIMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "study.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	old := []byte(`{"status":["正常"],"answers":{"plans":["Plus"],"official":["Codex CI","Web 网页"],"ciMode":["反代"]},"submittedAt":"2026-09-09T00:00:00Z"}`)
	if err := store.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(surveyBucket)
		if err := b.Put([]byte("record"), old); err != nil {
			return err
		}
		if err := b.SetSequence(17); err != nil {
			return err
		}
		meta := tx.Bucket(surveyCacheBucket)
		if err := meta.Put(surveyCacheStateKey, []byte(`{"obsolete":true}`)); err != nil {
			return err
		}
		return meta.Put(surveyStorageVersionKey, []byte("3"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	var first []byte
	for attempt := range 2 {
		store, err = Open(path, 10)
		if err != nil {
			t.Fatal(err)
		}
		err = store.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(surveyBucket)
			actual := b.Get([]byte("record"))
			expected := bytes.ReplaceAll(bytes.ReplaceAll(old, []byte("Codex CI"), []byte("Codex CLI")), []byte("ciMode"), []byte("cliMode"))
			var decoded, wanted any
			if err := json.Unmarshal(actual, &decoded); err != nil {
				return err
			}
			if err := json.Unmarshal(expected, &wanted); err != nil {
				return err
			}
			var q SurveySubmission
			if err := json.Unmarshal(actual, &q); err != nil {
				return err
			}
			if err := validateSurvey(q); err != nil {
				t.Error("migrated record rejected", err)
			}
			if !reflect.DeepEqual(decoded, wanted) {
				t.Fatalf("unexpected migrated record: %s", actual)
			}
			if b.Sequence() != 17 || b.Stats().KeyN != 1 {
				t.Error("record identity changed")
			}
			if tx.Bucket(surveyCacheBucket).Get(surveyCacheStateKey) != nil {
				t.Error("old cache survived")
			}
			if attempt == 0 {
				first = bytes.Clone(actual)
			} else if !bytes.Equal(first, actual) {
				t.Error("migration not idempotent")
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
