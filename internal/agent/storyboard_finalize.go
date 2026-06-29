package agent

import (
	"context"
	"log/slog"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent/skills"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func stackVideoAndAssemble(rc *runctx.Context) (config.StackVideoConfig, config.StackAssembleConfig) {
	asm := config.StackAssembleConfig{LLMReview: true}
	vid := config.StackVideoConfig{KeyframeMode: "multi", HeroShotCount: 3}
	if rc.App != nil && rc.App.Stack != nil {
		return rc.App.Stack.VideoConfig(), rc.App.Stack.AssembleConfig()
	}
	return vid, asm
}

// applyStoryboardPostProcess tier 标注、规则审查与可选 LLM 审查。
func applyStoryboardPostProcess(rc *runctx.Context, sb *artifacts.Storyboard) (*artifacts.StoryboardReviewReport, error) {
	vidCfg, asmCfg := stackVideoAndAssemble(rc)
	if strings.EqualFold(strings.TrimSpace(vidCfg.KeyframeMode), "tiered") {
		artifacts.AssignShotTiers(sb, vidCfg.HeroShotCount)
		slog.Info("shot tiers assigned", "hero_count", vidCfg.HeroShotCount, "mode", "tiered")
	}

	report := sb.ReviewStoryboard()
	if asmCfg.LLMReview {
		if summary, err := skills.ApplyLLMStoryboardReview(context.Background(), rc, sb); err != nil {
			slog.Warn("skill llm storyboard review failed", "err", err)
		} else if summary != "" {
			slog.Info("skill llm storyboard review", "summary", summary)
			report2 := sb.ReviewStoryboard()
			report.Fixed += report2.Fixed
			report.Issues = append(report.Issues, report2.Issues...)
		}
	}
	return &report, nil
}
