package study

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LingyeNBird/CodexSubscribeStudy/internal/database"
)

var ErrConflict = errors.New("revision conflicts")
var ErrCapacity = errors.New("contribution capacity")

type Stored struct {
	Report       Report `json:"report"`
	Digest       string `json:"digest"`
	ReceivedHour int64  `json:"received_hour"`
}

type IdentityState struct {
	Highest uint64 `json:"highest"`
}

type Store struct {
	db           *database.DB
	maxReporters int
	now          func() time.Time
}

func Open(path string, maxReporters int) (*Store, error) {
	if maxReporters < 1 {
		return nil, errors.New("invalid capacity")
	}
	db, err := database.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, maxReporters: maxReporters, now: time.Now}, nil
}
func (s *Store) Close() error             { return s.db.Close() }
func (s *Store) Backup(path string) error { return s.db.Backup(path) }

func reporterKey(public string) []byte { digest := sha256.Sum256([]byte(public)); return digest[:] }

func (s *Store) Put(report Report, body []byte) (bool, error) {
	duplicate := false
	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	key := reporterKey(report.PublicKey)
	err := s.db.Update(func(tx *sql.Tx) error {
		var highestText string
		err := tx.QueryRow("SELECT highest_revision FROM evidence_identities WHERE reporter_id=?", key).Scan(&highestText)
		exists := err == nil
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var highest uint64
		if exists {
			highest, err = strconv.ParseUint(highestText, 10, 64)
			if err != nil {
				return err
			}
		}
		var existingRevision, existingDigest string
		err = tx.QueryRow("SELECT revision,digest FROM evidence_batches WHERE reporter_id=? AND batch_id=?", key, report.BatchID).Scan(&existingRevision, &existingDigest)
		batchExists := err == nil
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if batchExists && existingRevision == strconv.FormatUint(report.Revision, 10) && existingDigest == digest {
			duplicate = true
			return nil
		}
		if report.Revision <= highest {
			return ErrConflict
		}
		if !exists {
			var count int
			if err := tx.QueryRow("SELECT COUNT(*) FROM evidence_identities").Scan(&count); err != nil {
				return err
			}
			if count >= s.maxReporters {
				return ErrCapacity
			}
		}
		if !batchExists {
			var count int
			if err := tx.QueryRow("SELECT COUNT(*) FROM evidence_batches").Scan(&count); err != nil {
				return err
			}
			if count >= 100000 {
				return ErrCapacity
			}
		}
		encoded, err := json.Marshal(report)
		if err != nil {
			return err
		}
		revision := strconv.FormatUint(report.Revision, 10)
		if _, err := tx.Exec(`INSERT INTO evidence_identities(reporter_id,highest_revision) VALUES(?,?)
   ON CONFLICT(reporter_id) DO UPDATE SET highest_revision=excluded.highest_revision`, key, revision); err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO evidence_batches(reporter_id,batch_id,revision,digest,received_hour,report_json) VALUES(?,?,?,?,?,?)
   ON CONFLICT(reporter_id,batch_id) DO UPDATE SET revision=excluded.revision,digest=excluded.digest,received_hour=excluded.received_hour,report_json=excluded.report_json`,
			key, report.BatchID, revision, digest, s.now().UTC().Unix()/3600, string(encoded))
		return err
	})
	return duplicate, err
}

func (s *Store) Walk(visit func(string, Summary, int64) error) error {
	return s.db.View(func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT reporter_id,report_json,received_hour FROM evidence_batches ORDER BY reporter_id,batch_id")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var key, body []byte
			var received int64
			if err := rows.Scan(&key, &body, &received); err != nil {
				return err
			}
			var report Report
			if err := json.Unmarshal(body, &report); err != nil {
				return err
			}
			if report.Summary == nil {
				return errors.New("stored summary missing")
			}
			if err := visit(hex.EncodeToString(key), *report.Summary, received); err != nil {
				return err
			}
		}
		return rows.Err()
	})
}
