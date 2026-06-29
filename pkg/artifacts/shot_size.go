package artifacts

import "strings"

const (
	ShotSizeWide   = "wide"
	ShotSizeMedium = "medium"
	ShotSizeClose  = "close"
)

// NormalizeShotSize 标准化景别：wide | medium | close。
func NormalizeShotSize(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "wide", "long", "远景", "远":
		return ShotSizeWide
	case "close", "closeup", "特写", "近", "近景":
		return ShotSizeClose
	default:
		return ShotSizeMedium
	}
}

// ShotSizePromptHint 注入文生图 prompt 的景别描述。
func ShotSizePromptHint(size string) string {
	switch NormalizeShotSize(size) {
	case ShotSizeWide:
		return "远景构图，环境全貌清晰，人物占画面较小，强调空间与氛围"
	case ShotSizeClose:
		return "特写构图，面部或关键细节占满画面，浅景深，背景虚化"
	default:
		return "中景构图，人物半身或全身，人物与环境平衡，叙事清晰"
	}
}

// ListShotSizeOptions 供 API / UI 渲染。
func ListShotSizeOptions() []ShotSizeOption {
	return []ShotSizeOption{
		{ID: ShotSizeWide, Label: "远景"},
		{ID: ShotSizeMedium, Label: "中景"},
		{ID: ShotSizeClose, Label: "特写"},
	}
}

// ShotSizeOption 景别枚举项。
type ShotSizeOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
