package agent

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/cost"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ProduceCostEstimate produce 阶段分项预估（人民币）。
type ProduceCostEstimate struct {
	ShotCount      int
	VideoAPICalls  int
	VideoSeconds   float64
	TTSCharacters  int
	ImageCount     int
	VideoCNY       float64
	TTSCNY         float64
	ImageCNY       float64
	TotalCNY       float64
}

// EstimateProduceCost 估算 produce 媒体成本（TTS + 出图 + i2v）。
func EstimateProduceCost(rc *runctx.Context, sb *artifacts.Storyboard) ProduceCostEstimate {
	out := ProduceCostEstimate{}
	if rc == nil || rc.App == nil || rc.App.Stack == nil || sb == nil {
		return out
	}
	stack := rc.App.Stack
	vidCfg := config.ApplyMediaSpecVideo(stack.VideoConfig(), config.MediaSpecFromCreative(rc.Creative))
	shots := sb.Shots
	if max := vidCfg.MaxProduceShots; max > 0 && len(shots) > max {
		shots = shots[:max]
	}
	out.ShotCount = len(shots)
	if out.ShotCount == 0 {
		return out
	}

	clipDefault := vidCfg.ClipDurationSec
	if clipDefault <= 0 {
		clipDefault = 8
	}
	rates := cost.RatesFromStack(stack)

	for _, shot := range shots {
		out.ImageCount++
		chars := utf8.RuneCountInString(shot.Narration)
		if chars == 0 {
			chars = utf8.RuneCountInString(shot.Subtitle)
		}
		out.TTSCharacters += chars

		pol := ShotProducePolicyFor(vidCfg, shot)
		segments := 1
		if pol.MultiKeyframe {
			beats := actionBeatsForShot(shot)
			if len(beats) >= 2 {
				segments = len(beats) - 1
			}
		}
		candidates := 1
		if pol.BoNEnabled && pol.BoNCandidates >= 2 {
			candidates = pol.BoNCandidates
		}
		out.VideoAPICalls += segments * candidates

		dur := shot.DurationSec
		if dur <= 0 {
			dur = float64(clipDefault)
		}
		dur = BillableVideoSeconds(dur, vidCfg)
		out.VideoSeconds += dur * float64(segments)
	}

	out.ImageCNY = rates.ImageCNY(out.ImageCount)
	out.TTSCNY = rates.TTSCNY(out.TTSCharacters)
	out.VideoCNY = rates.VideoCNY(out.VideoSeconds)
	// 若 BoN 产生额外 API 调用，按秒数比例放大 video 估算
	if out.VideoAPICalls > out.ShotCount && out.ShotCount > 0 {
		scale := float64(out.VideoAPICalls) / float64(out.ShotCount)
		out.VideoCNY *= scale
	}
	out.TotalCNY = out.ImageCNY + out.TTSCNY + out.VideoCNY
	return out
}

// CheckProduceBudget 在 produce 开始前校验：已花费 + produce 预估 ≤ 有效预算。
func CheckProduceBudget(rc *runctx.Context, sb *artifacts.Storyboard) error {
	if rc == nil || rc.App == nil || rc.App.Stack == nil || sb == nil {
		return nil
	}
	budget := EffectiveCostBudgetCNY(rc)
	if budget <= 0 {
		return nil
	}

	CapShotsForProduceBudget(rc, sb)

	spent := 0.0
	if rc.Manifest != nil && rc.Manifest.Cost != nil {
		spent = rc.Manifest.Cost.TotalCNY
	}
	est := EstimateProduceCost(rc, sb)
	projected := spent + est.TotalCNY

	if projected <= budget {
		return nil
	}
	return fmt.Errorf(
		"预算超限：已花费 %.2f 元 + produce 预估 %.2f 元 = %.2f 元，超过预算 %.2f 元（%d 镜 / %d 次 i2v，约 %.0f 秒成片）；请缩短目标时长或精简分镜旁白",
		math.Round(spent*100)/100,
		math.Round(est.TotalCNY*100)/100,
		math.Round(projected*100)/100,
		math.Round(budget*100)/100,
		est.ShotCount,
		est.VideoAPICalls,
		float64(rc.TargetDurationSec()),
	)
}
