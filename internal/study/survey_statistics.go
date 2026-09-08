package study

import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
	"math"
	"slices"
	"strconv"
	"strings"
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
type surveyStatistics struct {
	Total              int                      `json:"total"`
	Degraded           int                      `json:"degraded"`
	Banned             int                      `json:"banned"`
	Limited            int                      `json:"limited"`
	Statuses           []surveyStatusCount      `json:"statuses"`
	Normal             int                      `json:"normal"`
	Both               int                      `json:"both"`
	Factors            []surveyFactor           `json:"factors"`
	Associations       []surveyAssociationGroup `json:"associations"`
	OutcomeAssociation surveyAssociation        `json:"outcomeAssociation"`
}

func (q SurveySubmission) normalized() map[string][]string {
	result := make(map[string][]string, len(q.Answers)+8)
	for key, value := range q.Answers {
		result[key] = value
	}
	add := func(key, value string) { result[key] = []string{value} }
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
func eventPrecision(value string) []string {
	if value == "" {
		return nil
	}
	label := "只知道日期"
	if value == "unknown" {
		label = "我不清楚具体时间"
	} else if len(value) == 16 {
		label = "精确到分钟"
	}
	return []string{label}
}
func (s *Server) surveyStatistics() (surveyStatistics, error) {
	result := surveyStatistics{Factors: []surveyFactor{}, Associations: []surveyAssociationGroup{}}
	result.Statuses = []surveyStatusCount{
		{Label: "正常", Tone: "mint"}, {Label: "降智", Tone: "violet"},
		{Label: "封号", Tone: "peach"}, {Label: "降智、封号", Tone: "peach"},
		{Label: "风控（限流）", Tone: "sun"}, {Label: "降智、风控（限流）", Tone: "sun"},
		{Label: "封号、风控（限流）", Tone: "sun"}, {Label: "降智、封号、风控（限流）", Tone: "sun"},
	}
	type normalized struct {
		q       SurveySubmission
		answers map[string][]string
	}
	records := []normalized{}
	err := s.store.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(surveyBucket).ForEach(func(_, body []byte) error {
			var q SurveySubmission
			if err := json.Unmarshal(body, &q); err != nil {
				return err
			}
			records = append(records, normalized{q, q.normalized()})
			result.Total++
			if q.degraded() {
				result.Degraded++
			}
			if q.banned() {
				result.Banned++
			}
			if q.degraded() && q.banned() {
				result.Both++
			}
			if slices.Contains(q.Status, "正常") {
				result.Normal++
			}
			mask := 0
			if q.degraded() {
				mask |= 1
			}
			if q.banned() {
				mask |= 2
			}
			if q.limited() {
				mask |= 4
				result.Limited++
			}
			result.Statuses[mask].Count++
			return nil
		})
	})
	if err != nil {
		return result, err
	}
	for _, definition := range surveyCatalog.Definitions {
		key := definition.Key
		options := surveyCatalog.Choices[key]
		factor := surveyFactor{ID: key, Title: definition.Title, Description: definition.Description, Multiple: definition.Multiple, Eligibility: definition.Eligibility, Groups: map[string]surveyGroup{}}
		if factor.Eligibility == "" {
			factor.Eligibility = "分母为所选异常状态中回答此题的问卷；漏答不计入。"
		}
		for _, outcome := range []string{"degraded", "banned", "limited"} {
			group := surveyGroup{Rows: make([]surveyCount, len(options))}
			for i, label := range options {
				group.Rows[i].Label = label
			}
			for _, record := range records {
				applies := record.q.degraded()
				if outcome == "banned" {
					applies = record.q.banned()
				}
				if outcome == "limited" {
					applies = record.q.limited()
				}
				if !applies {
					continue
				}
				values := record.answers[key]
				if key == "eventTime" {
					if outcome == "limited" {
						continue
					}
					event := "degradationTime"
					if outcome == "banned" {
						event = "banTime"
					}
					values = eventPrecision(record.q.Details[event])
				}
				if len(values) == 0 {
					continue
				}
				group.Total++
				for i, label := range options {
					if slices.Contains(values, label) {
						group.Rows[i].Count++
					}
				}
			}
			factor.Groups[outcome] = group
		}
		result.Factors = append(result.Factors, factor)
		if slices.Contains([]string{"models", "discovery", "eventTime"}, key) {
			continue
		}
		group := surveyAssociationGroup{ID: key, Title: definition.Title, Scope: definition.Eligibility, Rows: []surveyAssociationRow{}}
		if group.Scope == "" {
			group.Scope = "所有回答此题的问卷。"
		}
		for _, record := range records {
			if len(record.answers[key]) > 0 {
				group.Total++
			}
		}
		for _, label := range options {
			counts := [3][4]int{}
			for _, record := range records {
				if len(record.answers[key]) == 0 {
					continue
				}
				selected := slices.Contains(record.answers[key], label)
				for i, event := range []bool{record.q.degraded(), record.q.banned(), record.q.limited()} {
					index := 0
					if !selected {
						index = 2
					}
					if !event {
						index++
					}
					counts[i][index]++
				}
			}
			d, b, l := counts[0], counts[1], counts[2]
			group.Rows = append(group.Rows, surveyAssociationRow{label, association(d[0], d[1], d[2], d[3]), association(b[0], b[1], b[2], b[3]), association(l[0], l[1], l[2], l[3])})
		}
		result.Associations = append(result.Associations, group)
	}
	result.OutcomeAssociation = association(result.Both, result.Degraded-result.Both, result.Banned-result.Both, result.Total-result.Degraded-result.Banned+result.Both)
	return result, nil
}
