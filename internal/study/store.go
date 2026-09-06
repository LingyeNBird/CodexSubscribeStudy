package study

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var ErrStale = errors.New("stale revision")
var ErrConflict = errors.New("revision conflicts")
var ErrCapacity = errors.New("contribution capacity")
var reportsBucket = []byte("reports-v1")
var tombstonesBucket = []byte("withdrawn-v1")

type Stored struct {
	Report       Report `json:"report"`
	Digest       string `json:"digest"`
	ReceivedHour int64  `json:"received_hour"`
}
type Tombstone struct {
	Revision     uint64 `json:"revision"`
	ReceivedHour int64  `json:"received_hour"`
}
type Store struct {
	db           *bolt.DB
	maxReporters int
	now          func() time.Time
}

func Open(path string, maxReporters int) (*Store, error) {
	if maxReporters < 1 {
		return nil, errors.New("invalid capacity")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, err
	}
	if err = db.Update(func(tx *bolt.Tx) error {
		for _, key := range [][]byte{reportsBucket, tombstonesBucket} {
			if _, e := tx.CreateBucketIfNotExists(key); e != nil {
				return e
			}
		}
		return nil
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, maxReporters: maxReporters, now: time.Now}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Backup(path string) error {
	return s.db.View(func(tx *bolt.Tx) error { return tx.CopyFile(path, 0600) })
}
func reporterKey(public string) []byte { digest := sha256.Sum256([]byte(public)); return digest[:] }

func (s *Store) Put(report Report, body []byte, withdraw bool) (bool, error) {
	duplicate := false
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	key := reporterKey(report.PublicKey)
	err := s.db.Update(func(tx *bolt.Tx) error {
		reports, tombs := tx.Bucket(reportsBucket), tx.Bucket(tombstonesBucket)
		var previous Stored
		var tomb Tombstone
		existing := reports.Get(key)
		retired := tombs.Get(key)
		if existing != nil {
			if err := json.Unmarshal(existing, &previous); err != nil {
				return err
			}
		}
		if retired != nil {
			if err := json.Unmarshal(retired, &tomb); err != nil {
				return err
			}
		}
		maximum := previous.Report.Revision
		if tomb.Revision > maximum {
			maximum = tomb.Revision
		}
		if report.Revision < maximum {
			return ErrStale
		}
		if report.Revision == maximum {
			if withdraw && existing == nil && retired != nil {
				duplicate = true
				return nil
			}
			if !withdraw && existing != nil && previous.Digest == hash {
				duplicate = true
				return nil
			}
			return ErrConflict
		}
		hour := s.now().UTC().Unix() / 3600
		if existing == nil && retired == nil && reports.Stats().KeyN+tombs.Stats().KeyN >= s.maxReporters {
			return ErrCapacity
		}
		if withdraw {
			data, _ := json.Marshal(Tombstone{report.Revision, hour})
			if err := tombs.Put(key, data); err != nil {
				return err
			}
			return reports.Delete(key)
		}
		// One contribution snapshot per installation/origin. Increasing a revision
		// REPLACES overlapping history instead of adding it to the sample count.
		data, err := json.Marshal(Stored{report, hash, hour})
		if err != nil {
			return err
		}
		if err = reports.Put(key, data); err != nil {
			return err
		}
		return tombs.Delete(key)
	})
	return duplicate, err
}

func (s *Store) Snapshot() ([]Summary, int64, error) {
	var summaries []Summary
	var updated int64
	cutoff := (s.now().UTC().Unix() / 3600) - 120*24
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(reportsBucket).ForEach(func(_, value []byte) error {
			var item Stored
			if err := json.Unmarshal(value, &item); err != nil {
				return err
			}
			if item.ReceivedHour < cutoff || item.Report.MethodDigest != protocol.Digest() {
				return nil
			}
			summaries = append(summaries, *item.Report.Summary)
			if item.ReceivedHour > updated {
				updated = item.ReceivedHour
			}
			return nil
		})
	})
	return summaries, updated, err
}

// Remove inactive statistics, retaining only an opaque key/revision tombstone
// so an old signed packet cannot silently restore a withdrawn/expired report.
func (s *Store) Prune() error {
	cutoff := s.now().UTC().Unix()/3600 - 120*24
	return s.db.Update(func(tx *bolt.Tx) error {
		reports, tombs := tx.Bucket(reportsBucket), tx.Bucket(tombstonesBucket)
		var expired [][]byte
		if err := reports.ForEach(func(key, value []byte) error {
			var item Stored
			if err := json.Unmarshal(value, &item); err != nil {
				return err
			}
			if item.ReceivedHour < cutoff {
				expired = append(expired, append([]byte(nil), key...))
				data, _ := json.Marshal(Tombstone{item.Report.Revision, item.ReceivedHour})
				if err := tombs.Put(key, data); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		for _, key := range expired {
			if err := reports.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}
