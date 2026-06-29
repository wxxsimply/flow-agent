package artifacts

import (
	"encoding/json"
	"os"
)

// ShotLanguageExpand 镜头语言扩写结果（约 3000 字 + 自动分镜）。
type ShotLanguageExpand struct {
	OpeningShot       string              `json:"opening_shot,omitempty"`
	ShotLanguageBrief string              `json:"shot_language_brief"` // 完整镜头语言文稿，目标约 3000 字
	StoryBackground   string              `json:"story_background"`
	Mood              string              `json:"mood,omitempty"`
	Tone              string              `json:"tone,omitempty"`
	Shots             []ExpandedShotInput `json:"shots"`
}

// ExpandedShotInput LLM 扩写后的单镜（用于生成 storyboard）。
type ExpandedShotInput struct {
	ID               string   `json:"id"`
	ShotSize         string   `json:"shot_size,omitempty"`
	CameraAngle      string   `json:"camera_angle,omitempty"`
	NarrativeBeat    string   `json:"narrative_beat,omitempty"`
	BriefExcerpt     string   `json:"brief_excerpt,omitempty"`
	SceneBackground  string   `json:"scene_background,omitempty"`
	CharacterMotion  string   `json:"character_motion,omitempty"`
	Dialogue         string   `json:"dialogue,omitempty"`
	MicroExpression  string   `json:"micro_expression,omitempty"`
	ActionBehavior   string   `json:"action_behavior,omitempty"`
	Narration        string   `json:"narration"`
	VisualPrompt     string   `json:"visual_prompt"`
	DurationSec      float64  `json:"duration_sec,omitempty"`
	ActionBeats      []string `json:"action_beats,omitempty"`
	PhysicsCues      string   `json:"physics_cues,omitempty"`
	ForbiddenPhysics string   `json:"forbidden_physics,omitempty"`
	CharacterCount   int      `json:"character_count,omitempty"`
	HeldProps        FlexString `json:"held_props,omitempty"`
}

// Save 写入 shot-language-expand.json。
func (e *ShotLanguageExpand) Save(path string) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadShotLanguageExpand 读取扩写结果。
func LoadShotLanguageExpand(path string) (*ShotLanguageExpand, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e ShotLanguageExpand
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
