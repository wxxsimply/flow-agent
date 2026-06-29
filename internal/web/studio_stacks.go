package web

import (
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
)

const (
	// StudioStackSample 样片：总成本封顶 5 元。
	StudioStackSample = "micro-movie-cap5"
	// StudioStackFinal 成片：每 30 秒 5 元。
	StudioStackFinal = "micro-movie-seedance"
)

// DefaultStudioStack 桌面/Web Studio 默认视频方案（样片）。
const DefaultStudioStack = StudioStackSample

type stackSummaryJSON struct {
	Name           string   `json:"name"`
	Label          string   `json:"label"`
	Description    string   `json:"description"`
	CostHint       string   `json:"cost_hint"`
	CostMode       string   `json:"cost_mode"`
	TargetDuration int      `json:"target_duration_sec"`
	ImageProvider  string   `json:"image_provider"`
	VideoProvider  string   `json:"video_provider"`
	TTSProvider    string   `json:"tts_provider"`
	RequiredHints  []string `json:"required_hints"`
}

var studioTierDefs = []struct {
	Name     string
	Label    string
	CostHint string
	CostMode string
}{
	{Name: StudioStackSample, Label: "样片", CostHint: "封顶 5 元", CostMode: "cap"},
	{Name: StudioStackFinal, Label: "成片", CostHint: "每 30 秒 5 元", CostMode: "per_30_sec"},
}

// normalizeStudioStack 将 Studio prefs / 旧 stack 名归一化为样片或成片。
func normalizeStudioStack(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return DefaultStudioStack
	}
	switch s {
	case StudioStackSample, StudioStackFinal:
		return s
	case "micro-movie-economy", "micro-movie-wan-quick":
		return StudioStackSample
	default:
		if strings.HasPrefix(s, "micro-movie-") {
			return StudioStackFinal
		}
		return s
	}
}

func listStudioStacks(root string) ([]stackSummaryJSON, error) {
	var out []stackSummaryJSON
	for _, tier := range studioTierDefs {
		stack, err := loadStackByName(root, tier.Name)
		if err != nil {
			continue
		}
		desc := stack.Description
		if strings.TrimSpace(desc) == "" {
			desc = tier.Label + " · " + tier.CostHint
		}
		out = append(out, stackSummaryJSON{
			Name:           tier.Name,
			Label:          tier.Label,
			Description:    desc,
			CostHint:       tier.CostHint,
			CostMode:       tier.CostMode,
			TargetDuration: stack.TargetDurationSec,
			ImageProvider:  stack.ImageConfig().Provider,
			VideoProvider:  stack.VideoConfig().Provider,
			TTSProvider:    stack.TTSConfig().Provider,
			RequiredHints:  requiredProviderHints(stack),
		})
	}
	if len(out) == 0 {
		path := filepath.Join(root, "config", "stacks", DefaultStudioStack+".yaml")
		stack, err := config.LoadStack(path)
		if err == nil && stack != nil {
			out = append(out, stackSummaryJSON{
				Name:           stack.Name,
				Label:          "样片",
				Description:    stack.Description,
				CostHint:       "封顶 5 元",
				CostMode:       "cap",
				TargetDuration: stack.TargetDurationSec,
				ImageProvider:  stack.ImageConfig().Provider,
				VideoProvider:  stack.VideoConfig().Provider,
				TTSProvider:    stack.TTSConfig().Provider,
				RequiredHints:  requiredProviderHints(stack),
			})
		}
	}
	return out, nil
}
