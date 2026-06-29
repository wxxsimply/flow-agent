package artifacts

import "testing"

func TestAssignShotTiers(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01"}, {ID: "s02"}, {ID: "s03"}, {ID: "s04"}, {ID: "s05"},
		},
	}
	AssignShotTiers(sb, 3)
	heroes := 0
	for _, s := range sb.Shots {
		if IsHeroShot(s) {
			heroes++
		}
	}
	if heroes < 3 {
		t.Fatalf("expected at least 3 hero shots, got %d", heroes)
	}
	if !IsHeroShot(sb.Shots[0]) {
		t.Fatal("first shot should be hero")
	}
}
