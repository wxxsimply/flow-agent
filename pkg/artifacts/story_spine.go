package artifacts

import (
	"encoding/json"
	"os"
)

// StorySpine Expand 阶段产物。
type StorySpine struct {
	Title              string      `json:"title"`
	Logline            string      `json:"logline"`
	Acts               []StoryAct  `json:"acts"`
	Characters         []Character `json:"characters"`
	Tone               string      `json:"tone"`
	Mood               string      `json:"mood"`        // BGM 情绪：悬疑/温暖/史诗…
	EmotionArc         string      `json:"emotion_arc"` // 情绪走向，供 BGM 与分镜
	AnimationStyle     string      `json:"animation_style,omitempty"`
	TargetDurationSec  int         `json:"target_duration_sec"`
}

// StoryAct 幕结构。
type StoryAct struct {
	Act     int      `json:"act"`
	Summary string   `json:"summary"`
	Beats   []string `json:"beats"`
}

// Character 角色定妆。
type Character struct {
	Name       string `json:"name"`
	Appearance string `json:"appearance"`
}

// LoadStorySpine 读取 story-spine.json。
func LoadStorySpine(path string) (*StorySpine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s StorySpine
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
