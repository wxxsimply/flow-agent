package artifacts

import (
	"strings"
	"testing"
)

func TestInferHeldProps_oneSword(t *testing.T) {
	got := InferHeldProps("他拔出一把长剑", "少年持剑立于雨夜")
	if got == "" {
		t.Fatal("expected held props")
	}
	if !strings.Contains(got, "右手") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestParsePropHands_rightDagger(t *testing.T) {
	h := ParsePropHands("右手单持匕首")
	if h.Right == "" {
		t.Fatalf("expected right hand prop, got %+v", h)
	}
	if !strings.Contains(h.Right, "匕首") {
		t.Fatalf("unexpected right: %q", h.Right)
	}
}

func TestFormatHeldProps(t *testing.T) {
	got := FormatHeldProps(PropHands{Right: "匕首"})
	if !strings.Contains(got, "右手：匕首") || !strings.Contains(got, "左手：空") {
		t.Fatalf("got %q", got)
	}
}

func TestPropHandConflict(t *testing.T) {
	if !PropHandConflict("右手持剑；右手持光球") {
		t.Fatal("expected conflict for same hand twice")
	}
	if PropHandConflict("右手：匕首；左手：空") {
		t.Fatal("unexpected conflict")
	}
}

func TestPropsHandConsistencyNeg(t *testing.T) {
	neg := PropsHandConsistencyNeg("右手：匕首；左手：空")
	for _, want := range []string{"道具换手", "prop vanish", "武器变形"} {
		if !strings.Contains(neg, want) {
			t.Fatalf("missing %q in %q", want, neg)
		}
	}
}

func TestPropLockBlock(t *testing.T) {
	block := PropLockBlock("右手：匕首；左手：空")
	if !strings.Contains(block, "[PROP_LOCK]") || !strings.Contains(block, "禁止") {
		t.Fatalf("got %q", block)
	}
}

func TestActionBeatHandSwapRisk(t *testing.T) {
	ok, _ := ActionBeatHandSwapRisk([]string{"预备", "右手将匕首由垂直转为横握", "收势"})
	if !ok {
		t.Fatal("expected swap risk")
	}
	ok, _ = ActionBeatHandSwapRisk([]string{"预备", "放下匕首", "左手接过", "收势"})
	if ok {
		t.Fatal("expected no risk when release explicit")
	}
}

func TestPropsNarrationVisualConflict(t *testing.T) {
	ok, msg := PropsNarrationVisualConflict("他拔出一把剑", "双手各持长剑对峙")
	if !ok || msg == "" {
		t.Fatalf("expected conflict, got ok=%v msg=%q", ok, msg)
	}
}

func TestSanitizeHeldProps_replacementChar(t *testing.T) {
	got := SanitizeHeldPropsText("右手：空\ufffd；左手：空 2D动画风格")
	if got == "" {
		t.Fatal("expected sanitized held props")
	}
	if strings.Contains(got, "\ufffd") || strings.Contains(got, "2D动画") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestApplyPropLocks_inheritPrevShot(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{
				ID:              "s01",
				SceneBackground: "雨夜街道",
				VisualPrompt:    "右手紧握匕首",
				HeldProps:       FlexString("右手：匕首；左手：空"),
			},
			{
				ID:              "s02",
				SceneBackground: "雨夜街道",
				VisualPrompt:    "少年继续行走",
			},
		},
	}
	n := sb.ApplyPropLocks()
	if n <= 0 {
		t.Fatal("expected fixes")
	}
	if sb.Shots[1].HeldProps.String() == "" {
		t.Fatal("expected s02 to inherit held_props")
	}
}

func TestApplyPropLocks(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{{
			ID:           "s01",
			Narration:    "他拔出一把剑",
			VisualPrompt: "双手各持双剑",
		}},
	}
	n := sb.ApplyPropLocks()
	if n <= 0 {
		t.Fatal("expected fixes")
	}
	if sb.Shots[0].HeldProps == "" {
		t.Fatal("expected held_props")
	}
}

func TestNormalizeHeldProps(t *testing.T) {
	shot := &Shot{VisualPrompt: "右手横握匕首，雨夜"}
	if !NormalizeHeldProps(shot) {
		t.Fatal("expected change")
	}
	if !HeldPropsHasHandSide(shot.HeldProps.String()) {
		t.Fatalf("got %q", shot.HeldProps.String())
	}
}
