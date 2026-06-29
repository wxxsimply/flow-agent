package agent

import (
	"testing"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

func TestExtractBriefBody(t *testing.T) {
	raw := "# 第一镜输入\n\n开场\n\n# 故事背景\n\n背景\n\n---\n\n正文扩写内容"
	got := ExtractBriefBody(raw)
	if got != "正文扩写内容" {
		t.Fatalf("got %q", got)
	}
}

func TestUseQuickBrief_reviewWorkflowForcesExpand(t *testing.T) {
	rc := &runctx.Context{
		StopAfterStage: "assemble",
		App: &config.App{
			Stack: &config.Stack{
				Assemble: map[string]any{"quick_assemble": true},
			},
		},
	}
	if useQuickBrief(rc) {
		t.Fatal("stop_after assemble must force full brief expand")
	}
}

func TestUseQuickBrief_quickStack(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Assemble: map[string]any{"quick_assemble": true},
			},
		},
	}
	if !useQuickBrief(rc) {
		t.Fatal("expected quick brief when quick_assemble and no review stop")
	}
}

func TestBriefExpandConfigFor_cap5Defaults(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Assemble: map[string]any{
					"brief_runes_min":     2000,
					"brief_runes_max":     2400,
					"brief_runes_floor":   1800,
					"brief_segment_count": 2,
				},
			},
		},
	}
	cfg := briefExpandConfigFor(rc)
	if cfg.RunesMin != 2000 || cfg.RunesMax != 2400 || cfg.RunesFloor != 1800 || cfg.SegmentCount != 2 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestDryRunShotLanguageExpand_fullBriefWhenNotQuick(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Assemble: map[string]any{
					"quick_assemble":    false,
					"brief_runes_min":   2000,
					"brief_runes_floor": 1800,
				},
			},
		},
	}
	exp := dryRunShotLanguageExpand(rc, "夜黑风高，我决定去复仇", 3, 3)
	n := utf8.RuneCountInString(exp.ShotLanguageBrief)
	if n < 1800 {
		t.Fatalf("dry-run brief too short: %d runes", n)
	}
}
