package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestPromptVariantForBoNWithPhysics(t *testing.T) {
	shot := &artifacts.Shot{
		PhysicsCues:      "鞋底贴地，重力向下",
		ForbiddenPhysics: "穿模，无支撑悬浮",
	}
	out := promptVariantForBoN("base", shot, 0)
	if !strings.Contains(out, "base") || !strings.Contains(out, "鞋底") {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, "固定机位") {
		t.Fatalf("variant 0 should fix camera: %q", out)
	}
	v2 := promptVariantForBoN("base", shot, 2)
	if !strings.Contains(v2, "禁止") || !strings.Contains(v2, "穿模") {
		t.Fatalf("variant 2 should include forbidden: %q", v2)
	}
}

func TestPromptVariantForBoN(t *testing.T) {
	base := "人物站立"
	v0 := promptVariantForBoN(base, nil, 0)
	if v0 == base {
		t.Fatal("variant should append suffix")
	}
	if !strings.Contains(v0, "固定机位") {
		t.Fatalf("got %q", v0)
	}
}

func TestFirstForbiddenItems(t *testing.T) {
	got := firstForbiddenItems("穿模，悬浮，瞬移", 2)
	if got != "穿模、悬浮" {
		t.Fatalf("got %q", got)
	}
}

func TestParseSignalStatsYAVG(t *testing.T) {
	out := `[Parsed_signalstats_0 @ 0x1] lavfi.signalstats.YAVG=12.345678`
	v, ok := parseSignalStatsYAVG(out)
	if !ok || v < 12.34 || v > 12.35 {
		t.Fatalf("parse YAVG: ok=%v v=%v", ok, v)
	}
}

func TestAbs(t *testing.T) {
	if abs(-3) != 3 || abs(2) != 2 {
		t.Fatal("abs")
	}
}

func TestBonVariantSuffix(t *testing.T) {
	s := bonVariantSuffix(1, "重力向下", "")
	if !strings.Contains(s, "单一主动作") {
		t.Fatalf("got %q", s)
	}
}
