package agent

import (
	"math"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// produceShotCapForTarget 按目标成片时长估算至少需要的镜数（受 stack max_produce_shots 限制）。
func produceShotCapForTarget(rc *runctx.Context) int {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return 6
	}
	vid := rc.App.Stack.VideoConfig()
	clip := vid.ClipDurationSec
	if clip <= 0 {
		clip = 5
	}
	target := rc.TargetDurationSec()
	if target <= 0 {
		target = rc.App.Stack.TargetDurationSec
	}
	if target <= 0 {
		target = 30
	}
	need := int(math.Ceil(float64(target) / float64(clip)))
	max := vid.MaxProduceShots
	if max <= 0 {
		max = 8
	}
	if need > max {
		need = max
	}
	if need < 3 {
		need = 3
	}
	return need
}

// reconcileTimelineToTarget 将时间轴总时长对齐到用户目标（TTS 偏短则拉伸，偏长则压缩；不超过 target）。
func reconcileTimelineToTarget(rc *runctx.Context, tl *artifacts.Timeline, audioTotalSec float64) {
	if rc == nil || tl == nil || len(tl.Shots) == 0 {
		return
	}
	target := float64(rc.TargetDurationSec())
	if target <= 0 {
		return
	}
	videoTotal := tl.TotalVideoSec()
	if videoTotal <= 0 {
		return
	}
	desired := target
	if videoTotal >= desired*0.95 && videoTotal <= desired*1.08 {
		return
	}
	scale := desired / videoTotal
	if scale < 0.2 || scale > 5.0 {
		return
	}
	for i := range tl.Shots {
		tl.Shots[i].DurationSec = math.Round(tl.Shots[i].DurationSec*scale*100) / 100
	}
	if tl.Audio != nil {
		tl.Audio.TotalSec = math.Min(audioTotalSec, tl.TotalVideoSec())
	}
}
