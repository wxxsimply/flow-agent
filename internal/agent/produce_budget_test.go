package agent

import (
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestEstimateProduceCost_cap5UnderBudget(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stack, err := config.LoadStack(filepath.Join(root, "config", "stacks", "micro-movie-cap5.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rc := &runctx.Context{
		App: &config.App{Stack: stack},
		Manifest: &artifacts.Manifest{
			Cost: &artifacts.CostLedger{TotalCNY: 0.5},
		},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{
			{ID: "s01", DurationSec: 8, Narration: "雨夜，少年站在天台。"},
			{ID: "s02", DurationSec: 8, Narration: "手机屏幕亮起，他深吸一口气。"},
			{ID: "s03", DurationSec: 8, Narration: "霓虹在雨幕里渐渐模糊。"},
		},
	}
	est := EstimateProduceCost(rc, sb)
	if est.ShotCount != 3 {
		t.Fatalf("shots=%d", est.ShotCount)
	}
	if est.TotalCNY > 4.8 {
		t.Fatalf("produce estimate too high: %.2f", est.TotalCNY)
	}
	if err := CheckProduceBudget(rc, sb); err != nil {
		t.Fatalf("should pass budget: %v (est=%.2f spent=0.5)", err, est.TotalCNY)
	}
}

func TestCheckProduceBudget_blocksWhenOverCap(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stack, err := config.LoadStack(filepath.Join(root, "config", "stacks", "micro-movie-cap5.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rc := &runctx.Context{
		App: &config.App{Stack: stack},
		Manifest: &artifacts.Manifest{
			Cost: &artifacts.CostLedger{TotalCNY: 4},
		},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{
			{ID: "s01", DurationSec: 10, Narration: "测试"},
			{ID: "s02", DurationSec: 10, Narration: "测试"},
			{ID: "s03", DurationSec: 10, Narration: "测试"},
		},
	}
	if err := CheckProduceBudget(rc, sb); err == nil {
		t.Fatal("expected budget block when spent=4 + produce est")
	}
}

func TestEffectiveCostBudget_seedancePer30Sec(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Name:                  "micro-movie-seedance",
				CostBudgetPer30SecCNY: 5,
				TargetDurationSec:     30,
			},
		},
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 30},
	}
	got := EffectiveCostBudgetCNY(rc)
	if got < 4.9 || got > 5.1 {
		t.Fatalf("30s want ~5, got %v", got)
	}
	rc.Creative.TargetDurationSec = 60
	if got2 := EffectiveCostBudgetCNY(rc); got2 < 9.9 || got2 > 10.1 {
		t.Fatalf("60s want ~10, got %v", got2)
	}
}

func TestEstimateProduceCost_seedance30sUnderBudget(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stack, err := config.LoadStack(filepath.Join(root, "config", "stacks", "micro-movie-seedance.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rc := &runctx.Context{
		App:      &config.App{Stack: stack},
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 30},
		Manifest: &artifacts.Manifest{Cost: &artifacts.CostLedger{TotalCNY: 1.5}},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{
			{ID: "s01", DurationSec: 5, Narration: "雨夜，少年站在天台边缘。"},
			{ID: "s02", DurationSec: 5, Narration: "手机屏幕亮起，他深吸一口气。"},
			{ID: "s03", DurationSec: 5, Narration: "霓虹在雨幕里渐渐模糊。"},
			{ID: "s04", DurationSec: 5, Narration: "他转身走向楼梯口。"},
			{ID: "s05", DurationSec: 5, Narration: "脚步声在空廊里回响。"},
			{ID: "s06", DurationSec: 5, Narration: "远处车灯划破夜色。"},
		},
	}
	applyStoryboardProduceCaps(rc, sb)
	if err := CheckProduceBudget(rc, sb); err != nil {
		t.Fatalf("seedance 30s economy should pass: %v shots=%d", err, len(sb.Shots))
	}
}

func TestParseSignalStatsYAVG_altFormat(t *testing.T) {
	v, ok := parseSignalStatsYAVG("frame=1 YAVG=9.5")
	if !ok || v < 9.4 || v > 9.6 {
		t.Fatalf("alt YAVG: ok=%v v=%v", ok, v)
	}
}
