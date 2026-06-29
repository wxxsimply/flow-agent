package cost

import "github.com/flow-agent/flow-agent/pkg/artifacts"

// TargetCheck 单项成本与 stack 目标区间对比结果。
type TargetCheck struct {
	Category string  // llm | tts | image | video | other | total
	Actual   float64 // 实际 CNY
	Min      float64 // 目标下限（无配置时为 0）
	Max      float64 // 目标上限（无配置时为 0）
	InRange  bool    // 在 [Min, Max] 内；无目标时恒为 true
	HasTarget bool
}

// CompareTargets 将 ledger 与 cost_targets_cny 对照（YAML 键名与分项字段对应）。
func CompareTargets(ledger *artifacts.CostLedger, targets map[string][]float64) []TargetCheck {
	if ledger == nil {
		return nil
	}
	actuals := map[string]float64{
		"llm":   ledger.LLMCNY,
		"tts":   ledger.TTSCNY,
		"image": ledger.ImageCNY,
		"video": ledger.VideoCNY,
		"other": ledger.OtherCNY,
		"total": ledger.TotalCNY,
	}
	order := []string{"llm", "tts", "image", "video", "other", "total"}
	out := make([]TargetCheck, 0, len(order))
	for _, cat := range order {
		actual := actuals[cat]
		rng, ok := targets[cat]
		tc := TargetCheck{Category: cat, Actual: actual}
		if !ok || len(rng) < 2 {
			tc.InRange = true
			out = append(out, tc)
			continue
		}
		tc.HasTarget = true
		tc.Min = rng[0]
		tc.Max = rng[1]
		tc.InRange = actual >= tc.Min && actual <= tc.Max
		out = append(out, tc)
	}
	return out
}

// AnyOutOfRange 是否存在超预算或未达下限的项。
func AnyOutOfRange(checks []TargetCheck) bool {
	for _, c := range checks {
		if c.HasTarget && !c.InRange {
			return true
		}
	}
	return false
}
