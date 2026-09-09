package study

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed survey_catalog.json
var surveyCatalogJSON []byte

type surveyDefinition struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Multiple    bool   `json:"multiple"`
	Eligibility string `json:"eligibility"`
}

type surveyCatalogData struct {
	Choices     map[string][]string `json:"choices"`
	Definitions []surveyDefinition  `json:"definitions"`
}

var surveyCatalog = func() surveyCatalogData {
	var value surveyCatalogData
	if err := json.Unmarshal(surveyCatalogJSON, &value); err != nil {
		panic(err)
	}
	return value
}()

type SurveySubmission struct {
	Status       []string            `json:"status"`
	Answers      map[string][]string `json:"answers"`
	Details      map[string]string   `json:"details"`
	UsagePattern []*int              `json:"usagePattern,omitempty"`
	IPRisk       *int                `json:"ipRisk,omitempty"`
}

func (q SurveySubmission) has(key, value string) bool { return slices.Contains(q.Answers[key], value) }
func (q SurveySubmission) degraded() bool             { return slices.Contains(q.Status, "降智") }
func (q SurveySubmission) banned() bool               { return slices.Contains(q.Status, "封号") }
func (q SurveySubmission) limited() bool              { return slices.Contains(q.Status, "风控（限流）") }

func validateSurvey(q SurveySubmission) error {
	hasConnection := q.has("usage", "直登") || q.has("usage", "反代")
	invalid := errors.New("invalid questionnaire")
	if len(q.Status) == 0 || len(q.Status) > 3 || len(q.Answers["plans"]) != 1 {
		return invalid
	}
	seenStatuses := map[string]bool{}
	for _, status := range q.Status {
		if !slices.Contains([]string{"正常", "降智", "封号", "风控（限流）"}, status) || seenStatuses[status] || (status == "正常" && len(q.Status) != 1) {
			return invalid
		}
		seenStatuses[status] = true
	}
	if q.IPRisk != nil && (*q.IPRisk < 0 || *q.IPRisk > 100 || !hasConnection) {
		return invalid
	}
	if q.UsagePattern != nil {
		if len(q.UsagePattern) != 24 {
			return invalid
		}
		for _, value := range q.UsagePattern {
			if value == nil || *value < 0 || *value > 24 {
				return invalid
			}
		}
	}
	for key, values := range q.Answers {
		options, ok := surveyCatalog.Choices[key]
		if !ok || key == "duration" || key == "people" || key == "concurrency" || key == "country" || key == "exitCountry" || key == "thirdMode" || key == "ipRisk" {
			return invalid
		}
		multi := slices.Contains([]string{"models", "tools", "usage", "official", "discovery", "limitedDiscovery"}, key)
		if len(values) == 0 || len(values) > len(options) || (!multi && len(values) != 1) {
			return invalid
		}
		seen := map[string]bool{}
		for _, value := range values {
			if !slices.Contains(options, value) || seen[value] {
				return invalid
			}
			seen[value] = true
		}
		if (key == "models" || key == "discovery") && !q.degraded() {
			return invalid
		}
		if key == "limitedDiscovery" && !q.limited() {
			return invalid
		}
		if key == "activation" && q.has("plans", "Free 免费") {
			return invalid
		}
		if key == "proxy" && !q.has("usage", "反代") {
			return invalid
		}
		if slices.Contains([]string{"network", "ipStability"}, key) && !hasConnection {
			return invalid
		}
		if key == "desktopMode" && !q.has("official", "Codex Desktop") {
			return invalid
		}
		if key == "ciMode" && !q.has("official", "Codex CI") {
			return invalid
		}
	}
	for key, value := range q.Details {
		if value == "" || len(value) > 2000 || strings.TrimSpace(value) != value {
			return invalid
		}
		switch key {
		case "limitedDiscoveryOther":
			if !q.limited() || !q.has("limitedDiscovery", "其他") {
				return invalid
			}
		case "discoveryOther":
			if !q.has("discovery", "其他") {
				return invalid
			}
		case "proxyOther":
			if !q.has("proxy", "其他") {
				return invalid
			}
		case "thirdPartyOther":
			if !q.has("tools", "其他") {
				return invalid
			}
		case "country", "exitCountry":
			if key == "exitCountry" && !hasConnection {
				return invalid
			}
			if key == "exitCountry" && value == "unknown" {
				continue
			}
			if !strings.Contains(" AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW ", " "+value+" ") || len(value) != 2 {
				return invalid
			}
		case "duration":
			number, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > 1000000 {
				return invalid
			}
			if !slices.Contains([]string{"天", "小时", "星期", "月", "年"}, q.Details["durationUnit"]) {
				return invalid
			}
		case "durationUnit":
			if q.Details["duration"] == "" {
				return invalid
			}
		case "people", "concurrency":
			if key == "people" && !q.has("shared", "是") {
				return invalid
			}
			if key == "concurrency" && value == "unknown" {
				continue
			}
			number, err := strconv.Atoi(value)
			if err != nil || number < 0 || number > 1000000 || (key == "people" && number < 2) {
				return invalid
			}
		case "degradationTime", "banTime":
			if key == "degradationTime" && !q.degraded() || key == "banTime" && !q.banned() {
				return invalid
			}
			if value == "unknown" {
				continue
			}
			layout := "2006-01-02"
			if len(value) == 16 {
				layout = "2006-01-02T15:04"
			}
			if _, err := time.Parse(layout, value); err != nil {
				return invalid
			}
		default:
			if !strings.HasPrefix(key, "toolMode:") {
				return invalid
			}
			name := strings.TrimPrefix(key, "toolMode:")
			if name == "Claude Code" || !q.has("tools", name) || !slices.Contains([]string{"直登（OAuth）", "反代"}, value) {
				return invalid
			}
		}
	}
	return nil
}

func (s *Server) serveSurvey(w http.ResponseWriter, r *http.Request) {
	expected := http.MethodGet
	if r.URL.Path == "/api/survey/submissions" {
		expected = http.MethodPost
	}
	if r.Method != expected {
		fail(w, 405, "method")
		return
	}
	if r.URL.Path == "/api/survey/statistics" {
		value, err := s.surveyStatistics()
		if err != nil {
			fail(w, 503, "storage_unavailable")
			return
		}
		respond(w, 200, value)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		fail(w, 415, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32768)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, 413, "body_too_large")
		return
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var q SurveySubmission
	if err = decoder.Decode(&q); err != nil {
		fail(w, 400, "invalid_submission")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || validateSurvey(q) != nil {
		fail(w, 400, "invalid_submission")
		return
	}
	if err = s.store.putSurvey(q); err != nil {
		fail(w, 503, "storage_unavailable")
		return
	}
	respond(w, 201, map[string]bool{"accepted": true})
}
