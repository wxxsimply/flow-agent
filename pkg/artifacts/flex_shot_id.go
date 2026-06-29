package artifacts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// flexShotID 兼容 LLM 输出数字或字符串镜号（1 → s01）。
func flexShotID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		i, err := num.Int64()
		if err != nil {
			return "", fmt.Errorf("flexShotID number: %w", err)
		}
		if i <= 0 {
			return "", fmt.Errorf("flexShotID: invalid id %d", i)
		}
		return fmt.Sprintf("s%02d", i), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", nil
		}
		if i, err := strconv.Atoi(s); err == nil && i > 0 {
			return fmt.Sprintf("s%02d", i), nil
		}
		return s, nil
	}
	return "", fmt.Errorf("flexShotID: unsupported JSON %s", string(raw))
}

// UnmarshalJSON 容忍 id 为数字或字符串。
func (s *Shot) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if idRaw, ok := raw["id"]; ok {
		id, err := flexShotID(idRaw)
		if err != nil {
			return err
		}
		idJSON, err := json.Marshal(id)
		if err != nil {
			return err
		}
		raw["id"] = idJSON
	}
	fixed, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	type alias Shot
	return json.Unmarshal(fixed, (*alias)(s))
}

// NormalizeShotIDs 填充空镜号并规范为 s01 格式。
func (s *Storyboard) NormalizeShotIDs() {
	if s == nil {
		return
	}
	for i := range s.Shots {
		id := strings.TrimSpace(s.Shots[i].ID)
		if id == "" {
			s.Shots[i].ID = fmt.Sprintf("s%02d", i+1)
			continue
		}
		if n, err := strconv.Atoi(id); err == nil && n > 0 {
			s.Shots[i].ID = fmt.Sprintf("s%02d", n)
		}
	}
}
