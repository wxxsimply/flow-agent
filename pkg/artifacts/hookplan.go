package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// HookPlan 本集钩子与分场景写作计划（hook_plan_v1）。
type HookPlan struct {
	EpisodeNo  int     `json:"episode_no"`
	HookType   string  `json:"hook_type"`
	HookLine   string  `json:"hook_line"`
	SceneCount int     `json:"scene_count"`
	Scenes     []Scene `json:"scenes"`
}

// Scene 单个写作场景。
type Scene struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Goal     string `json:"goal"`
	MaxChars int    `json:"max_chars,omitempty"`
}

// UnmarshalJSON 容忍 LLM 把 id / max_chars 输出成字符串。
func (s *Scene) UnmarshalJSON(data []byte) error {
	type alias struct {
		ID       json.RawMessage `json:"id"`
		Title    string          `json:"title"`
		Goal     string          `json:"goal"`
		MaxChars json.RawMessage `json:"max_chars,omitempty"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	s.Title = a.Title
	s.Goal = a.Goal
	id, err := flexInt(a.ID)
	if err != nil {
		return fmt.Errorf("scene.id: %w", err)
	}
	s.ID = id
	mc, err := flexInt(a.MaxChars)
	if err != nil {
		return fmt.Errorf("scene.max_chars: %w", err)
	}
	s.MaxChars = mc
	return nil
}

// flexInt 将 raw JSON 值转成 int，兼容 "1"、"scene-1"、1、null 等形态。
func flexInt(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return 0, err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, nil
		}
		// 提取末尾连续数字段（兼容 "scene-1" / "S01" / "第1场"）
		start := len(s)
		for start > 0 {
			r := s[start-1]
			if r < '0' || r > '9' {
				break
			}
			start--
		}
		digits := s[start:]
		if digits == "" {
			return 0, fmt.Errorf("no digits in %q", s)
		}
		return strconv.Atoi(digits)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, err
	}
	return n, nil
}

// PlannerOutput 模型返回的 plan 阶段 JSON 结构。
type PlannerOutput struct {
	BriefMD  string   `json:"brief_md"`
	HookPlan HookPlan `json:"hook_plan"`
}

// Validate 校验场景数量与 episode 编号。
func (h *HookPlan) Validate(episodeNo int) error {
	if h.EpisodeNo != 0 && h.EpisodeNo != episodeNo {
		return fmt.Errorf("hook_plan episode_no=%d want %d", h.EpisodeNo, episodeNo)
	}
	if h.SceneCount == 0 {
		h.SceneCount = len(h.Scenes)
	}
	if len(h.Scenes) == 0 {
		return fmt.Errorf("hook_plan: no scenes")
	}
	if h.SceneCount != len(h.Scenes) {
		return fmt.Errorf("hook_plan scene_count=%d but len(scenes)=%d", h.SceneCount, len(h.Scenes))
	}
	for i, s := range h.Scenes {
		if s.ID == 0 {
			h.Scenes[i].ID = i + 1
		}
	}
	return nil
}

// LoadHookPlan 从 run 产物目录读取 hook-plan.json。
func LoadHookPlan(path string) (*HookPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var h HookPlan
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
