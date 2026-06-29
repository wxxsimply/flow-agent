package artifacts

import "testing"

func TestBuildTimeline(t *testing.T) {
	sb := validTestStoryboard(180)
	tl := BuildTimeline(sb, 1)
	if len(tl.Shots) != len(sb.Shots) {
		t.Fatalf("shots=%d want %d", len(tl.Shots), len(sb.Shots))
	}
	if tl.Shots[0].ImagePath != "artifacts/media/shots/s01.png" {
		t.Fatalf("unexpected image path: %s", tl.Shots[0].ImagePath)
	}
}
