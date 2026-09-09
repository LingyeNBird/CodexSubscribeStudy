package study

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSurveyUsagePatternAndLimitedDiscovery(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{})
	send := func(value any, expected int) {
		t.Helper()
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		response := surveyCall(server, "POST", "/api/survey/submissions", string(body), "192.0.2.1:80", "")
		if response.Code != expected {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	}
	payload := func(pattern any) map[string]any {
		return map[string]any{"status": []string{"风控（限流）"}, "answers": map[string][]string{"plans": {"Plus"}, "limitedDiscovery": {"容量达到上限", "其他"}}, "details": map[string]string{"limitedDiscoveryOther": "private observation"}, "usagePattern": pattern}
	}
	for _, pattern := range []any{[]int{}, make([]int, 23), make([]int, 25)} {
		send(payload(pattern), 400)
	}
	for _, bad := range []any{nil, -1, 25, 0.5, "1"} {
		pattern := make([]any, 24)
		for i := range pattern {
			pattern[i] = 0
		}
		pattern[12] = bad
		send(payload(pattern), 400)
	}
	normal := payload(make([]int, 24))
	normal["status"] = []string{"正常"}
	send(normal, 400)
	missingOther := payload(make([]int, 24))
	missingOther["answers"] = map[string][]string{"plans": {"Plus"}, "limitedDiscovery": {"服务不可用"}}
	send(missingOther, 400)
	levels := make([]int, 24)
	for i := range levels {
		levels[i] = i + 1
	}
	send(payload(levels), 201)
	send(payload(make([]int, 24)), 201)
	send(map[string]any{"status": []string{"正常"}, "answers": map[string][]string{"plans": {"Free 免费"}}}, 201)
	response := surveyCall(server, "GET", "/api/survey/statistics", "", "192.0.2.1:80", "")
	var stats surveyStatistics
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 || stats.UsagePattern.Total != 2 {
		t.Fatal("missing answers treated as zero")
	}
	for hour, mean := range stats.UsagePattern.Levels {
		if mean != float64(hour+1)/2 {
			t.Fatalf("hour %d mean %f", hour, mean)
		}
	}
	found := false
	for _, factor := range stats.Factors {
		if factor.ID == "limitedDiscovery" {
			found = true
			if factor.Groups["limited"].Total != 2 || factor.Groups["limited"].Rows[0].Count != 2 || factor.Groups["limited"].Rows[3].Count != 2 {
				t.Fatal("incorrect limited discovery distribution")
			}
		}
	}
	if !found {
		t.Fatal("missing limited discovery distribution")
	}
	for _, group := range stats.Associations {
		if group.ID == "limitedDiscovery" {
			t.Fatal("outcome-gated question produced spurious association")
		}
	}
	if strings.Contains(response.Body.String(), "private observation") {
		t.Fatal("free text leaked")
	}
}
