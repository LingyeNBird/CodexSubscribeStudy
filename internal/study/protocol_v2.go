package study

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/LingyeNBird/CodexSubscribeStudy/protocol"
)

const ProtocolV2 = "codex-cost-study/2"
const MethodV2 = "pooled-profile/raw-only-2"
const MaxBodyV2 = 262144

var QualityV2 = []string{"missing_snapshot", "capture_gap", "missing_components", "unknown_control", "invalid_fact", "reset_or_saturation", "zero_progress", "external_usage_uncontrolled", "archived_source", "resource_limit"}
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type SummaryV2 struct {
	Requests      int64            `json:"requests"`
	GPT6Requests  int64            `json:"gpt6_requests"`
	OtherRequests int64            `json:"other_requests"`
	RawUSD        float64          `json:"raw_usd"`
	GPT6RawUSD    float64          `json:"gpt6_raw_usd"`
	QuotaPoints   float64          `json:"quota_points"`
	Intervals     int64            `json:"intervals"`
	Groups        int64            `json:"groups"`
	Contrasts     int64            `json:"contrasts"`
	GatewayOnly   bool             `json:"gateway_only"`
	Quality       map[string]int64 `json:"quality"`
	LogEvidence   [][]float64      `json:"log_evidence"`
	GPT6Quota     []float64        `json:"gpt6_quota"`
	Information   [][]float64      `json:"information"`
}

type ReportV2 struct {
	Protocol     string     `json:"protocol"`
	StudyID      string     `json:"study_id"`
	Method       string     `json:"method"`
	MethodDigest string     `json:"method_digest"`
	PublicKey    string     `json:"public_key"`
	Revision     uint64     `json:"revision"`
	BatchID      string     `json:"batch_id,omitempty"`
	Summary      *SummaryV2 `json:"summary,omitempty"`
}

type gridPoint struct {
	Factors [4]float64
	Family  int
}

var pointsV2, nullV2 = makeGridV2()

func makeGridV2() ([]gridPoint, int) {
	values := []float64{.5, 1, 1.5, 1.75, 2, 3}
	set := map[[4]float64]bool{}
	for _, a := range values {
		for _, b := range values {
			for _, c := range values {
				for _, d := range values {
					set[[4]float64{a, b, c, d}] = true
				}
			}
		}
	}
	for _, v := range []float64{1.25, 1.8, 2.5} {
		set[[4]float64{v, v, v, v}] = true
		for j := 0; j < 4; j++ {
			p := [4]float64{1, 1, 1, 1}
			p[j] = v
			set[p] = true
		}
	}
	result := make([]gridPoint, 0, len(set))
	for p := range set {
		result = append(result, gridPoint{Factors: p})
	}
	sort.Slice(result, func(i, j int) bool {
		for k := 0; k < 4; k++ {
			if result[i].Factors[k] != result[j].Factors[k] {
				return result[i].Factors[k] < result[j].Factors[k]
			}
		}
		return false
	})
	null := 0
	for i := range result {
		p := result[i].Factors
		count, index := 0, 0
		for j, v := range p {
			if v != 1 {
				count++
				index = j
			}
		}
		family := 6
		if count == 0 {
			family = 0
			null = i
		} else if p[0] == p[1] && p[1] == p[2] && p[2] == p[3] {
			family = 1
		} else if count == 1 {
			family = []int{5, 3, 2, 4}[index]
		}
		result[i].Family = family
	}
	return result, null
}
func GridDigestV2() string {
	h := sha256.New()
	var data [33]byte
	for _, p := range pointsV2 {
		for j, v := range p.Factors {
			binary.LittleEndian.PutUint64(data[j*8:], math.Float64bits(v))
		}
		data[32] = byte(p.Family)
		_, _ = h.Write(data[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Validate exact field spelling/presence as well as duplicate keys. encoding/json
// alone accepts case-insensitive aliases; signed scientific payloads must not.
func exactKeys(raw json.RawMessage, keys []string) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || len(object) != len(keys) {
		return errors.New("fields")
	}
	for _, key := range keys {
		value, ok := object[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errors.New("missing field")
		}
	}
	return nil
}
func DecodeV2(body []byte, signature, path string) (ReportV2, error) {
	var r ReportV2
	if len(body) > MaxBodyV2 || !utf8.Valid(body) {
		return r, errors.New("body")
	}
	if path != "/api/v2/reports" {
		return r, errors.New("path")
	}
	scan := json.NewDecoder(bytes.NewReader(body))
	scan.UseNumber()
	if err := rejectDuplicates(scan, 0); err != nil {
		return r, err
	}
	if _, err := scan.Token(); err != io.EOF {
		return r, errors.New("trailing")
	}
	keys := []string{"protocol", "study_id", "method", "method_digest", "public_key", "revision", "batch_id", "summary"}
	if err := exactKeys(body, keys); err != nil {
		return r, err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil {
		return r, errors.New("schema")
	}
	if r.Protocol != ProtocolV2 || r.StudyID != StudyID || r.Method != MethodV2 || r.MethodDigest != protocol.DigestV2() {
		return r, errors.New("version")
	}
	if r.Revision == 0 || r.Revision > 9007199254740991 {
		return r, errors.New("revision")
	}
	key, err := base64.StdEncoding.Strict().DecodeString(r.PublicKey)
	if err != nil || len(key) != 32 || base64.StdEncoding.EncodeToString(key) != r.PublicKey {
		return r, errors.New("identity")
	}
	sig, err := base64.StdEncoding.Strict().DecodeString(signature)
	if err != nil || len(sig) != 64 {
		return r, errors.New("signature")
	}
	if !ed25519.Verify(key, append([]byte("CodexSubscribeStudy/2\nPOST\n"+path+"\n"), body...), sig) {
		return r, errors.New("signature")
	}
	if !uuidPattern.MatchString(r.BatchID) || r.Summary == nil {
		return r, errors.New("batch")
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	skeys := []string{"requests", "gpt6_requests", "other_requests", "raw_usd", "gpt6_raw_usd", "quota_points", "intervals", "groups", "contrasts", "gateway_only", "quality", "log_evidence", "gpt6_quota", "information"}
	if err := exactKeys(raw["summary"], skeys); err != nil {
		return r, err
	}
	return r, r.Summary.Validate()
}

func (s *SummaryV2) Validate() error {
	if s.Requests < 0 || s.Requests > 1e12 || s.GPT6Requests < 0 || s.OtherRequests < 0 || s.GPT6Requests > s.Requests || s.OtherRequests > s.Requests || s.GPT6Requests+s.OtherRequests != s.Requests {
		return errors.New("counts")
	}
	if !finite(s.RawUSD, 0, 1e15) || !finite(s.GPT6RawUSD, 0, s.RawUSD+1e-8) || !finite(s.QuotaPoints, 0, 1e9) || s.Intervals < 0 || s.Intervals > 1e7 || s.Groups < 0 || s.Groups > s.Intervals || s.Contrasts != s.Intervals-s.Groups {
		return errors.New("totals")
	}
	if len(s.Quality) != len(QualityV2) {
		return errors.New("quality")
	}
	for _, key := range QualityV2 {
		v, ok := s.Quality[key]
		if !ok || v < 0 || v > 1e12 {
			return errors.New("quality")
		}
	}
	if len(s.LogEvidence) != 3 || len(s.GPT6Quota) != len(pointsV2) || len(s.Information) != 4 {
		return errors.New("dimensions")
	}
	for _, curve := range s.LogEvidence {
		if len(curve) != len(pointsV2) || math.Abs(curve[nullV2]) > 1e-7 {
			return errors.New("curve")
		}
		for _, v := range curve {
			if !finite(v, -1e10, 1e10) {
				return errors.New("curve")
			}
		}
	}
	for _, v := range s.GPT6Quota {
		if !finite(v, 0, s.QuotaPoints+1e-7) {
			return errors.New("quota")
		}
	}
	for i, row := range s.Information {
		if len(row) != 4 {
			return errors.New("information")
		}
		for j, v := range row {
			if !finite(v, -1e16, 1e16) || len(s.Information[j]) != 4 || math.Abs(v-s.Information[j][i]) > 1e-7 {
				return errors.New("information")
			}
		}
	}
	eigen := eigenvalues4(s.Information)
	for _, v := range eigen {
		if v < -1e-6 {
			return errors.New("information PSD")
		}
	}
	if s.Intervals == 0 {
		if s.QuotaPoints != 0 {
			return errors.New("empty intervals")
		}
	}
	if s.Intervals > 0 && s.Groups == 0 {
		return errors.New("groups")
	}
	if s.Intervals > s.Groups*32 {
		return errors.New("group size")
	}
	if s.Contrasts == 0 {
		for _, curve := range s.LogEvidence {
			for _, v := range curve {
				if v != 0 {
					return errors.New("unidentified group")
				}
			}
		}
	}
	if s.Contrasts == 0 {
		for _, row := range s.Information {
			for _, v := range row {
				if v != 0 {
					return errors.New("unidentified information")
				}
			}
		}
	}
	if s.GPT6Requests == 0 {
		for _, v := range s.GPT6Quota {
			if v != 0 {
				return errors.New("target quota without target requests")
			}
		}
		for _, curve := range s.LogEvidence {
			for _, v := range curve {
				if v != 0 {
					return errors.New("target evidence without target requests")
				}
			}
		}
	}
	return nil
}

// Jacobi eigenvalues for a symmetric 4x4 information matrix, no new dependency.
func eigenvalues4(input [][]float64) []float64 {
	var a [4][4]float64
	for i := 0; i < 4; i++ {
		copy(a[i][:], input[i])
	}
	for step := 0; step < 80; step++ {
		p, q, big := 0, 1, 0.
		for i := 0; i < 4; i++ {
			for j := i + 1; j < 4; j++ {
				if math.Abs(a[i][j]) > big {
					p, q, big = i, j, math.Abs(a[i][j])
				}
			}
		}
		if big < 1e-10 {
			break
		}
		angle := .5 * math.Atan2(2*a[p][q], a[q][q]-a[p][p])
		c, s := math.Cos(angle), math.Sin(angle)
		app, aqq, apq := a[p][p], a[q][q], a[p][q]
		for k := 0; k < 4; k++ {
			if k == p || k == q {
				continue
			}
			v, w := a[k][p], a[k][q]
			a[k][p] = c*v - s*w
			a[p][k] = a[k][p]
			a[k][q] = s*v + c*w
			a[q][k] = a[k][q]
		}
		a[p][p] = c*c*app - 2*c*s*apq + s*s*aqq
		a[q][q] = s*s*app + 2*c*s*apq + c*c*aqq
		a[p][q] = 0
		a[q][p] = 0
	}
	result := []float64{a[0][0], a[1][1], a[2][2], a[3][3]}
	sort.Float64s(result)
	return result
}
