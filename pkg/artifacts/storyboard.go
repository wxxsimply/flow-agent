package artifacts

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	MinAIVideoBudgetShots = 4
	MaxAIVideoBudgetShots = 6
	DurationToleranceSec  = 3.0
)

// Storyboard 分镜契约（schema: storyboard_v1），Produce 阶段输入。
type Storyboard struct {
	EpisodeNo         int     `json:"episode_no"`
	TargetDurationSec float64 `json:"target_duration_sec"`
	TotalNarrationSec float64 `json:"total_narration_sec"`
	Shots             []Shot  `json:"shots"`
}

// Shot 单个镜头。
type Shot struct {
	ID            string  `json:"id"`
	DurationSec   float64 `json:"duration_sec"`
	VisualType    string  `json:"visual_type"` // ken_burns | ai_video | ...
	AIVideoBudget bool    `json:"ai_video_budget,omitempty"`
	VisualPrompt  string  `json:"visual_prompt"`
	Narration     string  `json:"narration"`
	Subtitle      string  `json:"subtitle"`
	SFX           FlexString `json:"sfx,omitempty"`
	KeyframeRef   string   `json:"keyframe_ref,omitempty"`
	ActionBeats   []string `json:"action_beats,omitempty"`
	ShotSize      string   `json:"shot_size,omitempty"`   // wide | medium | close
	CameraAngle   string   `json:"camera_angle,omitempty"`
	NarrativeBeat string   `json:"narrative_beat,omitempty"`
	BriefExcerpt  string   `json:"brief_excerpt,omitempty"`
	UserSource       bool     `json:"user_source,omitempty"` // 用户逐镜输入（未扩写）
	Expanded         bool     `json:"expanded,omitempty"`    // 经镜头语言扩写，用于出图/视频
	PhysicsCues      string   `json:"physics_cues,omitempty"`      // 重力、支撑、接触等物理提示
	ForbiddenPhysics string   `json:"forbidden_physics,omitempty"` // 须避免的物理违规（用于 negative prompt）
	Tier             string   `json:"tier,omitempty"`              // hero | standard（分级制片）
	CharacterCount   int      `json:"character_count,omitempty"`   // 镜头内主角人数，默认 1
	HeldProps        FlexString `json:"held_props,omitempty"` // 道具锁定，如「仅一把剑」（兼容 LLM 输出数组）
	PropRefs         []string   `json:"prop_refs,omitempty"`  // 绑定 prop-sheets 中的物体 ID
	SceneBackground  string     `json:"scene_background,omitempty"` // 场景/环境，用于镜间是否硬切
}

// TotalDurationSec 各镜头时长之和，用于 duration_ok 门禁。
func (s *Storyboard) TotalDurationSec() float64 {
	var sum float64
	for _, shot := range s.Shots {
		sum += shot.DurationSec
	}
	return sum
}

// SyncTotalNarrationSec 将 total_narration_sec 同步为各镜时长之和。
func (s *Storyboard) SyncTotalNarrationSec() {
	s.TotalNarrationSec = s.TotalDurationSec()
}

// Validate 校验 storyboard_v1 契约与标准档约束。
func (s *Storyboard) Validate(episodeNo, targetSec int, policy ...StoryboardPolicy) error {
	pol := DefaultAIVideoPolicy()
	if len(policy) > 0 {
		pol = policy[0]
	}
	if pol.MinShots <= 0 {
		pol.MinShots = 4
	}
	if pol.MaxShots <= 0 {
		pol.MaxShots = 12
	}
	if s.EpisodeNo != 0 && s.EpisodeNo != episodeNo {
		return fmt.Errorf("episode_no=%d want %d", s.EpisodeNo, episodeNo)
	}
	target := float64(targetSec)
	if pol.RelaxDurationTarget {
		if s.TargetDurationSec <= 0 {
			s.TargetDurationSec = s.TotalDurationSec()
		}
	} else if s.TargetDurationSec == 0 {
		s.TargetDurationSec = target
	} else if math.Abs(s.TargetDurationSec-target) > 0.5 {
		return fmt.Errorf("target_duration_sec=%.1f want %.1f", s.TargetDurationSec, target)
	}
	if len(s.Shots) == 0 {
		return fmt.Errorf("shots: empty")
	}
	if len(s.Shots) < pol.MinShots || len(s.Shots) > pol.MaxShots {
		return fmt.Errorf("shots: count %d, want %d-%d", len(s.Shots), pol.MinShots, pol.MaxShots)
	}

	aiCount := 0
	for i, shot := range s.Shots {
		if strings.TrimSpace(shot.ID) == "" {
			return fmt.Errorf("shots[%d]: id required", i)
		}
		if shot.DurationSec <= 0 {
			return fmt.Errorf("shots[%d]: duration_sec must be > 0", i)
		}
		if strings.TrimSpace(shot.VisualType) == "" {
			return fmt.Errorf("shots[%d]: visual_type required", i)
		}
		if strings.TrimSpace(shot.Narration) == "" {
			return fmt.Errorf("shots[%d]: narration required", i)
		}
		if strings.TrimSpace(shot.Subtitle) == "" && strings.TrimSpace(shot.Narration) == "" {
			return fmt.Errorf("shots[%d]: subtitle or narration required", i)
		}
		if strings.TrimSpace(shot.VisualPrompt) == "" {
			return fmt.Errorf("shots[%d]: visual_prompt required", i)
		}
		if shot.AIVideoBudget {
			aiCount++
			if shot.VisualType != "ai_video" {
				return fmt.Errorf("shots[%d]: ai_video_budget=true requires visual_type=ai_video", i)
			}
		} else if shot.VisualType == "ai_video" {
			return fmt.Errorf("shots[%d]: visual_type=ai_video requires ai_video_budget=true", i)
		}
		if pol.AllShotsAIVideo && shot.VisualType == "ken_burns" {
			return fmt.Errorf("shots[%d]: video-native requires visual_type=ai_video", i)
		}
	}

	if pol.AllShotsAIVideo {
		if aiCount != len(s.Shots) {
			return fmt.Errorf("video-native: all %d shots must have ai_video_budget=true (got %d)", len(s.Shots), aiCount)
		}
	} else if aiCount < pol.MinAIVideo || aiCount > pol.MaxAIVideo {
		return fmt.Errorf("ai_video_budget shots=%d, want %d-%d", aiCount, pol.MinAIVideo, pol.MaxAIVideo)
	}

	total := s.TotalDurationSec()
	if !pol.RelaxDurationTarget {
		tol := DurationToleranceSec
		if pol.DurationToleranceSec > 0 {
			tol = pol.DurationToleranceSec
		}
		if math.Abs(total-target) > tol {
			return fmt.Errorf("total duration %.1fs, want %.1f±%.0fs", total, target, tol)
		}
	}

	s.SyncTotalNarrationSec()
	if math.Abs(s.TotalNarrationSec-total) > 1 {
		return fmt.Errorf("total_narration_sec %.1f inconsistent with shot sum %.1f", s.TotalNarrationSec, total)
	}
	s.EpisodeNo = episodeNo
	return nil
}

// CapShots 截断分镜数量（极速 produce 用）。
func (s *Storyboard) CapShots(maxShots int) {
	if s == nil || maxShots <= 0 || len(s.Shots) <= maxShots {
		return
	}
	s.Shots = s.Shots[:maxShots]
	s.SyncTotalNarrationSec()
}

// NormalizeDurations 按比例缩放各镜时长，使总和落在 [target-tol, target+tol]。
func (s *Storyboard) NormalizeDurations(targetSec int) {
	target := float64(targetSec)
	tol := DurationToleranceSec
	total := s.TotalDurationSec()
	if total <= 0 || math.Abs(total-target) <= tol {
		s.SyncTotalNarrationSec()
		return
	}

	scale := target / total
	for i := range s.Shots {
		s.Shots[i].DurationSec = math.Max(1, math.Round(s.Shots[i].DurationSec*scale*10)/10)
	}
	total = s.TotalDurationSec()
	diff := target - total
	if math.Abs(diff) <= tol {
		s.SyncTotalNarrationSec()
		return
	}
	// 微调最后一镜吸收误差
	if len(s.Shots) > 0 {
		last := len(s.Shots) - 1
		s.Shots[last].DurationSec = math.Max(1, s.Shots[last].DurationSec+diff)
	}
	s.SyncTotalNarrationSec()
}

// NarrationComplete 旁白是否以完整句结束。
func NarrationComplete(narration string) bool {
	n := strings.TrimSpace(narration)
	if n == "" {
		return false
	}
	r := []rune(n)
	last := r[len(r)-1]
	switch last {
	case '。', '！', '？', '…', '.', '!', '?':
		return true
	default:
		return false
	}
}

// IncompleteNarrationShots 返回旁白未完整结束的镜头 id。
func (s *Storyboard) IncompleteNarrationShots() []string {
	if s == nil {
		return nil
	}
	var out []string
	for _, shot := range s.Shots {
		if !NarrationComplete(shot.Narration) {
			out = append(out, shot.ID)
		}
	}
	return out
}

// DefaultSecPerRune 旁白时长估计：约 4 字/秒 → 0.25s/字。
const DefaultSecPerRune = 0.25

// NarrationRuneCount 统计旁白字数（不含空白）。
func NarrationRuneCount(narration string) int {
	n := 0
	for _, r := range narration {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			n++
		}
	}
	return n
}

// AlignDurationsFromNarration 按旁白字数比例分配各镜 duration_sec（director/TTS 驱动）。
func (s *Storyboard) AlignDurationsFromNarration(secPerRune float64) {
	if s == nil || len(s.Shots) == 0 {
		return
	}
	if secPerRune <= 0 {
		secPerRune = DefaultSecPerRune
	}
	totalRunes := 0
	for _, shot := range s.Shots {
		totalRunes += NarrationRuneCount(shot.Narration)
	}
	if totalRunes <= 0 {
		per := math.Max(1, s.TotalDurationSec()/float64(len(s.Shots)))
		if per <= 0 {
			per = 5
		}
		for i := range s.Shots {
			s.Shots[i].DurationSec = per
		}
	} else {
		for i := range s.Shots {
			runes := NarrationRuneCount(s.Shots[i].Narration)
			if runes <= 0 {
				runes = 1
			}
			s.Shots[i].DurationSec = math.Max(1, math.Round(float64(runes)*secPerRune*10)/10)
		}
	}
	s.TargetDurationSec = s.TotalDurationSec()
	s.SyncTotalNarrationSec()
}

// Save 写入 storyboard.json。
func (s *Storyboard) Save(path string) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadStoryboard 从文件读取 storyboard.json。
func LoadStoryboard(path string) (*Storyboard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sb Storyboard
	if err := json.Unmarshal(data, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}
