package study

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func surveyRows(tx *sql.Tx) (map[uint64][]byte, error) {
	rows, err := tx.Query("SELECT id,payload FROM survey_submissions ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[uint64][]byte{}
	for rows.Next() {
		var id uint64
		var body []byte
		if err := rows.Scan(&id, &body); err != nil {
			return nil, err
		}
		result[id] = bytes.Clone(body)
	}
	return result, rows.Err()
}

func TestSurveyCountsMigrationPreservesRawRecords(t *testing.T) {
	for _, size := range []int{0, 9} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "survey.sqlite")
			store, err := Open(path, 10)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { store.Close() }()
			for i := range size {
				if err := store.putSurvey(cacheQuestion(i)); err != nil {
					t.Fatal(err)
				}
			}
			before, err := NewServer(store, fstest.MapFS{}).surveyStatistics()
			if err != nil {
				t.Fatal(err)
			}
			var original map[uint64][]byte
			var sequence uint64
			if err := store.db.Update(func(tx *sql.Tx) error {
				var err error
				original, err = surveyRows(tx)
				if err != nil {
					return err
				}
				if err := tx.QueryRow("SELECT COALESCE((SELECT seq FROM sqlite_sequence WHERE name='survey_submissions'),0)").Scan(&sequence); err != nil {
					return err
				}
				body, err := readSurveyAggregate(tx)
				if err != nil {
					return err
				}
				var legacy map[string]json.RawMessage
				if err := json.Unmarshal(body, &legacy); err != nil {
					return err
				}
				legacy["version"] = json.RawMessage(`1`)
				legacy["result"] = json.RawMessage(`{"total":999999}`)
				body, err = json.Marshal(legacy)
				if err != nil {
					return err
				}
				_, err = tx.Exec("UPDATE survey_statistics SET payload=? WHERE id=1", string(body))
				return err
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
			server := NewServer(store, fstest.MapFS{})
			after, err := server.surveyStatistics()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("cache upgrade changed counts or range")
			}
			if err := store.db.View(func(tx *sql.Tx) error {
				current, err := surveyRows(tx)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(original, current) {
					return fmt.Errorf("cache upgrade changed raw records")
				}
				body, err := readSurveyAggregate(tx)
				if err != nil {
					return err
				}
				var cache map[string]json.RawMessage
				if err := json.Unmarshal(body, &cache); err != nil {
					return err
				}
				if string(cache["version"]) != "2" || cache["result"] != nil {
					return fmt.Errorf("obsolete derived cache remains")
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
				t.Fatal("append lost or duplicated history")
			}
			if err := store.db.View(func(tx *sql.Tx) error {
				current, err := surveyRows(tx)
				if err != nil {
					return err
				}
				for id, body := range original {
					if !bytes.Equal(current[id], body) {
						return fmt.Errorf("incremental refresh changed historical data")
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
