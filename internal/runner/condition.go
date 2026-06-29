package runner

import (
	"fmt"
	"math"
	"strings"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// EvaluateGateCondition 解析 YAML gate.condition 并校验（J4）。
func EvaluateGateCondition(rc *runctx.Context, g workflow.GateDefinition) error {
	cond := strings.TrimSpace(g.Condition)
	if cond == "" {
		return checkAutomaticGate(rc, g.ID)
	}
	if rc.DryRun && strings.Contains(cond, "chapter") {
		return nil
	}

	switch {
	case strings.Contains(cond, "continuity-report") && strings.Contains(cond, "critical_count"):
		return checkAutomaticGate(rc, "continuity_passed")
	case strings.Contains(cond, "compliance-report") && strings.Contains(cond, "blocked"):
		return checkAutomaticGate(rc, "no_block_issues")
	case strings.Contains(cond, "storyboard.total_duration_sec"):
		return checkAutomaticGate(rc, "duration_ok")
	case strings.Contains(cond, "chapter.md.char_count"):
		return checkAutomaticGate(rc, "length_in_range")
	case strings.Contains(cond, "sync-report") && strings.Contains(cond, "max_drift_sec"):
		return checkAutomaticGate(rc, "av_sync_ok")
	case strings.Contains(cond, "produce-degradation"):
		return checkAutomaticGate(rc, "visual_quality_ok")
	default:
		return fmt.Errorf("gate %q: unsupported condition %q", g.ID, cond)
	}
}

// evalDurationCondition 供测试直接调用 duration 表达式逻辑。
func evalDurationCondition(total, target, tolerance float64) bool {
	return math.Abs(total-target) <= tolerance
}
