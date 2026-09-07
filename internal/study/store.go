package study

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var ErrConflict = errors.New("revision conflicts")
var ErrCapacity = errors.New("contribution capacity")

var batchesBucket = []byte("evidence-batches")
var identitiesBucket = []byte("evidence-identities")

type Stored struct {
	Report       Report `json:"report"`
	Digest       string `json:"digest"`
	ReceivedHour int64  `json:"received_hour"`
}

type IdentityState struct {
	Highest uint64 `json:"highest"`
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
		for _, key := range [][]byte{batchesBucket, identitiesBucket} {
			if _, createErr := tx.CreateBucketIfNotExists(key); createErr != nil {
				return createErr
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, maxReporters: maxReporters, now: time.Now}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Backup(path string) error {
	return s.db.View(func(tx *bolt.Tx) error { return tx.CopyFile(path, 0600) })
}

func reporterKey(public string) []byte {
	digest := sha256.Sum256([]byte(public))
	return digest[:]
}

func batchKey(public, id string) []byte {
	return append(append(reporterKey(public), ':'), []byte(id)...)
}

func loadIdentity(tx *bolt.Tx, key []byte) (IdentityState, error) {
	var state IdentityState
	if data := tx.Bucket(identitiesBucket).Get(key); data != nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return state, err
		}
	}
	return state, nil
}

func saveIdentity(tx *bolt.Tx, key []byte, state IdentityState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return tx.Bucket(identitiesBucket).Put(key, data)
}

func (s *Store) Put(report Report, body []byte) (bool, error) {
	duplicate := false
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	key := reporterKey(report.PublicKey)
	bkey := batchKey(report.PublicKey, report.BatchID)
	err := s.db.Update(func(tx *bolt.Tx) error {
		state, err := loadIdentity(tx, key)
		if err != nil {
			return err
		}
		batches := tx.Bucket(batchesBucket)
		var existing Stored
		if raw := batches.Get(bkey); raw != nil {
			if err := json.Unmarshal(raw, &existing); err != nil {
				return err
			}
			if report.Revision == existing.Report.Revision && existing.Digest == hash {
				duplicate = true
				return nil
			}
		}
		if report.Revision <= state.Highest {
			return ErrConflict
		}
		if tx.Bucket(identitiesBucket).Get(key) == nil && tx.Bucket(identitiesBucket).Stats().KeyN >= s.maxReporters {
			return ErrCapacity
		}
		if batches.Get(bkey) == nil && batches.Stats().KeyN >= 100000 {
			return ErrCapacity
		}
		data, err := json.Marshal(Stored{Report: report, Digest: hash, ReceivedHour: s.now().UTC().Unix() / 3600})
		if err != nil {
			return err
		}
		if err := batches.Put(bkey, data); err != nil {
			return err
		}
		state.Highest = report.Revision
		return saveIdentity(tx, key, state)
	})
	return duplicate, err
}

func (s *Store) Walk(visit func(string, Summary, int64) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(batchesBucket).ForEach(func(k, value []byte) error {
			var item Stored
			if err := json.Unmarshal(value, &item); err != nil {
				return err
			}
			if item.Report.Summary == nil {
				return errors.New("stored summary missing")
			}
			return visit(hex.EncodeToString(k[:32]), *item.Report.Summary, item.ReceivedHour)
		})
	})
}
