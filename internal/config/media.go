package config

import (
	"fmt"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// MediaSpec 成片与出图分辨率。
type MediaSpec struct {
	AspectRatio string
	Width       int
	Height      int
	Resolution  string
	LabelZH     string
}

// MediaSpecFromCreative 根据用户选项解析横竖屏。
func MediaSpecFromCreative(c *artifacts.CreativeOptions) MediaSpec {
	orient := "portrait"
	if c != nil {
		orient = c.Orientation
	}
	switch strings.ToLower(strings.TrimSpace(orient)) {
	case "landscape", "horizontal", "16:9", "横屏", "横":
		return MediaSpec{
			AspectRatio: "16:9",
			Width:       1920,
			Height:      1080,
			Resolution:  "1920x1080",
			LabelZH:     "横屏16:9",
		}
	default:
		return MediaSpec{
			AspectRatio: "9:16",
			Width:       1080,
			Height:      1920,
			Resolution:  "1080x1920",
			LabelZH:     "竖屏9:16",
		}
	}
}

// ApplyMediaSpec 将横竖屏覆盖到 stack 出图/视频配置。
func ApplyMediaSpec(cfg StackImageConfig, spec MediaSpec) StackImageConfig {
	cfg.AspectRatio = spec.AspectRatio
	cfg.Width = spec.Width
	cfg.Height = spec.Height
	return cfg
}

// ApplyMediaSpecVideo 覆盖视频宽高比。
func ApplyMediaSpecVideo(cfg StackVideoConfig, spec MediaSpec) StackVideoConfig {
	cfg.AspectRatio = spec.AspectRatio
	return cfg
}

// StyleLabelZH 供 LLM 提示词使用的风格描述。
func StyleLabelZH(c *artifacts.CreativeOptions) string {
	anim := "2D动画"
	if c != nil && c.AnimationStyle == "3d" {
		anim = "3D动画"
	}
	theme := ThemeLabelZH(c)
	spec := MediaSpecFromCreative(c)
	return fmt.Sprintf("%s，%s，%s", anim, theme, spec.LabelZH)
}

// ThemeLabelZH 视觉主题中文名。
func ThemeLabelZH(c *artifacts.CreativeOptions) string {
	if c == nil {
		return "通用动画"
	}
	switch strings.ToLower(strings.TrimSpace(c.VisualTheme)) {
	case "generic", "default", "normal", "通用":
		return "通用动画"
	case "cinematic", "电影":
		return "电影写实"
	case "anime", "日系", "番剧":
		return "日系动漫"
	case "cartoon", "卡通":
		return "欧美卡通"
	case "ink_wash", "水墨", "ink":
		return "水墨国风"
	case "wuxia", "武侠", "古风":
		return "国风武侠"
	case "cyberpunk", "赛博朋克", "cyber":
		return "赛博朋克"
	case "ghibli", "吉卜力", "治愈":
		return "治愈手绘"
	case "noir", "悬疑":
		return "悬疑 Noir"
	case "commercial", "广告":
		return "商业广告"
	case "sketch", "素描":
		return "手绘素描"
	case "pixel", "像素":
		return "像素复古"
	case "arknights":
		return "战术插画"
	default:
		return "通用动画"
	}
}

// VisualThemePreset 前端可选视觉主题。
type VisualThemePreset struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"`
}

// ListVisualThemes 供 Web UI 展示。
func ListVisualThemes() []VisualThemePreset {
	return []VisualThemePreset{
		{ID: "generic", Label: "通用", Hint: "适配多数剧情"},
		{ID: "cinematic", Label: "电影写实", Hint: "镜头感强，偏写实"},
		{ID: "anime", Label: "日系动漫", Hint: "赛璐璐番剧感"},
		{ID: "cartoon", Label: "欧美卡通", Hint: "夸张造型，色彩明快"},
		{ID: "ink_wash", Label: "水墨国风", Hint: "留白晕染，东方意境"},
		{ID: "wuxia", Label: "国风武侠", Hint: "古装飘逸，江湖山水"},
		{ID: "cyberpunk", Label: "赛博朋克", Hint: "霓虹雨夜，高对比"},
		{ID: "ghibli", Label: "治愈手绘", Hint: "柔和自然，日常温情"},
		{ID: "noir", Label: "悬疑 Noir", Hint: "低调明暗，紧张氛围"},
		{ID: "commercial", Label: "商业广告", Hint: "干净高质，展示感强"},
		{ID: "sketch", Label: "手绘素描", Hint: "线稿质感，艺术感"},
		{ID: "pixel", Label: "像素复古", Hint: "8-bit 怀旧风"},
	}
}
