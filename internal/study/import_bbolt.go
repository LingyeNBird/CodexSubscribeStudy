package study

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

type ImportResult struct {
	Surveys         int
	Identities      int
	Batches         int
	AlreadyImported bool
}

func (s *Store) HasLegacyImport() (bool, error) {
	var exists bool
	err := s.db.View(func(tx *sql.Tx) error {
		return tx.QueryRow("SELECT EXISTS(SELECT 1 FROM legacy_imports)").Scan(&exists)
	})
	return exists, err
}

// ImportBbolt is a one-way, atomic import into an empty SQLite database.
// The legacy file is opened read-only and retained for rollback or archival.
func (s *Store) ImportBbolt(path string) (ImportResult, error) {
	legacy, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: true, Timeout: 3 * time.Second})
	if err != nil {
		return ImportResult{}, fmt.Errorf("open legacy database (stop its writer first): %w", err)
	}
	defer legacy.Close()
	file, err := os.Open(path)
	if err != nil {
		return ImportResult{}, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return ImportResult{}, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	result := ImportResult{}
	err = legacy.View(func(source *bolt.Tx) error {
		var integrityErr error
		for issue := range source.Check() {
			if integrityErr == nil {
				integrityErr = issue
			}
		}
		if integrityErr != nil {
			return fmt.Errorf("legacy integrity check: %w", integrityErr)
		}
		if err := source.ForEach(func(name []byte, bucket *bolt.Bucket) error {
			switch string(name) {
			case "evidence-identities", "evidence-batches", "survey-submissions", "survey-statistics", "survey-identities", "survey-meta":
				return nil
			}
			stats := bucket.Stats()
			if stats.KeyN == 0 && stats.BucketN == 1 && bucket.Sequence() == 0 {
				return nil
			}
			return fmt.Errorf("unsupported legacy bucket %q; source was not modified", name)
		}); err != nil {
			return err
		}
		if meta := source.Bucket([]byte("survey-statistics")); meta != nil {
			if raw := meta.Get([]byte("storage-version")); len(raw) > 0 {
				version, err := strconv.Atoi(string(raw))
				if err != nil || version < 1 || version > 4 {
					return errors.New("unsupported legacy survey storage version")
				}
			}
		}
		return s.db.Update(func(tx *sql.Tx) error {
			err := tx.QueryRow("SELECT survey_count,identity_count,batch_count FROM legacy_imports WHERE source_sha256=?", digest).Scan(&result.Surveys, &result.Identities, &result.Batches)
			if err == nil {
				result.AlreadyImported = true
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			var occupied int
			if err := tx.QueryRow(`SELECT (SELECT COUNT(*) FROM survey_submissions)+(SELECT COUNT(*) FROM evidence_identities)+(SELECT COUNT(*) FROM evidence_batches)+(SELECT COUNT(*) FROM legacy_imports)+COALESCE((SELECT seq FROM sqlite_sequence WHERE name='survey_submissions'),0)`).Scan(&occupied); err != nil {
				return err
			}
			if occupied != 0 {
				return errors.New("legacy import requires an empty SQLite database; refusing to merge or overwrite existing data")
			}
			if identities := source.Bucket([]byte("evidence-identities")); identities != nil {
				if err := identities.ForEach(func(key, body []byte) error {
					if len(key) != 32 || body == nil {
						return errors.New("invalid legacy reporter identity")
					}
					var state IdentityState
					if err := json.Unmarshal(body, &state); err != nil {
						return err
					}
					if _, err := tx.Exec("INSERT INTO evidence_identities(reporter_id,highest_revision) VALUES(?,?)", key, strconv.FormatUint(state.Highest, 10)); err != nil {
						return err
					}
					result.Identities++
					return nil
				}); err != nil {
					return err
				}
			}
			if batches := source.Bucket([]byte("evidence-batches")); batches != nil {
				if err := batches.ForEach(func(key, body []byte) error {
					if len(key) < 34 || key[32] != ':' || body == nil {
						return errors.New("invalid legacy batch key")
					}
					var item Stored
					if err := json.Unmarshal(body, &item); err != nil {
						return err
					}
					if item.Report.Summary == nil || !bytes.Equal(key[:32], reporterKey(item.Report.PublicKey)) || string(key[33:]) != item.Report.BatchID {
						return errors.New("legacy batch identity or summary mismatch")
					}
					var highestText string
					if err := tx.QueryRow("SELECT highest_revision FROM evidence_identities WHERE reporter_id=?", key[:32]).Scan(&highestText); err != nil {
						return err
					}
					highest, err := strconv.ParseUint(highestText, 10, 64)
					if err != nil {
						return err
					}
					if item.Report.Revision > highest {
						return errors.New("legacy batch exceeds reporter revision")
					}
					report, err := json.Marshal(item.Report)
					if err != nil {
						return err
					}
					if _, err := tx.Exec("INSERT INTO evidence_batches(reporter_id,batch_id,revision,digest,received_hour,report_json) VALUES(?,?,?,?,?,?)", key[:32], item.Report.BatchID, strconv.FormatUint(item.Report.Revision, 10), item.Digest, item.ReceivedHour, string(report)); err != nil {
						return err
					}
					result.Batches++
					return nil
				}); err != nil {
					return err
				}
			}
			aggregate := newSurveyAggregate()
			if surveys := source.Bucket([]byte("survey-submissions")); surveys != nil {
				if surveys.Sequence() > math.MaxInt64 {
					return errors.New("legacy survey sequence exceeds SQLite integer range")
				}
				if err := surveys.ForEach(func(key, body []byte) error {
					if len(key) != 8 || body == nil {
						return errors.New("invalid legacy survey key")
					}
					id := binary.BigEndian.Uint64(key)
					if id == 0 || id > surveys.Sequence() {
						return errors.New("legacy survey ID is outside its sequence")
					}
					upgraded, err := normalizeLegacySurvey(body)
					if err != nil {
						return err
					}
					var record storedSurveyRecord
					if err := json.Unmarshal(upgraded, &record); err != nil {
						return err
					}
					if err := aggregate.add(id, record); err != nil {
						return err
					}
					if err := insertSurveyRecord(tx, id, upgraded); err != nil {
						return err
					}
					result.Surveys++
					return nil
				}); err != nil {
					return err
				}
				changed, err := tx.Exec("UPDATE sqlite_sequence SET seq=? WHERE name='survey_submissions'", int64(surveys.Sequence()))
				if err != nil {
					return err
				}
				n, err := changed.RowsAffected()
				if err != nil {
					return err
				}
				if n == 0 {
					if _, err := tx.Exec("INSERT INTO sqlite_sequence(name,seq) VALUES('survey_submissions',?)", int64(surveys.Sequence())); err != nil {
						return err
					}
				}
			}
			aggregate.Range.ComputedAt = s.now().UTC()
			encoded, err := json.Marshal(aggregate)
			if err != nil {
				return err
			}
			if _, err := tx.Exec("INSERT INTO survey_statistics(id,payload) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", string(encoded)); err != nil {
				return err
			}
			_, err = tx.Exec("INSERT INTO legacy_imports(source_sha256,imported_at,survey_count,identity_count,batch_count) VALUES(?,?,?,?,?)", digest, s.now().UTC().Format(time.RFC3339Nano), result.Surveys, result.Identities, result.Batches)
			return err
		})
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, nil
}
