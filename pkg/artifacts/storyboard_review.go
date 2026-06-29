package artifacts

import (
	"encoding/json"
	"os"
	"strings"
)

// StoryboardReviewItem 单条审查意见。
type StoryboardReviewItem struct {
	ShotID   string `json:"shot_id,omitempty"`
	Severity string `json:"severity"` // warn | error
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// StoryboardReviewReport assemble 阶段规则审查报告。
type StoryboardReviewReport struct {
	Issues  []StoryboardReviewItem `json:"issues"`
	Summary string                 `json:"summary"`
	Fixed   int                    `json:"fixed"`
}

// ReviewStoryboardWithProps 规则审查并在有 prop-sheets 时检测跨镜物体切换。
func (s *Storyboard) ReviewStoryboardWithProps(propSheets *PropSheets) StoryboardReviewReport {
	report := s.ReviewStoryboard()
	if propSheets != nil && len(propSheets.Props) > 0 {
		report.Fixed += AlignHeldPropsToRegistry(s, propSheets)
		report.Fixed += ApplyPropRefs(s, propSheets)
		if issues := ReviewPropContinuity(s, propSheets); len(issues) > 0 {
			report.Issues = append(report.Issues, issues...)
		}
	}
	return report
}

// ReviewStoryboard 规则审查并自动修复可确定性问题（animation-craft + 物理字段）。
func (s *Storyboard) ReviewStoryboard() StoryboardReviewReport {
	report := StoryboardReviewReport{Summary: "ok"}
	if s == nil || len(s.Shots) == 0 {
		report.Summary = "empty storyboard"
		return report
	}
	fixed := s.DedupeNarrations()
	report.Fixed += fixed
	report.Fixed += s.ApplyPropLocks()

	for i := range s.Shots {
		shot := &s.Shots[i]
		if conflict, msg := PropsNarrationVisualConflict(shot.Narration, shot.VisualPrompt); conflict {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "held_props",
				Message: msg,
			})
		}
		if shot.CharacterCount > 1 {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "character_count",
				Message: "画面描述暗示多人，已标注 character_count>1",
			})
		}
		if p := strings.TrimSpace(shot.HeldProps.String()); p != "" {
			if !HeldPropsHasHandSide(p) {
				report.Issues = append(report.Issues, StoryboardReviewItem{
					ShotID: shot.ID, Severity: "warn", Field: "held_props",
					Message: "held_props 缺少左右手格式，已规范化",
				})
				NormalizeHeldProps(shot)
				report.Fixed++
			}
			if PropHandConflict(p) {
				report.Issues = append(report.Issues, StoryboardReviewItem{
					ShotID: shot.ID, Severity: "error", Field: "held_props",
					Message: "同手双物或左右手描述冲突，已按 visual_prompt 修正",
				})
				hands := ParsePropHands(shot.VisualPrompt)
				if hands.Left != "" || hands.Right != "" {
					shot.HeldProps = FlexString(FormatHeldProps(hands))
					report.Fixed++
				}
			}
		}
		if risky, msg := ActionBeatHandSwapRisk(shot.ActionBeats); risky {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "action_beats",
				Message: msg,
			})
		}
	}
	for i := range s.Shots {
		shot := &s.Shots[i]
		if len(shot.ActionBeats) < 3 {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "action_beats",
				Message: "少于 3 条 action_beats，已自动补全",
			})
			shot.ActionBeats = defaultActionBeats(*shot)
			report.Fixed++
		}
		if n := countActiveMotionBeats(shot.ActionBeats); n > 1 {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "action_beats",
				Message: "多条 action_beats 含主动作，已合并为单镜单一主动作",
			})
			if enforceSinglePrimaryAction(shot) {
				report.Fixed++
			}
		}
		if strings.TrimSpace(shot.PhysicsCues) == "" {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "physics_cues",
				Message: "缺少物理提示，已填入默认值",
			})
			shot.PhysicsCues = defaultPhysicsCues(*shot)
			report.Fixed++
		}
		if strings.TrimSpace(shot.ForbiddenPhysics) == "" {
			shot.ForbiddenPhysics = defaultForbiddenPhysics()
			report.Fixed++
		}
		if !physicsForbiddenPaired(shot.PhysicsCues, shot.ForbiddenPhysics) {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "warn", Field: "forbidden_physics",
				Message: "forbidden_physics 与 physics_cues 未成对，已补全默认反事实",
			})
			shot.ForbiddenPhysics = mergeForbiddenWithDefaults(shot.ForbiddenPhysics)
			report.Fixed++
		}
		if strings.TrimSpace(shot.Narration) == "" {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "error", Field: "narration",
				Message: "旁白为空",
			})
		} else if !NarrationComplete(shot.Narration) {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: shot.ID, Severity: "error", Field: "narration",
				Message: "旁白未以句号/问号/叹号结束，TTS 与字幕可能中途截断",
			})
		}
	}
	for i := 1; i < len(s.Shots); i++ {
		prev, curr := s.Shots[i-1], s.Shots[i]
		if SceneChanged(prev, curr) {
			continue
		}
		a := strings.TrimSpace(prev.SceneBackground)
		b := strings.TrimSpace(curr.SceneBackground)
		if a != "" && b != "" && !ScenesSimilar(a, b) {
			report.Issues = append(report.Issues, StoryboardReviewItem{
				ShotID: curr.ID, Severity: "warn", Field: "scene_background",
				Message: "同场景连续镜 scene_background 差异较大，可能影响镜间连贯",
			})
		}
	}
	if len(report.Issues) > 0 {
		report.Summary = "review completed with notes"
	}
	return report
}

func defaultActionBeats(shot Shot) []string {
	base := strings.TrimSpace(shot.VisualPrompt)
	if base == "" {
		base = "电影镜头"
	}
	return []string{
		base + "，预备姿态，重心稳定，脚贴地",
		base + "，动作进行，单一主动作，小幅位移",
		base + "，收势姿态，与预备一致，肢体无穿透",
	}
}

func defaultPhysicsCues(shot Shot) string {
	parts := []string{"重力向下", "角色足底与地面接触"}
	if v := strings.TrimSpace(shot.VisualPrompt); v != "" {
		parts = append(parts, "动作与场景符合空间逻辑")
	}
	return strings.Join(parts, "，")
}

func defaultForbiddenPhysics() string {
	return "穿模，物体穿透，无支撑悬浮，瞬间瞬移，多余肢体，违反重力，道具换手，道具消失，武器变形"
}

func mergeForbiddenWithDefaults(forbidden string) string {
	base := defaultForbiddenPhysics()
	f := strings.TrimSpace(forbidden)
	if f == "" {
		return base
	}
	return f + "，" + base
}

func physicsForbiddenPaired(cues, forbidden string) bool {
	cues = strings.TrimSpace(cues)
	forbidden = strings.TrimSpace(forbidden)
	if cues == "" || forbidden == "" {
		return false
	}
	pairs := []struct{ cue, forbid string }{
		{"贴地", "悬浮"}, {"重力", "违反重力"}, {"接触", "穿模"},
		{"支撑", "悬浮"}, {"足底", "滑步"}, {"因果", "未触即动"},
	}
	for _, p := range pairs {
		if strings.Contains(cues, p.cue) && strings.Contains(forbidden, p.forbid) {
			return true
		}
	}
	return len([]rune(forbidden)) >= 12
}

var activeMotionKeywords = []string{
	"走", "跑", "跳", "转身", "抬手", "推门", "倒", "挥", "踢", "抓", "握", "迈", "俯身", "站起",
}

func containsActiveMotion(s string) bool {
	for _, k := range activeMotionKeywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func countActiveMotionBeats(beats []string) int {
	n := 0
	for _, b := range beats {
		if containsActiveMotion(b) {
			n++
		}
	}
	return n
}

func enforceSinglePrimaryAction(shot *Shot) bool {
	if shot == nil || len(shot.ActionBeats) < 3 {
		return false
	}
	prep := strings.TrimSpace(shot.ActionBeats[0])
	if prep == "" || containsActiveMotion(prep) {
		prep = "预备姿态，重心稳定，足部贴地支撑"
	}
	primary := strings.TrimSpace(shot.ActionBeats[1])
	if primary == "" || !containsActiveMotion(primary) {
		base := strings.TrimSpace(shot.VisualPrompt)
		if base == "" {
			base = "角色"
		}
		primary = base + "，单一主动作，幅度极小"
	}
	settle := "收势静止，肢体姿态与预备一致，无穿透"
	shot.ActionBeats = []string{prep, primary, settle}
	return true
}

// SaveStoryboardReview 写入审查报告。
func SaveStoryboardReview(path string, report StoryboardReviewReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
