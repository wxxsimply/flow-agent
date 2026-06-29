package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestTruncateMotionPrompt(t *testing.T) {
	long := strings.Repeat("动", 900)
	got := truncateMotionPrompt(long, 800)
	if len([]rune(got)) > 801 {
		t.Fatalf("expected truncation, got %d runes", len([]rune(got)))
	}
}

func TestShotDirectImagePrompt_userTextOnly(t *testing.T) {
	rc := &runctx.Context{
		Workflow: "micro-movie",
		Creative: &artifacts.CreativeOptions{
			InputMode:      "director",
			AnimationStyle: "2d",
			VisualTheme:    "arknights",
		},
	}
	shot := artifacts.Shot{
		VisualPrompt: "少年站在雨夜天台",
		Narration:    "不应混入旁白以外的扩写",
		ShotSize:     artifacts.ShotSizeClose,
		UserSource:   true,
	}
	p := shotDirectImagePrompt(rc, shot)
	if !strings.Contains(p, "少年站在雨夜天台") {
		t.Fatal("missing user visual")
	}
	if !strings.Contains(p, "特写") {
		t.Fatal("missing shot size hint")
	}
	if strings.Contains(p, "不应混入旁白以外的扩写") {
		t.Fatal("should not add narration when visual set")
	}
}
