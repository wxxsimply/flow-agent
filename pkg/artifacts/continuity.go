package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ContinuityIssue 单条一致性问题。
type ContinuityIssue struct {
	Severity   string `json:"severity"` // critical | warning
	Category   string `json:"category"`
	SceneID    int    `json:"scene_id,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ContinuityReport continuity_report_v1。
type ContinuityReport struct {
	EpisodeNo      int               `json:"episode_no"`
	CriticalCount  int               `json:"critical_count"`
	WarningCount   int               `json:"warning_count"`
	Issues         []ContinuityIssue `json:"issues"`
	Passed         bool              `json:"passed"`
	CharacterPatch map[string]any    `json:"character_state_patch,omitempty"`
}

// Recount 根据 issues 重算计数与 passed。
func (r *ContinuityReport) Recount() {
	r.CriticalCount = 0
	r.WarningCount = 0
	for _, iss := range r.Issues {
		switch iss.Severity {
		case "critical":
			r.CriticalCount++
		case "warning":
			r.WarningCount++
		}
	}
	r.Passed = r.CriticalCount == 0
}

// CriticalSceneIDs 返回需重写的场景 id（去重）。
func (r *ContinuityReport) CriticalSceneIDs() []int {
	return r.SceneIDsForRewrite(nil)
}

// SceneIDsForRewrite 根据 critical 问题与 hook-plan 场景列表决定需重写的场景。
// 若建议涉及下一幕（如「Scene 5 开头」），且下一幕存在，一并重写。
func (r *ContinuityReport) SceneIDsForRewrite(validScenes map[int]bool) []int {
	seen := map[int]bool{}
	var ids []int
	add := func(id int) {
		if id <= 0 || seen[id] {
			return
		}
		if validScenes != nil && !validScenes[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, iss := range r.Issues {
		if iss.Severity != "critical" || iss.SceneID <= 0 {
			continue
		}
		add(iss.SceneID)
		combined := iss.Message + " " + iss.Suggestion
		if MentionsNextScene(combined) {
			add(iss.SceneID + 1)
		}
	}
	return ids
}

// MentionsNextScene 判断修复建议是否涉及下一幕。
func MentionsNextScene(s string) bool {
	return strings.Contains(s, "Scene 5") || strings.Contains(s, "场景 5") ||
		strings.Contains(s, "下一幕") || strings.Contains(s, "下一场") ||
		strings.Contains(s, "Scene 4结尾") || strings.Contains(s, "下集")
}

// LoadContinuityReport 读取 continuity-report.json。
func LoadContinuityReport(path string) (*ContinuityReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r ContinuityReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.CriticalCount == 0 && r.WarningCount == 0 && len(r.Issues) > 0 {
		r.Recount()
	}
	return &r, nil
}

// Save 写入 JSON 文件。
func (r *ContinuityReport) Save(path string) error {
	r.Recount()
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Validate 基本字段校验。
func (r *ContinuityReport) Validate(episodeNo int) error {
	if r.EpisodeNo != 0 && r.EpisodeNo != episodeNo {
		return fmt.Errorf("episode_no mismatch: %d vs %d", r.EpisodeNo, episodeNo)
	}
	r.EpisodeNo = episodeNo
	r.Recount()
	return nil
}
