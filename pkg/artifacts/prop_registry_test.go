package artifacts

import (
	"strings"
	"testing"
)

func TestExtractPropAppearance_ChineseNoPanic(t *testing.T) {
	// 长中文 prompt：字节下标与 rune 下标差异大，旧实现会 slice bounds panic
	prefix := strings.Repeat("雨夜霓虹巷，刺客潜行，", 20)
	visual := prefix + "右手紧握匕首，寒光一闪"
	got := extractPropAppearance("匕首", visual)
	if got == "" {
		t.Fatal("expected excerpt")
	}
	if !strings.Contains(got, "匕首") {
		t.Fatalf("excerpt=%q", got)
	}
}

func TestCollectPropsFromStoryboard_HeldAndHero(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{
				ID:              "s01",
				SceneBackground: "王殿",
				VisualPrompt:    "国王坐在王座上，右手紧握长剑",
				HeldProps:       FlexString("右手：长剑；左手：空"),
			},
			{
				ID:              "s02",
				SceneBackground: "王殿",
				VisualPrompt:    "国王仍坐王座，长剑横于膝上",
				HeldProps:       FlexString("右手：长剑；左手：空"),
			},
			{
				ID:           "s03",
				VisualPrompt: "雨巷，右手持匕首",
				HeldProps:    FlexString("右手：匕首；左手：空"),
			},
		},
	}
	sheets := CollectPropsFromStoryboard(sb)
	if sheets == nil {
		t.Fatal("expected prop sheets")
	}
	if len(sheets.Props) < 2 {
		t.Fatalf("expected at least held+hero props, got %d", len(sheets.Props))
	}
	var hasSword, hasDagger, hasThrone bool
	for _, pe := range sheets.Props {
		switch pe.Name {
		case "长剑":
			hasSword = true
			if pe.Category != PropCategoryHeld {
				t.Fatalf("长剑 category want held, got %s", pe.Category)
			}
		case "匕首":
			hasDagger = true
		case "王座":
			hasThrone = true
			if pe.Category != PropCategoryHeroScene {
				t.Fatalf("王座 category want hero_scene, got %s", pe.Category)
			}
		}
	}
	if !hasSword || !hasDagger {
		t.Fatalf("missing held props: sword=%v dagger=%v", hasSword, hasDagger)
	}
	if !hasThrone {
		t.Fatal("expected hero scene prop 王座")
	}
}

func TestApplyPropRefs(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", VisualPrompt: "右手持匕首", HeldProps: FlexString("右手：匕首；左手：空")},
		},
	}
	sheets := &PropSheets{
		Props: []PropSheetEntry{{
			ID: "p01-dagger", Name: "匕首", Category: PropCategoryHeld, Appearance: "银质匕首",
		}},
	}
	n := ApplyPropRefs(sb, sheets)
	if n != 1 {
		t.Fatalf("ApplyPropRefs fixed=%d want 1", n)
	}
	if len(sb.Shots[0].PropRefs) != 1 || sb.Shots[0].PropRefs[0] != "p01-dagger" {
		t.Fatalf("prop_refs=%v", sb.Shots[0].PropRefs)
	}
}

func TestReviewPropContinuity_HandSwitch(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{
				ID:              "s01",
				SceneBackground: "室内",
				HeldProps:       FlexString("右手：匕首；左手：空"),
			},
			{
				ID:              "s02",
				SceneBackground: "室内",
				HeldProps:       FlexString("右手：长剑；左手：空"),
			},
		},
	}
	sheets := &PropSheets{
		Props: []PropSheetEntry{
			{ID: "p01-dagger", Name: "匕首", Category: PropCategoryHeld},
			{ID: "p02-sword", Name: "长剑", Category: PropCategoryHeld},
		},
	}
	ApplyPropRefs(sb, sheets)
	issues := ReviewPropContinuity(sb, sheets)
	if len(issues) == 0 {
		t.Fatal("expected continuity error for hand prop switch")
	}
	if issues[0].Severity != "error" {
		t.Fatalf("severity=%s", issues[0].Severity)
	}
	if !strings.Contains(issues[0].Message, "匕首") || !strings.Contains(issues[0].Message, "长剑") {
		t.Fatalf("message=%q", issues[0].Message)
	}
}
