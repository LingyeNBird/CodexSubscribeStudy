package study

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	bolt "go.etcd.io/bbolt"
)

func createLegacySurveys(t *testing.T, version string, records map[uint64][]byte, sequence uint64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte("survey-submissions"))
		if err != nil {
			return err
		}
		for id, body := range records {
			var key [8]byte
			binary.BigEndian.PutUint64(key[:], id)
			if err := bucket.Put(key[:], body); err != nil {
				return err
			}
		}
		if err := bucket.SetSequence(sequence); err != nil {
			return err
		}
		meta, err := tx.CreateBucket([]byte("survey-statistics"))
		if err != nil {
			return err
		}
		if version != "" {
			if err := meta.Put([]byte("storage-version"), []byte(version)); err != nil {
				return err
			}
		}
		return meta.Put([]byte("aggregate"), []byte(`{"obsolete":true}`))
	})
	closeErr := db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return path
}

func TestImportBboltPreservesAllEvidenceAndHistory(t *testing.T) {
	old := []byte(`{"status":"降智并封号","answers":{"plans":["Plus"],"models":["GPT-5.6 Sora","GPT-5.6 Terra"],"official":["Codex CI"],"ciMode":["反代"],"usage":["反代"],"network":["宽带"],"quality":["优秀"]},"submittedAt":"2026-09-09T12:00:00+08:00","extra":"preserved"}`)
	unchanged := []byte(`{"status":["正常"],"answers":{"plans":["Free 免费"]},"details":{},"extra":"untouched"}`)
	source := createLegacySurveys(t, "2", map[uint64][]byte{2: old, 7: unchanged}, 42)
	report, body, _ := exampleReport(t)
	legacy, err := bolt.Open(source, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = legacy.Update(func(tx *bolt.Tx) error {
		identities, err := tx.CreateBucket([]byte("evidence-identities"))
		if err != nil {
			return err
		}
		state, _ := json.Marshal(IdentityState{Highest: math.MaxUint64})
		if err := identities.Put(reporterKey(report.PublicKey), state); err != nil {
			return err
		}
		batches, err := tx.CreateBucket([]byte("evidence-batches"))
		if err != nil {
			return err
		}
		stored, err := json.Marshal(Stored{Report: report, Digest: fmt.Sprintf("%x", sha256.Sum256(body)), ReceivedHour: 123456})
		if err != nil {
			return err
		}
		key := append(append(reporterKey(report.PublicKey), ':'), []byte(report.BatchID)...)
		return batches.Put(key, stored)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	beforeFile, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	store := openTest(t, 1)
	first, err := store.ImportBbolt(source)
	if err != nil {
		t.Fatal(err)
	}
	if first.Surveys != 2 || first.Identities != 1 || first.Batches != 1 || first.AlreadyImported {
		t.Fatal(first)
	}
	if err := store.db.Check(); err != nil {
		t.Fatal(err)
	}
	if err := store.db.View(func(tx *sql.Tx) error {
		current, err := surveyRows(tx)
		if err != nil {
			return err
		}
		if len(current) != 2 || !bytes.Equal(current[7], unchanged) {
			return fmt.Errorf("unchanged record or identity lost")
		}
		var record storedSurveyRecord
		if err := json.Unmarshal(current[2], &record); err != nil {
			return err
		}
		if !reflect.DeepEqual(record.Status, []string{"降智", "封号"}) || record.Answers["models"][0] != "GPT-5.6 Sol" || record.Answers["official"][0] != "Codex CLI" || record.Answers["cliMode"][0] != "反代" || record.Answers["network"][0] != "家宽" {
			return fmt.Errorf("historical answer upgrade failed")
		}
		if record.Answers["ciMode"] != nil || record.Answers["quality"] != nil || string(record.LegacyIPQuality) != `["优秀"]` || record.IPRisk != nil {
			return fmt.Errorf("legacy quality or renamed field changed meaning")
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(current[2], &raw); err != nil {
			return err
		}
		if string(raw["extra"]) != `"preserved"` || string(raw["submittedAt"]) != `"2026-09-09T12:00:00+08:00"` {
			return fmt.Errorf("extra fields or timestamp changed")
		}
		var indexed string
		if err := tx.QueryRow("SELECT submitted_at FROM survey_submissions WHERE id=2").Scan(&indexed); err != nil {
			return err
		}
		if indexed != "2026-09-09T04:00:00.000000000Z" {
			return fmt.Errorf("indexed UTC timestamp wrong")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	visits := 0
	if err := store.Walk(func(_ string, summary Summary, hour int64) error {
		visits++
		if hour != 123456 || !reflect.DeepEqual(summary, *report.Summary) {
			return fmt.Errorf("evidence changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if visits != 1 {
		t.Fatal("missing evidence")
	}
	duplicate, err := store.Put(report, body)
	if err != nil || !duplicate {
		t.Fatal("retry after import lost idempotence", err)
	}
	report.Revision++
	if _, err := store.Put(report, []byte("changed")); err != ErrConflict {
		t.Fatal("uint64 highest revision lost", err)
	}
	second, err := store.ImportBbolt(source)
	if err != nil || !second.AlreadyImported {
		t.Fatal("repeat import", err)
	}
	if err := store.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	stats, err := NewServer(store, fstest.MapFS{}).surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Range.LastSubmissionID != 43 || stats.Statuses[0] != 2 || stats.Statuses[3] != 1 || stats.Range.UnknownTimeCount != 1 {
		t.Fatal("sequence, counts, or unknown timestamps changed", stats.Range)
	}
	afterFile, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeFile, afterFile) {
		t.Fatal("legacy source was modified")
	}
}

func TestImportBboltRollbackAndNonemptyProtection(t *testing.T) {
	source := createLegacySurveys(t, "4", map[uint64][]byte{1: []byte(`{"status":["正常"],"answers":{"plans":["Plus"]}}`), 2: []byte("invalid")}, 2)
	store := openTest(t, 10)
	if _, err := store.ImportBbolt(source); err == nil {
		t.Fatal("invalid source accepted")
	}
	if err := store.db.View(func(tx *sql.Tx) error {
		for _, table := range []string{"survey_submissions", "survey_statistics", "legacy_imports", "evidence_identities", "evidence_batches"} {
			var count int
			if err := tx.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
				return err
			}
			if count != 0 {
				return fmt.Errorf("partial import in %s", table)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	valid := createLegacySurveys(t, "4", map[uint64][]byte{1: []byte(`{"status":["正常"],"answers":{"plans":["Plus"]}}`)}, 1)
	if _, err := store.ImportBbolt(valid); err != nil {
		t.Fatal("retry after failed import", err)
	}
	other := openTest(t, 10)
	if err := other.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := other.ImportBbolt(valid); err == nil {
		t.Fatal("nonempty target overwritten")
	}
	stats, err := NewServer(other, fstest.MapFS{}).surveyStatistics()
	if err != nil || stats.Statuses[0] != 1 {
		t.Fatal("existing target changed", err)
	}
}

func TestImportRetiredEmptyBucketsWithoutDiscardingUnknownData(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(fmt.Sprint(populated), func(t *testing.T) {
			source := createLegacySurveys(t, "4", map[uint64][]byte{}, 0)
			legacy, err := bolt.Open(source, 0600, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = legacy.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucket([]byte("reports-v1"))
				if err != nil {
					return err
				}
				if populated {
					return bucket.Put([]byte("record"), []byte("retained data"))
				}
				_, err = tx.CreateBucket([]byte("withdrawn-v1"))
				return err
			})
			closeErr := legacy.Close()
			if err != nil {
				t.Fatal(err)
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			store := openTest(t, 10)
			result, err := store.ImportBbolt(source)
			if populated {
				if err == nil {
					t.Fatal("unknown populated bucket was silently discarded")
				}
			} else if err != nil || result.Surveys != 0 {
				t.Fatal("retired empty buckets prevented import", err)
			}
		})
	}
}
