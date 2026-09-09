package study

import (
	"slices"
	"strconv"
	"strings"
	"time"
)

const surveySummaryVersion = 2

type surveyOptionSummary struct {
	Label  string `json:"label"`
	Counts [8]int `json:"counts"`
}

type surveyFactorSummary struct {
	ID                string                `json:"id"`
	Title             string                `json:"title"`
	Description       string                `json:"description"`
	Multiple          bool                  `json:"multiple"`
	DistributionScope string                `json:"distributionScope"`
	AssociationScope  string                `json:"associationScope"`
	Comparable        bool                  `json:"comparable"`
	Applicable        [8]int                `json:"applicable"`
	Options           []surveyOptionSummary `json:"options"`
}

type surveyUsageSummary struct {
	Total int     `json:"total"`
	Sums  [24]int `json:"sums"`
}

type surveySummary struct {
	Version       int                   `json:"version"`
	CatalogDigest string                `json:"catalogDigest"`
	Statuses      [8]int                `json:"statuses"`
	Factors       []surveyFactorSummary `json:"factors"`
	UsagePattern  surveyUsageSummary    `json:"usagePattern"`
	Range         surveyStatisticsRange `json:"range"`
}

type surveyStatisticsRange struct {
	FirstSubmissionID uint64     `json:"firstSubmissionId"`
	LastSubmissionID  uint64     `json:"lastSubmissionId"`
	FirstSubmittedAt  *time.Time `json:"firstSubmittedAt"`
	LastSubmittedAt   *time.Time `json:"lastSubmittedAt"`
	UnknownTimeCount  int        `json:"unknownTimeCount"`
	ComputedAt        time.Time  `json:"computedAt"`
}

func (q SurveySubmission) normalized() map[string][]string {
	result := make(map[string][]string, len(q.Answers)+8)
	for key, value := range q.Answers {
		result[key] = value
	}
	add := func(key, value string) { result[key] = []string{value} }
	if q.IPRisk != nil {
		index := 5
		for i, limit := range []int{15, 25, 40, 50, 70} {
			if *q.IPRisk < limit {
				index = i
				break
			}
		}
		add("ipRisk", surveyCatalog.Choices["ipRisk"][index])
	}
	for _, key := range []string{"country", "exitCountry"} {
		code := q.Details[key]
		if code == "" {
			continue
		}
		name := map[string]string{"PH": "菲律宾", "JP": "日本", "US": "美国", "BO": "玻利维亚", "unknown": "我不知道"}[code]
		if name == "" {
			name = "其他国家或地区"
		}
		add(key, name)
	}
	if raw := q.Details["duration"]; raw != "" {
		days, _ := strconv.ParseFloat(raw, 64)
		days *= map[string]float64{"天": 1, "小时": 1.0 / 24, "星期": 7, "月": 30, "年": 365}[q.Details["durationUnit"]]
		index := 5
		for i, limit := range []float64{1, 7, 30, 90, 365} {
			if days < limit {
				index = i
				break
			}
		}
		add("duration", surveyCatalog.Choices["duration"][index])
	}
	if raw := q.Details["people"]; raw != "" {
		number, _ := strconv.Atoi(raw)
		index := 4
		for i, limit := range []int{2, 5, 10, 20} {
			if number <= limit {
				index = i
				break
			}
		}
		add("people", surveyCatalog.Choices["people"][index])
	}
	if raw := q.Details["concurrency"]; raw != "" {
		index := 6
		if raw != "unknown" {
			number, _ := strconv.Atoi(raw)
			index = 5
			for i, limit := range []int{0, 1, 3, 5, 10} {
				if number <= limit {
					index = i
					break
				}
			}
		}
		add("concurrency", surveyCatalog.Choices["concurrency"][index])
	}
	for key, value := range q.Details {
		if strings.HasPrefix(key, "toolMode:") && !slices.Contains(result["thirdMode"], value) {
			result["thirdMode"] = append(result["thirdMode"], value)
		}
	}
	return result
}
