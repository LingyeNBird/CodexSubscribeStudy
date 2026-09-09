package study

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

type surveyRate struct {
	Events int `json:"events"`
	Total  int `json:"total"`
}
type surveyAssociation struct {
	Selected   surveyRate `json:"selected"`
	Unselected surveyRate `json:"unselected"`
	Phi        *float64   `json:"phi"`
}

func association(a, b, c, d int) surveyAssociation {
	result := surveyAssociation{Selected: surveyRate{a, a + b}, Unselected: surveyRate{c, c + d}}
	denominator := math.Sqrt(float64(a+b) * float64(c+d) * float64(a+c) * float64(b+d))
	if denominator > 0 {
		phi := (float64(a)*float64(d) - float64(b)*float64(c)) / denominator
		result.Phi = &phi
	}
	return result
}

type surveyCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}
type surveyGroup struct {
	Total int           `json:"total"`
	Rows  []surveyCount `json:"rows"`
}
type surveyFactor struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Multiple    bool                   `json:"multiple"`
	Eligibility string                 `json:"eligibility"`
	Groups      map[string]surveyGroup `json:"groups"`
}
type surveyAssociationRow struct {
	Label    string            `json:"label"`
	Degraded surveyAssociation `json:"degraded"`
	Banned   surveyAssociation `json:"banned"`
	Limited  surveyAssociation `json:"limited"`
}
type surveyAssociationGroup struct {
	ID    string                 `json:"id"`
	Title string                 `json:"title"`
	Scope string                 `json:"scope"`
	Total int                    `json:"total"`
	Rows  []surveyAssociationRow `json:"rows"`
}
type surveyStatusCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
	Tone  string `json:"tone"`
}
type surveyUsagePattern struct {
	Total  int         `json:"total"`
	Levels [24]float64 `json:"levels"`
}
type surveyStatistics struct {
	Total        int                      `json:"total"`
	Degraded     int                      `json:"degraded"`
	Banned       int                      `json:"banned"`
	Limited      int                      `json:"limited"`
	Statuses     []surveyStatusCount      `json:"statuses"`
	Normal       int                      `json:"normal"`
	Both         int                      `json:"both"`
	Factors      []surveyFactor           `json:"factors"`
	Associations []surveyAssociationGroup `json:"associations"`
	UsagePattern surveyUsagePattern       `json:"usagePattern"`
	Range        surveyStatisticsRange    `json:"range"`
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
