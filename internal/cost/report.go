package cost

import (
	"fmt"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// FormatReport 生成人类可读的成本报告（含用量与预算对照）。
func FormatReport(runID string, ledger *artifacts.CostLedger, checks []TargetCheck) string {
	var b strings.Builder
	fmt.Fprintf(&b, "成本账本 run_id=%s\n", runID)
	if ledger == nil {
		b.WriteString("  (无数据)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "  LLM    CNY %.2f  (prompt_tokens=%d completion_tokens=%d)\n",
		ledger.LLMCNY, ledger.LLMPromptTokens, ledger.LLMCompletionTokens)
	fmt.Fprintf(&b, "  TTS    CNY %.2f  (characters=%d)\n", ledger.TTSCNY, ledger.TTSCharacters)
	fmt.Fprintf(&b, "  出图   CNY %.2f  (images=%d)\n", ledger.ImageCNY, ledger.ImageCount)
	fmt.Fprintf(&b, "  视频   CNY %.2f  (seconds=%.1f)\n", ledger.VideoCNY, ledger.VideoSeconds)
	fmt.Fprintf(&b, "  其他   CNY %.2f\n", ledger.OtherCNY)
	fmt.Fprintf(&b, "  合计   CNY %.2f\n", ledger.TotalCNY)

	if len(checks) > 0 {
		b.WriteString("\n预算对照 (standard-tier cost_targets_cny):\n")
		for _, c := range checks {
			if !c.HasTarget {
				continue
			}
			status := budgetStatus(c)
			fmt.Fprintf(&b, "  %-6s CNY %6.2f  目标 CNY %.0f-%.0f  %s\n",
				c.Category, c.Actual, c.Min, c.Max, status)
		}
	}
	return b.String()
}

func budgetStatus(c TargetCheck) string {
	if c.InRange {
		return "OK"
	}
	if c.Actual < c.Min {
		return "低于目标"
	}
	return "高于目标"
}
