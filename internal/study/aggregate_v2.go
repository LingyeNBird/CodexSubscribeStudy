package study

import (
	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
	"math"
	"sort"
	"time"
)

type TotalsV2 struct {
	Contributors  int     `json:"contributors"`
	Batches       int     `json:"batches"`
	Requests      int64   `json:"requests"`
	GPT6Requests  int64   `json:"gpt6_requests"`
	OtherRequests int64   `json:"other_requests"`
	RawUSD        float64 `json:"raw_usd"`
	GPT6RawUSD    float64 `json:"gpt6_raw_usd"`
	QuotaPoints   float64 `json:"quota_points"`
	Intervals     int64   `json:"intervals"`
	Contrasts     int64   `json:"contrasts"`
}
type ParameterV2 struct {
	Name string  `json:"name"`
	Mean float64 `json:"mean"`
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}
type ResultV2 struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	State              string           `json:"state"`
	Method             string           `json:"method"`
	MethodDigest       string           `json:"method_digest"`
	UpdatedAt          string           `json:"updated_at"`
	Totals             TotalsV2         `json:"totals"`
	Causes             []Cause          `json:"causes"`
	Parameters         []ParameterV2    `json:"parameters"`
	Sensitivity        [][]float64      `json:"drift_support"`
	GPT6Quota          *ParameterV2     `json:"gpt6_quota"`
	InformationRank    int              `json:"information_rank"`
	MaximumSourceShare float64          `json:"maximum_source_information_share"`
	Quality            map[string]int64 `json:"quality"`
	Warnings           []string         `json:"warnings"`
	Legacy             Totals           `json:"legacy_archive"`
	ConfidenceMeaning  string           `json:"confidence_meaning"`
}
type accumulatorV2 struct {
	result  ResultV2
	curves  [][]float64
	quota   []float64
	info    [][]float64
	sources map[string]float64
	updated int64
}

func newAccumulatorV2() *accumulatorV2 {
	n := len(pointsV2)
	a := &accumulatorV2{curves: make([][]float64, 3), quota: make([]float64, n), info: make([][]float64, 4), sources: map[string]float64{}}
	for i := range a.curves {
		a.curves[i] = make([]float64, n)
	}
	for i := range a.info {
		a.info[i] = make([]float64, 4)
	}
	a.result = ResultV2{ID: StudyID, Title: "GPT-6 额度异常归因", State: "no_data", Method: MethodV2, MethodDigest: protocol.DigestV2(), Quality: map[string]int64{}, Warnings: []string{}, Parameters: []ParameterV2{}, Sensitivity: [][]float64{},
		ConfidenceMeaning: "共同参数证据相加后只加一次先验；为固定计费前提和容量工作模型下的条件支持度，不是官方真实机制的已校准概率。"}
	return a
}
func (a *accumulatorV2) add(identity string, s SummaryV2, hour int64) error {
	t := &a.result.Totals
	t.Batches++
	t.Requests += s.Requests
	t.GPT6Requests += s.GPT6Requests
	t.OtherRequests += s.OtherRequests
	t.RawUSD += s.RawUSD
	t.GPT6RawUSD += s.GPT6RawUSD
	t.QuotaPoints += s.QuotaPoints
	t.Intervals += s.Intervals
	t.Contrasts += s.Contrasts
	if hour > a.updated {
		a.updated = hour
	}
	a.sources[identity] += 0
	for i := 0; i < 4; i++ {
		a.sources[identity] += math.Max(0, s.Information[i][i])
		for j := 0; j < 4; j++ {
			a.info[i][j] += s.Information[i][j]
		}
	}
	for j := range pointsV2 {
		a.quota[j] += s.GPT6Quota[j]
		for i := 0; i < 3; i++ {
			a.curves[i][j] += s.LogEvidence[i][j]
		}
	}
	for key, v := range s.Quality {
		a.result.Quality[key] += v
	}
	return nil
}
func weightsV2(curve []float64) []float64 {
	counts := [7]int{}
	for _, p := range pointsV2 {
		counts[p.Family]++
	}
	weights := make([]float64, len(curve))
	max := -math.MaxFloat64
	for i, v := range curve {
		weights[i] = v - math.Log(float64(counts[pointsV2[i].Family])) - math.Log(7)
		if weights[i] > max {
			max = weights[i]
		}
	}
	sum := 0.
	for i := range weights {
		weights[i] = math.Exp(weights[i] - max)
		sum += weights[i]
	}
	for i := range weights {
		weights[i] /= sum
	}
	return weights
}
func quantileV2(values, weights []float64, level float64) float64 {
	ids := make([]int, len(values))
	for i := range ids {
		ids[i] = i
	}
	sort.Slice(ids, func(i, j int) bool { return values[ids[i]] < values[ids[j]] })
	sum := 0.
	for _, i := range ids {
		sum += weights[i]
		if sum >= level {
			return values[i]
		}
	}
	return values[ids[len(ids)-1]]
}
func causesV2(curve []float64, enabled bool) ([]Cause, []float64) {
	weights := weightsV2(curve)
	result := make([]Cause, 7)
	mass := make([]float64, 7)
	for j := 0; j < 7; j++ {
		result[j] = Cause{ID: Families[j], Label: Labels[j], Factors: []float64{1, 1, 1, 1}}
	}
	means := make([][4]float64, 7)
	for i, p := range pointsV2 {
		mass[p.Family] += weights[i]
		for j := 0; j < 4; j++ {
			means[p.Family][j] += weights[i] * p.Factors[j]
		}
	}
	for i := range result {
		if enabled {
			v := mass[i]
			result[i].Support = &v
		}
		if mass[i] > 0 {
			for j := 0; j < 4; j++ {
				result[i].Factors[j] = means[i][j] / mass[i]
			}
		}
	}
	return result, mass
}
func hasSignal(curve []float64) bool {
	lo, hi := curve[0], curve[0]
	for _, v := range curve {
		lo = math.Min(lo, v)
		hi = math.Max(hi, v)
	}
	return hi-lo > 1e-7
}
func argmax(v []float64) int {
	i := 0
	for j := range v {
		if v[j] > v[i] {
			i = j
		}
	}
	return i
}
func (a *accumulatorV2) finish() ResultV2 {
	r := &a.result
	r.Totals.Contributors = len(a.sources)
	if a.updated > 0 {
		r.UpdatedAt = time.Unix(a.updated*3600, 0).UTC().Format("2006-01-02")
	}
	primary := hasSignal(a.curves[1])
	r.Causes, _ = causesV2(a.curves[1], primary)
	if r.Totals.Batches > 0 {
		r.State = "uninformative"
	}
	if primary {
		r.State = "conditional"
	}
	eigen := eigenvalues4(a.info)
	scale := math.Max(1e-8, eigen[3]*1e-6)
	for _, v := range eigen {
		if v > scale {
			r.InformationRank++
		}
	}
	information := 0.
	for _, v := range a.sources {
		information += v
		r.MaximumSourceShare = math.Max(r.MaximumSourceShare, v)
	}
	if information > 0 {
		r.MaximumSourceShare /= information
	}
	r.Warnings = append(r.Warnings, "固定FAST=2、GPT长上下文=1，其他模型价格假定正确。容量变化与模型选择同步或存在系统性漏记时，仍可能高支持误判。")
	if r.InformationRank < 4 {
		r.Warnings = append(r.Warnings, "全局证据仍有不可辨识方向；平坦证据不会因来源多而凭空变成信息。所有贡献仍已接收。")
	}
	if r.MaximumSourceShare > .5 {
		r.Warnings = append(r.Warnings, "一个安装贡献了超过一半的信息量；来源独立性和集中程度需额外审阅。")
	}
	if r.Quality["external_usage_uncontrolled"] > 0 {
		r.Warnings = append(r.Warnings, "部分贡献未确认用量覆盖，已保留并纳入；结论仍以没有未记录消耗为条件。")
	}
	if primary {
		w := weightsV2(a.curves[1])
		names := []string{"input", "cache_creation", "cache_read", "output"}
		for j := 0; j < 4; j++ {
			values := make([]float64, len(w))
			mean := 0.
			for i, p := range pointsV2 {
				values[i] = p.Factors[j]
				mean += w[i] * values[i]
			}
			r.Parameters = append(r.Parameters, ParameterV2{names[j], mean, quantileV2(values, w, .05), quantileV2(values, w, .95)})
		}
		mainWinner := -1
		for index, curve := range a.curves {
			_, mass := causesV2(curve, true)
			r.Sensitivity = append(r.Sensitivity, mass)
			if index == 1 {
				mainWinner = argmax(mass)
			}
		}
		for _, mass := range r.Sensitivity {
			if argmax(mass) != mainWinner {
				r.State = "sensitive"
				r.Warnings = append(r.Warnings, "不同容量漂移强度产生不同领先解释；请同时查看敏感性结果。")
				break
			}
		}
		mean := 0.
		for i, v := range a.quota {
			mean += w[i] * v
		}
		r.GPT6Quota = &ParameterV2{"gpt6_quota_points", mean, quantileV2(a.quota, w, .05), quantileV2(a.quota, w, .95)}
	}
	return *r
}
func (s *Store) AggregateV2() (ResultV2, error) {
	accumulator := newAccumulatorV2()
	if err := s.WalkV2(accumulator.add); err != nil {
		return ResultV2{}, err
	}
	result := accumulator.finish()
	// Historical v1 is retained and shown separately, never added to v2 counts
	// or support: the same original history may have been reprocessed as v2.
	legacy, _, err := s.Snapshot()
	if err != nil {
		return ResultV2{}, err
	}
	for _, v := range legacy {
		result.Legacy.Contributors++
		result.Legacy.Requests += v.Requests
		result.Legacy.RawUSD += v.RawUSD
		if v.Eligible {
			result.Legacy.EligibleContributors++
		}
	}
	return result, nil
}
