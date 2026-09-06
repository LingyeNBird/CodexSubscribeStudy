package study

import (
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"math"
	"math/rand"
	"sort"
	"time"
)

type Totals struct {
	Contributors         int     `json:"contributors"`
	EligibleContributors int     `json:"eligible_contributors"`
	Requests             int64   `json:"requests"`
	GPT6Requests         int64   `json:"gpt6_requests"`
	BaselineRequests     int64   `json:"baseline_requests"`
	RawUSD               float64 `json:"raw_usd"`
	GPT6RawUSD           float64 `json:"gpt6_raw_usd"`
	QuotaPoints          float64 `json:"quota_points"`
	Cycles               int     `json:"cycles"`
	Blocks               int     `json:"blocks"`
}
type Cause struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Support   *float64  `json:"support"`
	ScoreMean *float64  `json:"score_mean"`
	ScoreLow  *float64  `json:"score_low"`
	ScoreHigh *float64  `json:"score_high"`
	Factors   []float64 `json:"factor_estimates"`
}
type Result struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	State             string         `json:"state"`
	Method            string         `json:"method"`
	MethodDigest      string         `json:"method_digest"`
	UpdatedAt         string         `json:"updated_at"`
	WindowDays        int            `json:"window_days"`
	Totals            Totals         `json:"totals"`
	Causes            []Cause        `json:"causes"`
	Quality           map[string]int `json:"quality"`
	IdentifiableSites []int          `json:"identifiable_sites"`
	MinContributors   int            `json:"minimum_contributors"`
	ConfidenceMeaning string         `json:"confidence_meaning"`
}

func Aggregate(summaries []Summary, updated int64) Result {
	result := Result{ID: StudyID, Title: "GPT-6 额度异常归因", State: "collecting", Method: Method, MethodDigest: protocol.Digest(), WindowDays: 90, Quality: map[string]int{}, IdentifiableSites: make([]int, 4), MinContributors: 3,
		ConfidenceMeaning: "两级重抽样预测胜率；是候选集内的探索性支持度，不是真实计费机制的概率或因果证明。"}
	if updated > 0 {
		result.UpdatedAt = time.Unix(updated*3600, 0).UTC().Format("2006-01-02")
	}
	var eligible []Summary
	for _, s := range summaries {
		result.Totals.Contributors++
		result.Totals.Requests += s.Requests
		result.Totals.GPT6Requests += s.GPT6Requests
		result.Totals.BaselineRequests += s.BaselineRequests
		result.Totals.RawUSD += s.RawUSD
		result.Totals.GPT6RawUSD += s.GPT6RawUSD
		result.Totals.QuotaPoints += s.QuotaPoints
		result.Totals.Cycles += s.Cycles
		result.Totals.Blocks += s.Blocks
		result.Quality[s.Status]++
		if s.Eligible {
			eligible = append(eligible, s)
			for j, v := range s.Identifiable {
				if v {
					result.IdentifiableSites[j]++
				}
			}
		}
	}
	result.Totals.EligibleContributors = len(eligible)
	for i, name := range Families {
		result.Causes = append(result.Causes, Cause{ID: name, Label: Labels[i], Factors: []float64{1, 1, 1, 1}})
	}
	if len(summaries) == 0 {
		result.State = "no_data"
	}
	if len(eligible) < 3 {
		return result
	}
	n := len(eligible)
	rng := rand.New(rand.NewSource(60806))
	const draws = 1024
	means := make([]float64, 7)
	factor := make([][]float64, 7)
	for i := range factor {
		factor[i] = make([]float64, 4)
	}
	covariance := make([][][]float64, n)
	baseWeight := make([]float64, n)
	weightSum := 0.0
	for i, s := range eligible {
		covariance[i], _ = cholesky(s.ScoreCov)
		// One prolific installation is not treated as millions of independent votes.
		w := math.Min(float64(s.Cycles), 10)
		baseWeight[i] = w
		weightSum += w
		for j := 0; j < 7; j++ {
			means[j] += w * s.ScoreMean[j]
			for k := 0; k < 4; k++ {
				factor[j][k] += w * s.Factors[j][k]
			}
		}
	}
	for j := 0; j < 7; j++ {
		means[j] /= weightSum
		for k := 0; k < 4; k++ {
			factor[j][k] /= weightSum
		}
	}
	support := make([]float64, 7)
	drawScores := make([][]float64, 7)
	for i := range drawScores {
		drawScores[i] = make([]float64, 0, draws)
	}
	for b := 0; b < draws; b++ {
		score := make([]float64, 7)
		denom := 0.0
		for i, s := range eligible {
			w := rng.ExpFloat64() * baseWeight[i]
			denom += w
			normal := make([]float64, 7)
			for j := range normal {
				normal[j] = rng.NormFloat64()
			}
			// Finite-summary, small-cycle uncertainty approximation. A Student-t
			// perturbation is more conservative than pretending means are exact.
			df := s.Cycles - 1
			if df > 30 {
				df = 30
			}
			chi := 0.0
			for k := 0; k < df; k++ {
				z := rng.NormFloat64()
				chi += z * z
			}
			scale := math.Sqrt(float64(df)/math.Max(chi, 1e-12)) / math.Sqrt(float64(s.Cycles))
			for j := 1; j < 7; j++ {
				noise := 0.0
				for k := 0; k <= j; k++ {
					noise += covariance[i][j][k] * normal[k]
				}
				value := math.Max(-4, math.Min(4, s.ScoreMean[j]+noise*scale))
				score[j] += w * value
			}
		}
		maximum := 0.0
		for j := 0; j < 7; j++ {
			score[j] /= denom
			if score[j] > maximum {
				maximum = score[j]
			}
			drawScores[j] = append(drawScores[j], score[j])
		}
		count := 0
		for _, v := range score {
			if math.Abs(v-maximum) < 1e-8 {
				count++
			}
		}
		for j, v := range score {
			if math.Abs(v-maximum) < 1e-8 {
				support[j] += 1 / float64(count*draws)
			}
		}
	}
	result.State = "exploratory"
	order := []int{0, 1, 2, 3, 4, 5, 6}
	sort.SliceStable(order, func(i, j int) bool { return means[order[i]] > means[order[j]] })
	positive, negative := 0, 0
	a, b := order[0], order[1]
	for _, s := range eligible {
		difference := s.ScoreMean[a] - s.ScoreMean[b]
		variance := math.Max(0, s.ScoreCov[a][a]+s.ScoreCov[b][b]-2*s.ScoreCov[a][b]) / float64(s.Cycles)
		threshold := 2*math.Sqrt(variance) + .01
		if difference > threshold {
			positive++
		}
		if difference < -threshold {
			negative++
		}
	}
	if positive > 0 && negative > 0 {
		result.State = "heterogeneous"
	}
	for j := 0; j < 7; j++ {
		sort.Float64s(drawScores[j])
		lo, hi := drawScores[j][draws/20], drawScores[j][draws*19/20]
		result.Causes[j].Support = &support[j]
		result.Causes[j].ScoreMean = &means[j]
		result.Causes[j].ScoreLow = &lo
		result.Causes[j].ScoreHigh = &hi
		result.Causes[j].Factors = factor[j]
	}
	return result
}
