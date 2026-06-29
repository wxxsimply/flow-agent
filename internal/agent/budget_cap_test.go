package agent

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestTrimNarrationForClip_sentenceBoundary(t *testing.T) {
	long := "黑暗再次降临，骑士去驱逐邪恶的统领。月光下，他勒马而立，目光如炬。"
	got := TrimNarrationForClip(long, 16)
	if !strings.HasSuffix(got, "。") && !strings.HasSuffix(got, "，") {
		t.Fatalf("expected boundary trim, got %q", got)
	}
	if got == "黑暗再次降临，骑士去驱逐邪恶的统" {
		t.Fatalf("must not hard-cut mid-word: %q", got)
	}
}

func TestTrimNarrationForClip_keepsShort(t *testing.T) {
	n := "雨夜，天台边缘，少年紧握手机，霓虹在雨幕里模糊。"
	if got := TrimNarrationForClip(n, 32); got != n {
		t.Fatalf("short narration should be unchanged: %q", got)
	}
}

func TestCapStoryboardForBudget_capsLongNarration(t *testing.T) {
	stack := &config.Stack{
		Name:              "micro-movie-cap5",
		TargetDurationSec: 18,
		CostBudgetCNY:     5,
		Video: map[string]any{
			"clip_duration_sec": 4,
			"keyframe_mode":     "single",
			"enabled":           true,
			"all_shots":         true,
		},
	}
	rc := &runctx.Context{
		App: &config.App{Stack: stack},
		Manifest: &artifacts.Manifest{
			Cost: &artifacts.CostLedger{TotalCNY: 0.04},
		},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{
			{ID: "s01", DurationSec: 14.5, Narration: "深夜的街道空无一人，路灯昏黄，一个身影从巷口走出，脚步沉稳。他停下，仰头望向那扇漆黑的窗，深吸一口气，然后继续前行。"},
			{ID: "s02", DurationSec: 16.8, Narration: "他走下楼梯，进入地下室。昏黄的灯光下，他走向木桌，拿起一个泛黄的信封，抽出里面的照片。"},
			{ID: "s03", DurationSec: 18.3, Narration: "他进入办公室，从抽屉里找到一把匕首。转身时，门口出现一个人影。两人对峙，灯光忽明忽暗。"},
		},
	}
	applyStoryboardProduceCaps(rc, sb)
	for _, shot := range sb.Shots {
		if !artifacts.NarrationComplete(shot.Narration) {
			t.Fatalf("%s narration incomplete: %q", shot.ID, shot.Narration)
		}
	}
	est := EstimateProduceCost(rc, sb)
	if est.TotalCNY > 4.5 {
		t.Fatalf("produce est too high after cap: %.2f (video=%.2f)", est.TotalCNY, est.VideoCNY)
	}
	if err := CheckProduceBudget(rc, sb); err != nil {
		t.Fatalf("budget should pass: %v", err)
	}
}

func TestMaxNarrationRunesPerShot_cap5(t *testing.T) {
	if got := MaxNarrationRunesPerShot(18, 3); got != 24 {
		t.Fatalf("want 24 got %d", got)
	}
}

func TestTargetDurationSec_clampedForCap5(t *testing.T) {
	rc := &runctx.Context{
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 120},
		App: &config.App{
			Stack: &config.Stack{
				Name:              "micro-movie-cap5",
				TargetDurationSec: 18,
				CostBudgetCNY:     5,
			},
		},
	}
	if rc.TargetDurationSec() != 18 {
		t.Fatalf("want 18 got %d", rc.TargetDurationSec())
	}
}

func TestTrimNarrationForClip_noMidCharWhenNoBoundary(t *testing.T) {
	// 无标点长串：保留全文
	n := strings.Repeat("字", 40)
	got := TrimNarrationForClip(n, 16)
	if utf8.RuneCountInString(got) != 40 {
		t.Fatalf("expected full text preserved, got len=%d", utf8.RuneCountInString(got))
	}
}
