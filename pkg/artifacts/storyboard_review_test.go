package artifacts

import "testing"

func TestReviewStoryboardFillsPhysics(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{{
			ID: "s01", DurationSec: 5, VisualPrompt: "雨夜街道",
			Narration: "他停下脚步",
		}},
	}
	report := sb.ReviewStoryboard()
	if report.Fixed < 2 {
		t.Fatalf("expected auto-fixes, got fixed=%d", report.Fixed)
	}
	if sb.Shots[0].PhysicsCues == "" {
		t.Fatal("physics_cues should be filled")
	}
	if sb.Shots[0].ForbiddenPhysics == "" {
		t.Fatal("forbidden_physics should be filled")
	}
	if len(sb.Shots[0].ActionBeats) < 3 {
		t.Fatal("action_beats should be filled")
	}
}

func TestReviewStoryboardSinglePrimaryAction(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{{
			ID: "s01", DurationSec: 5, VisualPrompt: "走廊",
			Narration: "他向前走",
			ActionBeats: []string{
				"迈步向前",
				"转身开门",
				"收势站立",
			},
			PhysicsCues:      "重力向下，足底贴地",
			ForbiddenPhysics: "穿模，无支撑悬浮，违反重力",
		}},
	}
	report := sb.ReviewStoryboard()
	if report.Fixed == 0 {
		t.Fatal("expected fix for multiple active beats")
	}
	if countActiveMotionBeats(sb.Shots[0].ActionBeats) > 1 {
		t.Fatalf("expected single active beat, got %v", sb.Shots[0].ActionBeats)
	}
}

func TestPhysicsForbiddenPaired(t *testing.T) {
	if !physicsForbiddenPaired("重力向下，足底贴地", "穿模，无支撑悬浮，违反重力") {
		t.Fatal("should detect paired cues/forbidden")
	}
	if physicsForbiddenPaired("雨夜街道", "穿模") {
		t.Fatal("short unrelated pair should fail")
	}
}

func TestEnforceSinglePrimaryAction(t *testing.T) {
	shot := Shot{
		VisualPrompt: "角色扶栏",
		ActionBeats:  []string{"预备", "抬手扶栏", "收势"},
	}
	if !enforceSinglePrimaryAction(&shot) {
		t.Fatal("expected change")
	}
	if len(shot.ActionBeats) != 3 {
		t.Fatalf("beats=%v", shot.ActionBeats)
	}
}
