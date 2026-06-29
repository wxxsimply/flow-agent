package agent

import (
	"log/slog"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// DefaultCostPer30SecCNY Studio Seedance 默认：每 30 秒成片预算 5 元（含扩写 + produce）。
const DefaultCostPer30SecCNY = 5.0

// EffectiveCostBudgetCNY 本次运行的总成本硬顶（元）。优先 cost_budget_per_30_sec_cny × 时长/30。
func EffectiveCostBudgetCNY(rc *runctx.Context) float64 {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return 0
	}
	stack := rc.App.Stack
	if stack.CostBudgetPer30SecCNY > 0 {
		target := rc.TargetDurationSec()
		if target <= 0 {
			target = stack.TargetDurationSec
		}
		if target <= 0 {
			target = 30
		}
		return stack.CostBudgetPer30SecCNY * float64(target) / 30.0
	}
	return stack.CostBudgetCNY
}

// IsEconomyStack 低成本栈：启用旁白截断、clip 计费等策略。
func IsEconomyStack(rc *runctx.Context) bool {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return false
	}
	if IsBudgetCapStack(rc) {
		return true
	}
	return rc.App.Stack.CostBudgetPer30SecCNY > 0
}

// IsBudgetCapStack 固定 5 元封顶栈（如 micro-movie-cap5）。
func IsBudgetCapStack(rc *runctx.Context) bool {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return false
	}
	b := rc.App.Stack.CostBudgetCNY
	return b > 0 && b <= 5 && rc.App.Stack.CostBudgetPer30SecCNY <= 0
}

// StackTargetDurationSec 栈配置的目标成片时长（秒），未配置返回 0。
func StackTargetDurationSec(rc *runctx.Context) int {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return 0
	}
	return rc.App.Stack.TargetDurationSec
}

// ClipDurationForRun 当前栈单镜 i2v 时长上限（秒），用于视频计费估算。
func ClipDurationForRun(rc *runctx.Context) int {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return 0
	}
	d := rc.App.Stack.VideoConfig().ClipDurationSec
	if d > 0 {
		return d
	}
	return 4
}

// MaxNarrationRunesPerShot 按 target/镜数估算每镜旁白字数上限（约 4 字/秒）。
func MaxNarrationRunesPerShot(targetSec, numShots int) int {
	if numShots <= 0 {
		numShots = 1
	}
	if targetSec <= 0 {
		targetSec = numShots * 6
	}
	perShotSec := float64(targetSec) / float64(numShots)
	maxRunes := int(perShotSec / artifacts.DefaultSecPerRune)
	if maxRunes < 12 {
		maxRunes = 12
	}
	return maxRunes
}

// TrimNarrationForClip 将旁白截到字数上限；优先在句号/逗号边界截断，禁止句中硬切。
func TrimNarrationForClip(narration string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 32
	}
	r := []rune(strings.TrimSpace(narration))
	if len(r) <= maxRunes {
		return string(r)
	}
	chunk := r[:maxRunes]
	if idx := lastRuneBoundaryIndex(chunk, isSentenceEndRune); idx >= 0 {
		return string(chunk[:idx+1])
	}
	if idx := lastRuneBoundaryIndex(chunk, isCommaRune); idx >= 0 {
		return string(chunk[:idx+1])
	}
	// 无法在边界内完整表达时保留原文，避免「…的统」类残片
	return string(r)
}

func isSentenceEndRune(r rune) bool {
	switch r {
	case '。', '！', '？', '…', '.', '!', '?':
		return true
	default:
		return false
	}
}

func isCommaRune(r rune) bool {
	return r == '，' || r == ','
}

func lastRuneBoundaryIndex(runes []rune, ok func(rune) bool) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if ok(runes[i]) {
			return i
		}
	}
	return -1
}

// CapStoryboardForBudget 低成本栈：按句截短过长旁白，不覆盖 TTS 驱动时长。
func CapStoryboardForBudget(rc *runctx.Context, sb *artifacts.Storyboard) {
	if sb == nil || !IsEconomyStack(rc) {
		return
	}
	target := rc.TargetDurationSec()
	if target <= 0 {
		target = StackTargetDurationSec(rc)
	}
	if target <= 0 {
		target = len(sb.Shots) * ClipDurationForRun(rc)
	}
	maxRunes := MaxNarrationRunesPerShot(target, len(sb.Shots))
	for i := range sb.Shots {
		orig := sb.Shots[i].Narration
		narr := TrimNarrationForClip(orig, maxRunes)
		if narr != orig && !artifacts.NarrationComplete(narr) {
			slog.Warn("budget cap trimmed narration without sentence end; keeping original",
				"shot", sb.Shots[i].ID, "max_runes", maxRunes)
			narr = orig
		}
		sb.Shots[i].Narration = narr
		if strings.TrimSpace(sb.Shots[i].Subtitle) == "" || sb.Shots[i].Subtitle == orig {
			sb.Shots[i].Subtitle = narr
		}
	}
	sb.TargetDurationSec = float64(target)
	sb.SyncTotalNarrationSec()
}

// CapShotsForProduceBudget 在 produce 前按剩余预算裁减镜数，避免整单因镜数过多被拒。
func CapShotsForProduceBudget(rc *runctx.Context, sb *artifacts.Storyboard) {
	if sb == nil || rc == nil || rc.App == nil || rc.App.Stack == nil {
		return
	}
	budget := EffectiveCostBudgetCNY(rc)
	if budget <= 0 {
		return
	}
	spent := 0.0
	if rc.Manifest != nil && rc.Manifest.Cost != nil {
		spent = rc.Manifest.Cost.TotalCNY
	}
	remaining := budget - spent
	if remaining <= 0 {
		return
	}
	vid := rc.App.Stack.VideoConfig()
	minShots := 2
	if vid.MaxProduceShots > 0 && vid.MaxProduceShots < minShots {
		minShots = vid.MaxProduceShots
	}
	if minShots < 1 {
		minShots = 1
	}
	for len(sb.Shots) > minShots {
		est := EstimateProduceCost(rc, sb)
		if est.TotalCNY <= remaining {
			break
		}
		sb.CapShots(len(sb.Shots) - 1)
		if IsEconomyStack(rc) {
			CapStoryboardForBudget(rc, sb)
		}
		sb.NormalizeDurations(rc.TargetDurationSec())
	}
}

// BillableVideoSeconds 预算估算 / i2v 请求用的计费秒数（受 clip 上限约束）。
func BillableVideoSeconds(durationSec float64, vidCfg config.StackVideoConfig) float64 {
	if durationSec <= 0 {
		return 0
	}
	if vidCfg.EnforceClipDurationCap && vidCfg.ClipDurationSec > 0 {
		cap := float64(vidCfg.ClipDurationSec)
		if durationSec > cap {
			return cap
		}
	}
	return durationSec
}
