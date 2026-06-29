package artifacts

import "testing"

func TestChapterCharBounds(t *testing.T) {
	min, max := ChapterCharBounds(180, nil)
	if min != 600 || max != 1800 {
		t.Fatalf("180s: got min=%d max=%d", min, max)
	}
	min, max = ChapterCharBounds(90, nil)
	if min != 300 || max != 900 {
		t.Fatalf("90s: got min=%d max=%d", min, max)
	}
}

func TestChapterCharBounds90sAccepts549(t *testing.T) {
	min, max := ChapterCharBounds(90, nil)
	n := 549
	if n < min || n > max {
		t.Fatalf("549 should be in [%d,%d]", min, max)
	}
}

func TestCountChapterBodyRunes(t *testing.T) {
	md := "## Scene 1\n\n你好世界。\n\n## Scene 2\n\n测试。\n"
	n := CountChapterBodyRunes(md)
	if n != 8 {
		t.Fatalf("got %d want 8", n)
	}
}
