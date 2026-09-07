package study

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
)

var batchesBucket = []byte("evidence-batches-v2")
var identitiesBucket = []byte("evidence-identities-v2")

type StoredV2 struct {
	Report       ReportV2 `json:"report"`
	Digest       string   `json:"digest"`
	ReceivedHour int64    `json:"received_hour"`
}
type IdentityState struct {
	Highest   uint64 `json:"highest"`
	Withdrawn uint64 `json:"withdrawn"`
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
	var legacy Stored
	if data := tx.Bucket(reportsBucket).Get(key); data != nil {
		if err := json.Unmarshal(data, &legacy); err != nil {
			return state, err
		}
		if legacy.Report.Revision > state.Highest {
			state.Highest = legacy.Report.Revision
		}
	}
	var tomb Tombstone
	if data := tx.Bucket(tombstonesBucket).Get(key); data != nil {
		if err := json.Unmarshal(data, &tomb); err != nil {
			return state, err
		}
		if tomb.Revision > state.Highest {
			state.Highest = tomb.Revision
		}
		if tomb.Revision > state.Withdrawn {
			state.Withdrawn = tomb.Revision
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

func (s *Store) PutV2(report ReportV2, body []byte, withdraw bool) (bool, error) {
	if withdraw {
		return s.withdrawAll(report.PublicKey, report.Revision)
	}
	duplicate := false
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	key := reporterKey(report.PublicKey)
	bkey := batchKey(report.PublicKey, report.BatchID)
	err := s.db.Update(func(tx *bolt.Tx) error {
		state, err := loadIdentity(tx, key)
		if err != nil {
			return err
		}
		if report.Revision <= state.Withdrawn {
			return ErrStale
		}
		batches := tx.Bucket(batchesBucket)
		var old StoredV2
		if raw := batches.Get(bkey); raw != nil {
			if err := json.Unmarshal(raw, &old); err != nil {
				return err
			}
			if report.Revision == old.Report.Revision && old.Digest == hash {
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
		// This is storage exhaustion protection, not a scientific sample gate.
		// Existing batches remain updateable; no data is pruned to make room.
		if batches.Get(bkey) == nil && batches.Stats().KeyN >= 100000 {
			return ErrCapacity
		}
		hour := s.now().UTC().Unix() / 3600
		data, err := json.Marshal(StoredV2{report, hash, hour})
		if err != nil {
			return err
		}
		if err := batches.Put(bkey, data); err != nil {
			return err
		}
		state.Highest = report.Revision
		// Keep the withdrawal floor even after re-consent: old batches must
		// never reappear via a delayed formerly valid signature.
		return saveIdentity(tx, key, state)
	})
	return duplicate, err
}

func (s *Store) withdrawAll(public string, revision uint64) (bool, error) {
	duplicate := false
	key := reporterKey(public)
	err := s.db.Update(func(tx *bolt.Tx) error {
		state, err := loadIdentity(tx, key)
		if err != nil {
			return err
		}
		if revision < state.Highest {
			return ErrStale
		}
		if revision == state.Highest {
			if revision == state.Withdrawn {
				duplicate = true
				return nil
			}
			return ErrConflict
		}
		if tx.Bucket(identitiesBucket).Get(key) == nil && tx.Bucket(identitiesBucket).Stats().KeyN >= s.maxReporters {
			return ErrCapacity
		}
		prefix := append(append([]byte{}, key...), ':')
		bucket := tx.Bucket(batchesBucket)
		// Cursor delete preserves the next key; no per-request public endpoint.
		c := bucket.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			if err := c.Delete(); err != nil {
				return err
			}
		}
		if err := tx.Bucket(reportsBucket).Delete(key); err != nil {
			return err
		}
		tomb, err := json.Marshal(Tombstone{revision, s.now().UTC().Unix() / 3600})
		if err != nil {
			return err
		}
		if err := tx.Bucket(tombstonesBucket).Put(key, tomb); err != nil {
			return err
		}
		return saveIdentity(tx, key, IdentityState{Highest: revision, Withdrawn: revision})
	})
	return duplicate, err
}

// WalkV2 streams one batch at a time. Memory does not grow with retained history.
// The identity hash stays in-process to compute independent-source diagnostics.
func (s *Store) WalkV2(visit func(string, SummaryV2, int64) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(batchesBucket).ForEach(func(k, value []byte) error {
			var item StoredV2
			if err := json.Unmarshal(value, &item); err != nil {
				return err
			}
			if item.Report.Summary == nil {
				return fmt.Errorf("stored summary missing")
			}
			return visit(hex.EncodeToString(k[:32]), *item.Report.Summary, item.ReceivedHour)
		})
	})
}
