package study

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"

	bolt "go.etcd.io/bbolt"
)

// Bump when normalization or counting changes; catalog changes invalidate by digest.
const surveyCacheVersion = 1

var surveyCacheStateKey = []byte("aggregate")
var surveyCatalogDigest = func() string {
	digest := sha256.Sum256(surveyCatalogJSON)
	return hex.EncodeToString(digest[:])
}()

// Counts are indexed by the eight mutually exclusive combinations of outcomes.
type surveyFactorCounts struct {
	Applicable [8]int            `json:"applicable"`
	Options    map[string][8]int `json:"options"`
}
type surveyAccumulator struct {
	Statuses   [8]int                         `json:"statuses"`
	Factors    map[string]*surveyFactorCounts `json:"factors"`
	UsageCount int                            `json:"usageCount"`
	UsageSums  [24]int                        `json:"usageSums"`
}
type surveyCachedAggregate struct {
	Version       int                   `json:"version"`
	CatalogDigest string                `json:"catalogDigest"`
	Counts        surveyAccumulator     `json:"counts"`
	Range         surveyStatisticsRange `json:"range"`
	Result        surveyStatistics      `json:"result"`
}

func newSurveyAggregate() surveyCachedAggregate {
	state := surveyCachedAggregate{Version: surveyCacheVersion, CatalogDigest: surveyCatalogDigest, Counts: surveyAccumulator{Factors: make(map[string]*surveyFactorCounts, len(surveyCatalog.Definitions))}}
	for _, definition := range surveyCatalog.Definitions {
		options := make(map[string][8]int, len(surveyCatalog.Choices[definition.Key]))
		for _, label := range surveyCatalog.Choices[definition.Key] {
			options[label] = [8]int{}
		}
		state.Counts.Factors[definition.Key] = &surveyFactorCounts{Options: options}
	}
	return state
}

func decodeSurveyAggregate(raw []byte) (surveyCachedAggregate, bool) {
	var state surveyCachedAggregate
	if len(raw) == 0 || json.Unmarshal(raw, &state) != nil || state.Version != surveyCacheVersion || state.CatalogDigest != surveyCatalogDigest {
		return state, false
	}
	for _, definition := range surveyCatalog.Definitions {
		counts := state.Counts.Factors[definition.Key]
		if counts == nil {
			return state, false
		}
		for _, label := range surveyCatalog.Choices[definition.Key] {
			if _, ok := counts.Options[label]; !ok {
				return state, false
			}
		}
	}
	return state, true
}

func lastSurveyID(bucket *bolt.Bucket) (uint64, error) {
	key, _ := bucket.Cursor().Last()
	if key == nil {
		return 0, nil
	}
	if len(key) != 8 {
		return 0, errors.New("invalid stored survey key")
	}
	return binary.BigEndian.Uint64(key), nil
}

func (state *surveyCachedAggregate) add(id uint64, record storedSurveyRecord) error {
	q := record.SurveySubmission
	mask := 0
	if q.degraded() {
		mask |= 1
	}
	if q.banned() {
		mask |= 2
	}
	if q.limited() {
		mask |= 4
	}
	state.Counts.Statuses[mask]++
	if q.UsagePattern != nil {
		if len(q.UsagePattern) != 24 {
			return errors.New("invalid stored usage pattern")
		}
		for hour, level := range q.UsagePattern {
			if level == nil || *level < 0 || *level > 24 {
				return errors.New("invalid stored usage pattern")
			}
			state.Counts.UsageSums[hour] += *level
		}
		state.Counts.UsageCount++
	}
	for key, values := range q.normalized() {
		counts := state.Counts.Factors[key]
		if counts == nil || len(values) == 0 {
			continue
		}
		counts.Applicable[mask]++
		for index, label := range values {
			if slices.Contains(values[:index], label) {
				continue
			}
			option, ok := counts.Options[label]
			if !ok {
				return errors.New("unknown stored survey option")
			}
			option[mask]++
			counts.Options[label] = option
		}
	}
	if state.Range.FirstSubmissionID == 0 {
		state.Range.FirstSubmissionID = id
	}
	state.Range.LastSubmissionID = id
	if record.SubmittedAt == nil {
		state.Range.UnknownTimeCount++
	} else {
		if state.Range.FirstSubmittedAt == nil || record.SubmittedAt.Before(*state.Range.FirstSubmittedAt) {
			state.Range.FirstSubmittedAt = record.SubmittedAt
		}
		if state.Range.LastSubmittedAt == nil || record.SubmittedAt.After(*state.Range.LastSubmittedAt) {
			state.Range.LastSubmittedAt = record.SubmittedAt
		}
	}
	return nil
}

func outcomeCount(counts [8]int, bit int) int {
	total := 0
	for mask, count := range counts {
		if mask&bit != 0 {
			total += count
		}
	}
	return total
}
func totalCount(counts [8]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}
func countedAssociation(selected, applicable [8]int, bit int) surveyAssociation {
	a := outcomeCount(selected, bit)
	b := totalCount(selected) - a
	c := outcomeCount(applicable, bit) - a
	d := totalCount(applicable) - a - b - c
	return association(a, b, c, d)
}

func (state *surveyCachedAggregate) statistics() surveyStatistics {
	counts := &state.Counts
	result := surveyStatistics{
		Total: totalCount(counts.Statuses), Normal: counts.Statuses[0],
		Degraded: outcomeCount(counts.Statuses, 1), Banned: outcomeCount(counts.Statuses, 2), Limited: outcomeCount(counts.Statuses, 4),
		Both:    counts.Statuses[3] + counts.Statuses[7],
		Factors: make([]surveyFactor, 0, len(surveyCatalog.Definitions)), Associations: []surveyAssociationGroup{}, Range: state.Range,
	}
	result.Statuses = []surveyStatusCount{
		{Label: "正常", Tone: "mint"}, {Label: "降智", Tone: "violet"},
		{Label: "封号", Tone: "peach"}, {Label: "降智、封号", Tone: "peach"},
		{Label: "风控（限流）", Tone: "sun"}, {Label: "降智、风控（限流）", Tone: "sun"},
		{Label: "封号、风控（限流）", Tone: "sun"}, {Label: "降智、封号、风控（限流）", Tone: "sun"},
	}
	for index, count := range counts.Statuses {
		result.Statuses[index].Count = count
	}
	result.UsagePattern.Total = counts.UsageCount
	if counts.UsageCount > 0 {
		for hour, sum := range counts.UsageSums {
			result.UsagePattern.Levels[hour] = float64(sum) / float64(counts.UsageCount)
		}
	}
	for _, definition := range surveyCatalog.Definitions {
		key := definition.Key
		factorCounts := counts.Factors[key]
		factor := surveyFactor{ID: key, Title: definition.Title, Description: definition.Description, Multiple: definition.Multiple, Eligibility: definition.Eligibility, Groups: make(map[string]surveyGroup, 3)}
		if factor.Eligibility == "" {
			factor.Eligibility = "分母为所选异常状态中回答此题的问卷；漏答不计入。"
		}
		for index, outcome := range []string{"degraded", "banned", "limited"} {
			bit := 1 << index
			group := surveyGroup{Total: outcomeCount(factorCounts.Applicable, bit), Rows: make([]surveyCount, 0, len(surveyCatalog.Choices[key]))}
			for _, label := range surveyCatalog.Choices[key] {
				group.Rows = append(group.Rows, surveyCount{Label: label, Count: outcomeCount(factorCounts.Options[label], bit)})
			}
			factor.Groups[outcome] = group
		}
		result.Factors = append(result.Factors, factor)
		if slices.Contains([]string{"models", "discovery", "limitedDiscovery"}, key) {
			continue
		}
		group := surveyAssociationGroup{ID: key, Title: definition.Title, Scope: definition.Eligibility, Total: totalCount(factorCounts.Applicable), Rows: make([]surveyAssociationRow, 0, len(surveyCatalog.Choices[key]))}
		if group.Scope == "" {
			group.Scope = "所有回答此题的问卷。"
		}
		for _, label := range surveyCatalog.Choices[key] {
			selected := factorCounts.Options[label]
			group.Rows = append(group.Rows, surveyAssociationRow{Label: label, Degraded: countedAssociation(selected, factorCounts.Applicable, 1), Banned: countedAssociation(selected, factorCounts.Applicable, 2), Limited: countedAssociation(selected, factorCounts.Applicable, 4)})
		}
		result.Associations = append(result.Associations, group)
	}
	result.OutcomeAssociation = association(result.Both, result.Degraded-result.Both, result.Banned-result.Both, result.Total-result.Degraded-result.Banned+result.Both)
	return result
}

func (s *Server) surveyStatistics() (surveyStatistics, error) {
	var result surveyStatistics
	hit := false
	err := s.store.db.View(func(tx *bolt.Tx) error {
		last, err := lastSurveyID(tx.Bucket(surveyBucket))
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(tx.Bucket(surveyCacheBucket).Get(surveyCacheStateKey))
		if valid && state.Range.LastSubmissionID == last {
			result = state.Result
			hit = true
		}
		return nil
	})
	if err != nil || hit {
		return result, err
	}
	// The write transaction serializes refreshes and commits counts, cutoff and result together.
	err = s.store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(surveyBucket)
		meta := tx.Bucket(surveyCacheBucket)
		last, err := lastSurveyID(bucket)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(meta.Get(surveyCacheStateKey))
		if valid && state.Range.LastSubmissionID == last {
			result = state.Result
			return nil
		}
		if !valid || state.Range.LastSubmissionID > last {
			state = newSurveyAggregate()
		}
		var next [8]byte
		binary.BigEndian.PutUint64(next[:], state.Range.LastSubmissionID+1)
		cursor := bucket.Cursor()
		for key, body := cursor.Seek(next[:]); key != nil; key, body = cursor.Next() {
			if len(key) != 8 {
				return errors.New("invalid stored survey key")
			}
			var record storedSurveyRecord
			if err := json.Unmarshal(body, &record); err != nil {
				return err
			}
			if err := state.add(binary.BigEndian.Uint64(key), record); err != nil {
				return err
			}
		}
		state.Range.ComputedAt = s.store.now().UTC()
		state.Result = state.statistics()
		encoded, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if err := meta.Put(surveyCacheStateKey, encoded); err != nil {
			return err
		}
		result = state.Result
		return nil
	})
	return result, err
}
