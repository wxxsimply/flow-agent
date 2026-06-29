package artifacts

import "testing"

func TestRepairShots_FillsEmptyNarration(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", DurationSec: 5, VisualType: "ai_video", AIVideoBudget: true, VisualPrompt: "场景1", Narration: "第一句旁白。"},
			{ID: "s02", DurationSec: 5, VisualType: "ai_video", AIVideoBudget: true, VisualPrompt: "场景2", Subtitle: "字幕二"},
			{ID: "s03", DurationSec: 5, VisualType: "ai_video", AIVideoBudget: true, VisualPrompt: "场景3"},
		},
	}
	n := sb.RepairShots("", nil)
	if n == 0 {
		t.Fatal("expected repairs")
	}
	if sb.Shots[1].Narration != "字幕二" {
		t.Fatalf("s02 narration=%q", sb.Shots[1].Narration)
	}
	if sb.Shots[2].Narration == "" {
		t.Fatal("s03 narration still empty")
	}
}

func TestRepairShots_FromScriptLines(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", DurationSec: 5, VisualType: "ai_video", AIVideoBudget: true, VisualPrompt: "a"},
			{ID: "s02", DurationSec: 5, VisualType: "ai_video", AIVideoBudget: true, VisualPrompt: "b"},
		},
	}
	lines := []string{"第一场旁白", "第二场旁白"}
	sb.RepairShots("", lines)
	if sb.Shots[0].Narration != "第一场旁白" || sb.Shots[1].Narration != "第二场旁白" {
		t.Fatalf("got %q / %q", sb.Shots[0].Narration, sb.Shots[1].Narration)
	}
}
