package study

import (
	"encoding/json"
	"errors"
)

// normalizeLegacySurvey upgrades historical bbolt records without writing to the source file.
func normalizeLegacySurvey(body []byte) ([]byte, error) {
	var record map[string]json.RawMessage
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, err
	}
	changed := false
	if raw := record["status"]; len(raw) > 0 && raw[0] == '"' {
		var old string
		if err := json.Unmarshal(raw, &old); err != nil {
			return nil, err
		}
		var statuses []string
		switch old {
		case "正常", "降智", "封号":
			statuses = []string{old}
		case "降智并封号":
			statuses = []string{"降智", "封号"}
		default:
			return nil, errors.New("invalid stored survey status")
		}
		record["status"], _ = json.Marshal(statuses)
		changed = true
	}
	var answers map[string]json.RawMessage
	if raw := record["answers"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &answers); err != nil {
			return nil, err
		}
	}
	answersChanged := false
	if raw := answers["official"]; len(raw) > 0 {
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for index, value := range values {
			if value == "Codex CI" {
				values[index] = "Codex CLI"
				answersChanged = true
			}
		}
		if answersChanged {
			answers["official"], _ = json.Marshal(values)
		}
	}
	if raw, exists := answers["ciMode"]; exists {
		answers["cliMode"] = raw
		delete(answers, "ciMode")
		answersChanged = true
	}
	if raw := answers["models"]; len(raw) > 0 {
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for index, value := range values {
			if value == "GPT-5.6 Sora" {
				values[index] = "GPT-5.6 Sol"
				answersChanged = true
			}
		}
		if answersChanged {
			answers["models"], _ = json.Marshal(values)
		}
	}
	if raw := answers["network"]; len(raw) > 0 {
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for index, value := range values {
			if value == "宽带" {
				values[index] = "家宽"
				answersChanged = true
			}
		}
		if answersChanged {
			answers["network"], _ = json.Marshal(values)
		}
	}
	if raw, exists := answers["quality"]; exists {
		record["legacyIPQuality"] = raw
		delete(answers, "quality")
		answersChanged = true
	}
	if answersChanged {
		record["answers"], _ = json.Marshal(answers)
		changed = true
	}
	if !changed {
		return body, nil
	}
	return json.Marshal(record)
}
