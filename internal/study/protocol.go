package study

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"unicode/utf8"

	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
)

const Protocol = "codex-cost-study/1"
const StudyID = "gpt6-components"
const Method = "log-capacity-loco/1"
const MaxBody = 32768

var Families = []string{"unchanged", "global", "cache_read", "cache_creation", "output", "input", "mixed"}
var Labels = []string{"无需额外倍率", "整体倍率", "缓存读倍率", "缓存创建倍率", "输出倍率", "输入倍率", "混合倍率"}
var Exclusions = []string{"missing_snapshot_time", "snapshot_conflict", "excluded_observation", "reset_or_saturation", "insufficient_progress", "capture_gap", "invalid_capture", "missing_components", "nonstandard_request", "other_model", "cost_mismatch", "insufficient_cycle", "resource_limit"}
var statuses = map[string]bool{"insufficient_data": true, "unidentifiable": true, "model_mismatch": true, "drift_sensitive": true, "exploratory": true, "external_usage_uncontrolled": true}

// No free-text, per-request/interval data, IP, account or participant fields.
type Summary struct {
	WindowDays       int              `json:"window_days"`
	Requests         int64            `json:"requests"`
	BaselineRequests int64            `json:"baseline_requests"`
	GPT6Requests     int64            `json:"gpt6_requests"`
	RawUSD           float64          `json:"raw_usd"`
	GPT6RawUSD       float64          `json:"gpt6_raw_usd"`
	QuotaPoints      float64          `json:"quota_points"`
	Cycles           int              `json:"cycles"`
	Blocks           int              `json:"blocks"`
	Eligible         bool             `json:"eligible"`
	GatewayOnly      bool             `json:"gateway_only"`
	Status           string           `json:"status"`
	Identifiable     []bool           `json:"identifiable"`
	DesignRank       int              `json:"design_rank"`
	Exclusions       map[string]int64 `json:"exclusions"`
	ScoreMean        []float64        `json:"score_mean"`
	ScoreCov         [][]float64      `json:"score_cov"`
	Support          []float64        `json:"support"`
	Factors          [][]float64      `json:"factor_estimates"`
}

type Report struct {
	Protocol     string   `json:"protocol"`
	StudyID      string   `json:"study_id"`
	Method       string   `json:"method"`
	MethodDigest string   `json:"method_digest"`
	PublicKey    string   `json:"public_key"`
	Revision     uint64   `json:"revision"`
	Summary      *Summary `json:"summary,omitempty"`
}

// rejectDuplicates prevents ambiguous signed JSON interpretation, including
// duplicate keys nested in a summary. It also bounds JSON nesting depth.
func rejectDuplicates(dec *json.Decoder, depth int) error {
	if depth > 8 {
		return errors.New("nesting")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate")
			}
			seen[name] = true
			if err = rejectDuplicates(dec, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			if err = rejectDuplicates(dec, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("delimiter")
	}
	_, err = dec.Token()
	return err
}

func Decode(body []byte, signature, path string) (Report, error) {
	var report Report
	if len(body) > MaxBody || !utf8.Valid(body) {
		return report, errors.New("body")
	}
	scan := json.NewDecoder(bytes.NewReader(body))
	scan.UseNumber()
	if err := rejectDuplicates(scan, 0); err != nil {
		return report, err
	}
	if _, err := scan.Token(); err != io.EOF {
		return report, errors.New("trailing JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return report, errors.New("schema")
	}
	if report.Protocol != Protocol || report.StudyID != StudyID || report.Method != Method || report.MethodDigest != protocol.Digest() {
		return report, errors.New("version")
	}
	if report.Revision == 0 || report.Revision > 9007199254740991 {
		return report, errors.New("revision")
	}
	key, err := base64.StdEncoding.Strict().DecodeString(report.PublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize || base64.StdEncoding.EncodeToString(key) != report.PublicKey {
		return report, errors.New("identity")
	}
	sig, err := base64.StdEncoding.Strict().DecodeString(signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return report, errors.New("signature")
	}
	signed := append([]byte("CodexSubscribeStudy/1\nPOST\n"+path+"\n"), body...)
	if !ed25519.Verify(ed25519.PublicKey(key), signed, sig) {
		return report, errors.New("signature")
	}
	if path == "/api/v1/withdraw" {
		if report.Summary != nil {
			return report, errors.New("withdrawal body")
		}
		return report, nil
	}
	if report.Summary == nil {
		return report, errors.New("summary required")
	}
	return report, report.Summary.Validate()
}

func finite(v, lo, hi float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= lo && v <= hi }
func (s *Summary) Validate() error {
	if s.WindowDays != 90 || s.Requests < 200 || s.Requests > 1e9 || s.BaselineRequests < 0 || s.GPT6Requests < 0 || s.BaselineRequests > s.Requests || s.GPT6Requests > s.Requests || s.BaselineRequests+s.GPT6Requests != s.Requests {
		return errors.New("counts")
	}
	if !finite(s.RawUSD, 0, 1e12) || !finite(s.GPT6RawUSD, 0, s.RawUSD) || !finite(s.QuotaPoints, 0, 140000) || s.Cycles < 1 || s.Cycles > 1400 || s.Blocks < s.Cycles*8 || s.Blocks > 50000 || s.DesignRank < 0 || s.DesignRank > 4 {
		return errors.New("totals")
	}
	if s.RawUSD != math.Trunc(s.RawUSD) || s.GPT6RawUSD != math.Trunc(s.GPT6RawUSD) {
		return errors.New("cost rounding")
	}
	if !statuses[s.Status] || len(s.Identifiable) != 4 || len(s.ScoreMean) != 7 || len(s.ScoreCov) != 7 || len(s.Support) != 7 || len(s.Factors) != 7 {
		return errors.New("dimensions")
	}
	if s.Eligible != (s.Status == "exploratory") {
		return errors.New("eligibility")
	}
	if s.Eligible && (!s.GatewayOnly || s.Cycles < 2 || s.Blocks < 24 || s.BaselineRequests < 50 || s.GPT6Requests < 50 || s.DesignRank == 0) {
		return errors.New("quality")
	}
	if len(s.Exclusions) != len(Exclusions) {
		return errors.New("exclusions")
	}
	for _, key := range Exclusions {
		v, ok := s.Exclusions[key]
		if !ok || v < 0 || v > 1e9 {
			return errors.New("exclusions")
		}
	}
	sum := 0.0
	for i := 0; i < 7; i++ {
		if !finite(s.ScoreMean[i], -4, 4) || !finite(s.Support[i], 0, 1) || len(s.ScoreCov[i]) != 7 || len(s.Factors[i]) != 4 {
			return errors.New("scores")
		}
		sum += s.Support[i]
		for j, v := range s.ScoreCov[i] {
			if !finite(v, -64, 64) {
				return errors.New("covariance")
			}
			if len(s.ScoreCov[j]) != 7 || math.Abs(v-s.ScoreCov[j][i]) > 1e-7 {
				return errors.New("symmetry")
			}
		}
		for _, f := range s.Factors[i] {
			if !finite(f, .5, 3) {
				return errors.New("factors")
			}
		}
	}
	for _, v := range s.ScoreCov[0] {
		if math.Abs(v) > 1e-7 {
			return errors.New("baseline covariance")
		}
	}
	if math.Abs(s.ScoreMean[0]) > 1e-8 {
		return errors.New("baseline")
	}
	if (s.Eligible && math.Abs(sum-1) > 1e-5) || (!s.Eligible && sum != 0) {
		return errors.New("support")
	}
	if _, err := cholesky(s.ScoreCov); err != nil {
		return err
	}
	for _, v := range s.Factors[0] {
		if v != 1 {
			return errors.New("null factors")
		}
	}
	for _, v := range s.Factors[1] {
		if math.Abs(v-s.Factors[1][0]) > .0001 {
			return errors.New("global factors")
		}
	}
	for family, active := range map[int]int{2: 2, 3: 1, 4: 3, 5: 0} {
		for j, v := range s.Factors[family] {
			if j != active && v != 1 {
				return errors.New("single factors")
			}
		}
	}
	return nil
}

// PSD covariance with a tiny fixed tolerance for eight-decimal transmission.
func cholesky(a [][]float64) ([][]float64, error) {
	l := make([][]float64, 7)
	for i := range l {
		l[i] = make([]float64, 7)
	}
	for i := 0; i < 7; i++ {
		for j := 0; j <= i; j++ {
			x := a[i][j]
			if i == j {
				x += 1e-6
			}
			for k := 0; k < j; k++ {
				x -= l[i][k] * l[j][k]
			}
			if i == j {
				if x <= 0 {
					return nil, fmt.Errorf("non-PSD covariance")
				}
				l[i][j] = math.Sqrt(x)
			} else {
				l[i][j] = x / l[j][j]
			}
		}
	}
	return l, nil
}
