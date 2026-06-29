package artifacts

import (
	"strings"
	"testing"
)

func TestBuildContinuityLedger(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{{
			ID:           "s01",
			ShotSize:     "wide",
			VisualPrompt: "三十岁左右男子，深灰夹克，雨夜霓虹巷口驻足",
			PhysicsCues:  "重力向下",
			Narration:    "他停下了脚步",
		}},
	}
	md := BuildContinuityLedger(sb, "故事发生在雨夜。", nil)
	for _, want := range []string{"锁定词", "三十岁左右", "s01", "重力向下"} {
		if !strings.Contains(md, want) {
			t.Fatalf("ledger missing %q: %s", want, md)
		}
	}
}

func TestBuildLightCharacterSheets(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{VisualPrompt: "短发女子，红色风衣", HeldProps: FlexString("右手：匕首；左手：空")},
			{VisualPrompt: "中景", HeldProps: FlexString("右手：伞；左手：空")},
		},
	}
	s := BuildLightCharacterSheets(sb)
	if s == nil || len(s.Characters) != 1 || s.Characters[0].Appearance == "" {
		t.Fatal("expected light character sheet")
	}
	if strings.Contains(s.Characters[0].Appearance, "道具锁定") {
		t.Fatalf("appearance must not merge held_props: %q", s.Characters[0].Appearance)
	}
	if strings.Contains(s.Characters[0].Appearance, "伞") {
		t.Fatal("appearance must not include props from other shots")
	}
}
