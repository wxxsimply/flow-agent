package agent

import (
	"log/slog"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// applyStoryboardProduceCaps 按 stack 截断镜数并重新对齐总时长。
func applyStoryboardProduceCaps(rc *runctx.Context, sb *artifacts.Storyboard) {
	if sb == nil || rc == nil || rc.App == nil || rc.App.Stack == nil {
		return
	}
	vid := rc.App.Stack.VideoConfig()
	maxShots := vid.MaxProduceShots
	if needed := produceShotCapForTarget(rc); needed > maxShots {
		maxShots = needed
	}
	if maxShots > 0 {
		sb.CapShots(maxShots)
	}
	pol := StoryboardPolicyForRun(rc)
	if pol.RelaxDurationTarget {
		sb.AlignDurationsFromNarration(artifacts.DefaultSecPerRune)
		slog.Info("storyboard durations aligned from narration",
			"estimated_sec", sb.TotalDurationSec(),
			"target_sec", rc.TargetDurationSec())
	} else {
		sb.NormalizeDurations(rc.TargetDurationSec())
	}
	CapStoryboardForBudget(rc, sb)
	CapShotsForProduceBudget(rc, sb)
}
