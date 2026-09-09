package study

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
)

// Version 2 drops derived results; version 1 has identical counts and is upgraded in place.
const surveyCacheVersion = 2

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
	if len(raw) == 0 || json.Unmarshal(raw, &state) != nil || state.CatalogDigest != surveyCatalogDigest {
		return state, false
	}
	if state.Version != surveyCacheVersion && !(surveyCacheVersion == 2 && state.Version == 1) {
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

func lastSurveyID(tx *sql.Tx) (uint64, error) {
	var id uint64
	err := tx.QueryRow("SELECT COALESCE(MAX(id),0) FROM survey_submissions").Scan(&id)
	return id, err
}

func readSurveyAggregate(tx *sql.Tx) ([]byte, error) {
	var body []byte
	err := tx.QueryRow("SELECT payload FROM survey_statistics WHERE id=1").Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return body, err
}

func (state *surveyCachedAggregate) add(id uint64, record storedSurveyRecord) error {
	q := record.SurveySubmission
	mask := surveyStatusMask(q)
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

func (state *surveyCachedAggregate) summary() surveySummary {
	result := surveySummary{
		Version: surveySummaryVersion, CatalogDigest: state.CatalogDigest,
		Statuses: state.Counts.Statuses, Range: state.Range,
		UsagePattern: surveyUsageSummary{Total: state.Counts.UsageCount, Sums: state.Counts.UsageSums},
		Factors:      make([]surveyFactorSummary, 0, len(surveyCatalog.Definitions)),
	}
	for _, definition := range surveyCatalog.Definitions {
		counts := state.Counts.Factors[definition.Key]
		factor := surveyFactorSummary{
			ID: definition.Key, Title: definition.Title, Description: definition.Description,
			Multiple: definition.Multiple, Applicable: counts.Applicable,
			DistributionScope: definition.Eligibility, AssociationScope: definition.Eligibility,
			Comparable: !slices.Contains([]string{"models", "discovery", "limitedDiscovery"}, definition.Key),
			Options:    make([]surveyOptionSummary, 0, len(surveyCatalog.Choices[definition.Key])),
		}
		if factor.DistributionScope == "" {
			factor.DistributionScope = "分母为所选异常状态中回答此题的问卷；漏答不计入。"
		}
		if factor.AssociationScope == "" {
			factor.AssociationScope = "所有回答此题的问卷。"
		}
		for _, label := range surveyCatalog.Choices[definition.Key] {
			factor.Options = append(factor.Options, surveyOptionSummary{Label: label, Counts: counts.Options[label]})
		}
		result.Factors = append(result.Factors, factor)
	}
	return result
}

func (s *Server) surveyStatistics() (surveySummary, error) {
	var result surveySummary
	hit := false
	err := s.store.db.View(func(tx *sql.Tx) error {
		last, err := lastSurveyID(tx)
		if err != nil {
			return err
		}
		raw, err := readSurveyAggregate(tx)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(raw)
		if valid && state.Version == surveyCacheVersion && state.Range.LastSubmissionID == last {
			result = state.summary()
			hit = true
		}
		return nil
	})
	if err != nil || hit {
		return result, err
	}
	// Refreshes and version upgrades atomically commit counts and their cutoff, never raw answers.
	err = s.store.db.Update(func(tx *sql.Tx) error {
		last, err := lastSurveyID(tx)
		if err != nil {
			return err
		}
		raw, err := readSurveyAggregate(tx)
		if err != nil {
			return err
		}
		state, valid := decodeSurveyAggregate(raw)
		if valid && state.Version == surveyCacheVersion && state.Range.LastSubmissionID == last {
			result = state.summary()
			return nil
		}
		if !valid || state.Range.LastSubmissionID > last {
			state = newSurveyAggregate()
		}
		previousCutoff := state.Range.LastSubmissionID
		rows, err := tx.Query("SELECT id,payload FROM survey_submissions WHERE id>? ORDER BY id", state.Range.LastSubmissionID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uint64
			var body []byte
			if err := rows.Scan(&id, &body); err != nil {
				return err
			}
			var record storedSurveyRecord
			if err := json.Unmarshal(body, &record); err != nil {
				return err
			}
			if err := state.add(id, record); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if state.Range.ComputedAt.IsZero() || state.Range.LastSubmissionID != previousCutoff {
			state.Range.ComputedAt = s.store.now().UTC()
		}
		state.Version = surveyCacheVersion
		encoded, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO survey_statistics(id,payload) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", string(encoded)); err != nil {
			return err
		}
		result = state.summary()
		return nil
	})
	return result, err
}
