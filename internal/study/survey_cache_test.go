package study

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	bolt "go.etcd.io/bbolt"
)

func cacheQuestion(index int) SurveySubmission {
	states := [][]string{{"正常"}, {"降智"}, {"封号"}, {"降智", "封号"}, {"风控（限流）"}, {"降智", "风控（限流）"}, {"封号", "风控（限流）"}, {"降智", "封号", "风控（限流）"}}
	q := SurveySubmission{Status: states[index%8], Answers: map[string][]string{"plans": {"Plus"}, "usage": {"直登", "反代"}, "network": {"家宽"}}, Details: map[string]string{}}
	if index%3 == 0 {
		q.Answers["plans"] = []string{"Free 免费"}
		q.Answers["network"] = []string{"机场"}
	}
	if index%2 == 0 {
		q.Answers["tools"] = []string{"Pi", "Cursor"}
		q.Details["toolMode:Pi"] = "反代"
		q.Details["toolMode:Cursor"] = "反代"
	}
	if index%5 != 0 {
		risk := index % 101
		q.IPRisk = &risk
	}
	if index%3 != 0 {
		q.UsagePattern = make([]*int, 24)
		for hour := range q.UsagePattern {
			value := (index + hour) % 25
			q.UsagePattern[hour] = &value
		}
	}
	if q.degraded() {
		q.Answers["discovery"] = []string{"画鹈鹕"}
	}
	if q.limited() {
		q.Answers["limitedDiscovery"] = []string{"服务不可用"}
	}
	return q
}

func TestSurveyIncrementalMatchesRawEvidence(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	var records []SurveySubmission
	for batch := 0; batch < 4; batch++ {
		for i := 0; i < 13; i++ {
			q := cacheQuestion(len(records))
			if err := validateSurvey(q); err != nil {
				t.Fatal(err)
			}
			if err := store.putSurvey(q); err != nil {
				t.Fatal(err)
			}
			records = append(records, q)
		}
		result, err := server.surveyStatistics()
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != len(records) || result.Range.LastSubmissionID != uint64(len(records)) {
			t.Fatal("incremental total or cutoff")
		}
		for _, factor := range result.Factors {
			for _, outcome := range []string{"degraded", "banned", "limited"} {
				expected := 0
				for _, q := range records {
					applies := q.degraded()
					if outcome == "banned" {
						applies = q.banned()
					}
					if outcome == "limited" {
						applies = q.limited()
					}
					if applies && len(q.normalized()[factor.ID]) > 0 {
						expected++
					}
				}
				if factor.Groups[outcome].Total != expected {
					t.Fatalf("%s %s denominator", factor.ID, outcome)
				}
			}
		}
		for _, group := range result.Associations {
			for _, row := range group.Rows {
				for i, actual := range []surveyAssociation{row.Degraded, row.Banned, row.Limited} {
					table := [4]int{}
					for _, q := range records {
						values := q.normalized()[group.ID]
						if len(values) == 0 {
							continue
						}
						selected := false
						for _, value := range values {
							if value == row.Label {
								selected = true
							}
						}
						event := q.degraded()
						if i == 1 {
							event = q.banned()
						}
						if i == 2 {
							event = q.limited()
						}
						slot := 0
						if !selected {
							slot = 2
						}
						if !event {
							slot++
						}
						table[slot]++
					}
					expected := association(table[0], table[1], table[2], table[3])
					if !reflect.DeepEqual(actual, expected) {
						t.Fatalf("%s %s outcome %d: %#v != %#v", group.ID, row.Label, i, actual, expected)
					}
				}
			}
		}
		sums := [24]int{}
		n := 0
		for _, q := range records {
			if q.UsagePattern != nil {
				n++
				for hour, value := range q.UsagePattern {
					sums[hour] += *value
				}
			}
		}
		if result.UsagePattern.Total != n {
			t.Fatal("usage denominator")
		}
		for hour, sum := range sums {
			if result.UsagePattern.Levels[hour] != float64(sum)/float64(n) {
				t.Fatal("averages were added instead of sums")
			}
		}
	}
}

func TestSurveyCacheSurvivesRestartAndSkipsOldRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, fstest.MapFS{})
	if err := store.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	first, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	// A cached historical record is deliberately unreadable: only the persisted counts may be used.
	if err := store.db.Update(func(tx *bolt.Tx) error {
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], 1)
		return tx.Bucket(surveyBucket).Put(key[:], []byte("unreadable historical record"))
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
	defer store.Close()
	server = NewServer(store, fstest.MapFS{})
	hit, err := server.surveyStatistics()
	if err != nil {
		t.Fatal("cache rescanned history", err)
	}
	if !reflect.DeepEqual(first, hit) {
		t.Fatal("cache hit changed result")
	}
	if err := store.putSurvey(cacheQuestion(7)); err != nil {
		t.Fatal(err)
	}
	delta, err := server.surveyStatistics()
	if err != nil {
		t.Fatal("incremental refresh rescanned history", err)
	}
	if delta.Total != 2 || delta.Normal != 1 || delta.Both != 1 || delta.Range.LastSubmissionID != 2 {
		t.Fatal("lost persistent base counts")
	}
}

func TestSurveyCacheRefreshIsAtomicAndConcurrent(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	first, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 0 || first.Range.LastSubmissionID != 0 || first.Range.FirstSubmittedAt != nil {
		t.Fatal("empty range")
	}
	if err := store.putSurvey(cacheQuestion(1)); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], id)
		return bucket.Put(key[:], []byte("invalid"))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.surveyStatistics(); err == nil {
		t.Fatal("invalid incremental record accepted")
	}
	if err := store.db.View(func(tx *bolt.Tx) error {
		state, valid := decodeSurveyAggregate(tx.Bucket(surveyCacheBucket).Get(surveyCacheStateKey))
		if !valid || state.Result.Total != 0 || state.Range.LastSubmissionID != 0 {
			return fmt.Errorf("failed refresh advanced progress")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *bolt.Tx) error {
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], 2)
		body, _ := json.Marshal(cacheQuestion(2))
		return tx.Bucket(surveyBucket).Put(key[:], body)
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) { defer wg.Done(); failures <- store.putSurvey(cacheQuestion(i)) }(i)
		go func() { defer wg.Done(); _, err := server.surveyStatistics(); failures <- err }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 22 || result.Range.LastSubmissionID != 22 {
		t.Fatal("concurrent refresh lost or duplicated records")
	}
}

func TestSurveyServerTimeAndLegacyRiskMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.db")
	store, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"network":["宽带"],"quality":["优秀"]}}`
	if err := store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], id)
		if err := bucket.Put(key[:], []byte(legacy)); err != nil {
			return err
		}
		return tx.Bucket(surveyCacheBucket).Delete(surveyStorageVersionKey)
	}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	store.now = func() time.Time { return now }
	server := NewServer(store, fstest.MapFS{})
	submit := func(body string) int {
		r := httptest.NewRequest("POST", "/api/survey/submissions", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w.Code
	}
	for _, risk := range []int{0, 100} {
		if code := submit(fmt.Sprintf(`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"network":["机场"]},"ipRisk":%d}`, risk)); code != 201 {
			t.Fatal(code)
		}
	}
	for _, body := range []string{
		`{"status":["正常"],"answers":{"plans":["Plus"]},"ipRisk":10}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"]},"ipRisk":101}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"]},"ipRisk":-1}`,
		`{"status":["正常"],"answers":{"plans":["Plus"]},"submittedAt":"2000-01-01T00:00:00Z"}`,
		`{"status":["正常"],"answers":{"plans":["Plus"],"usage":["反代"],"quality":["优秀"]}}`,
	} {
		if submit(body) != 400 {
			t.Fatal("invalid risk or client timestamp accepted")
		}
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Range.UnknownTimeCount != 1 || stats.Range.FirstSubmittedAt == nil || !stats.Range.FirstSubmittedAt.Equal(now) || !stats.Range.LastSubmittedAt.Equal(now) {
		t.Fatal("server time / legacy unknown time")
	}
	for _, group := range stats.Associations {
		if group.ID == "ipRisk" && group.Total != 2 {
			t.Fatal("legacy rating fabricated a numeric score")
		}
		if group.ID == "network" && (group.Rows[0].Degraded.Selected.Total != 1 || group.Rows[2].Degraded.Selected.Total != 2) {
			t.Fatal("network migration")
		}
	}
	if err := store.db.View(func(tx *bolt.Tx) error {
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], 1)
		var record storedSurveyRecord
		if err := json.Unmarshal(tx.Bucket(surveyBucket).Get(key[:]), &record); err != nil {
			return err
		}
		if record.SubmittedAt != nil || string(record.LegacyIPQuality) != `["优秀"]` {
			return fmt.Errorf("legacy data lost or timestamp fabricated")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSurveyCacheInvalidationRebuildsEvidence(t *testing.T) {
	store := openTest(t, 10)
	server := NewServer(store, fstest.MapFS{})
	if err := store.putSurvey(cacheQuestion(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := server.surveyStatistics(); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(tx *bolt.Tx) error {
		meta := tx.Bucket(surveyCacheBucket)
		state, valid := decodeSurveyAggregate(meta.Get(surveyCacheStateKey))
		if !valid {
			return fmt.Errorf("missing cache")
		}
		state.CatalogDigest = "previous-catalog"
		body, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if err := meta.Put(surveyCacheStateKey, body); err != nil {
			return err
		}
		var key [8]byte
		binary.BigEndian.PutUint64(key[:], 1)
		replacement, err := json.Marshal(cacheQuestion(7))
		if err != nil {
			return err
		}
		return tx.Bucket(surveyBucket).Put(key[:], replacement)
	}); err != nil {
		t.Fatal(err)
	}
	stats, err := server.surveyStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 1 || stats.Normal != 0 || stats.Both != 1 || stats.Limited != 1 {
		t.Fatal("stale cached counts reused after invalidation")
	}
}
