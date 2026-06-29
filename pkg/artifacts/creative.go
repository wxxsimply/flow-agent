package artifacts

import (
	"encoding/json"
	"os"
	"strings"
)

// UserShotInput 用户逐镜输入（director 模式）。
type UserShotInput struct {
	Narration  string `json:"narration"`
	VisualDesc string `json:"visual_desc,omitempty"`
	ShotSize   string `json:"shot_size,omitempty"` // wide | medium | close
}

// CreativeOptions 用户创作参数（CLI → 全流程）。
type CreativeOptions struct {
	InputMode         string          `json:"input_mode,omitempty"` // director（第一镜→3千字→自动分镜）| auto
	AnimationStyle    string          `json:"animation_style"`      // 2d | 3d
	Orientation       string          `json:"orientation,omitempty"`
	VisualTheme       string          `json:"visual_theme,omitempty"`
	Plot              string          `json:"plot,omitempty"`        // director：第一镜文本
	OpeningShot       string          `json:"opening_shot,omitempty"` // 同 plot，优先使用
	Shots             []UserShotInput `json:"shots,omitempty"`       // 兼容：仅取第一镜
	BGMMode           string          `json:"bgm_mode"`
	BGMPath           string          `json:"bgm_path,omitempty"`
	NarratorVoice     string          `json:"narrator_voice,omitempty"`
	TargetDurationSec int             `json:"target_duration_sec,omitempty"`
}

// IsDirector 是否为逐镜导演模式。
func (c *CreativeOptions) IsDirector() bool {
	if c == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(c.InputMode), "director")
}

// IsAuto 是否为 LLM 自动分镜模式。
func (c *CreativeOptions) IsAuto() bool {
	if c == nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.InputMode), "auto")
}

// Normalize 校验并填默认值。
func (c *CreativeOptions) Normalize() {
	if c == nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(c.AnimationStyle)) {
	case "3d", "3D", "三维":
		c.AnimationStyle = "3d"
	default:
		c.AnimationStyle = "2d"
	}

	switch strings.ToLower(strings.TrimSpace(c.Orientation)) {
	case "landscape", "horizontal", "16:9", "横屏", "横":
		c.Orientation = "landscape"
	default:
		c.Orientation = "portrait"
	}

	switch strings.ToLower(strings.TrimSpace(c.VisualTheme)) {
	case "generic", "default", "normal", "通用":
		c.VisualTheme = "generic"
	case "ink_wash", "水墨", "ink":
		c.VisualTheme = "ink_wash"
	case "cyberpunk", "赛博朋克", "cyber":
		c.VisualTheme = "cyberpunk"
	case "ghibli", "吉卜力", "治愈":
		c.VisualTheme = "ghibli"
	case "wuxia", "武侠", "古风":
		c.VisualTheme = "wuxia"
	case "arknights", "明日方舟":
		c.VisualTheme = "arknights"
	default:
		if strings.TrimSpace(c.VisualTheme) != "" {
			c.VisualTheme = strings.ToLower(strings.TrimSpace(c.VisualTheme))
		} else {
			c.VisualTheme = "generic"
		}
	}

	switch strings.ToLower(strings.TrimSpace(c.NarratorVoice)) {
	case "female":
		c.NarratorVoice = "narrator_female"
	default:
		id := strings.ToLower(strings.TrimSpace(c.NarratorVoice))
		if id == "" {
			c.NarratorVoice = "epic_male"
		} else {
			c.NarratorVoice = id
		}
	}

	if c.TargetDurationSec <= 0 {
		c.TargetDurationSec = 150
	}
	if c.TargetDurationSec < 15 {
		c.TargetDurationSec = 15
	}
	if c.TargetDurationSec > 180 {
		c.TargetDurationSec = 180
	}

	switch strings.ToLower(strings.TrimSpace(c.InputMode)) {
	case "auto":
		c.InputMode = "auto"
	default:
		if len(c.Shots) > 0 || strings.TrimSpace(c.Plot) != "" || strings.TrimSpace(c.OpeningShot) != "" {
			c.InputMode = "director"
		} else {
			c.InputMode = "auto"
		}
	}

	for i := range c.Shots {
		c.Shots[i].ShotSize = NormalizeShotSize(c.Shots[i].ShotSize)
	}
	if strings.TrimSpace(c.OpeningShot) == "" && strings.TrimSpace(c.Plot) != "" {
		c.OpeningShot = strings.TrimSpace(c.Plot)
	}
	if strings.TrimSpace(c.Plot) == "" && strings.TrimSpace(c.OpeningShot) != "" {
		c.Plot = strings.TrimSpace(c.OpeningShot)
	}
	if strings.TrimSpace(c.OpeningShot) == "" && len(c.Shots) > 0 {
		in := c.Shots[0]
		c.OpeningShot = strings.TrimSpace(in.Narration)
		if v := strings.TrimSpace(in.VisualDesc); v != "" {
			c.OpeningShot += "。画面：" + v
		}
		c.Plot = c.OpeningShot
	}

	if strings.TrimSpace(c.BGMPath) != "" {
		c.BGMMode = "path"
		return
	}
	switch strings.ToLower(strings.TrimSpace(c.BGMMode)) {
	case "off", "none", "false":
		c.BGMMode = "off"
	default:
		c.BGMMode = "auto"
	}
}

// LoadCreativeOptions 读取 creative-options.json。
func LoadCreativeOptions(path string) (*CreativeOptions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c CreativeOptions
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.Normalize()
	return &c, nil
}
