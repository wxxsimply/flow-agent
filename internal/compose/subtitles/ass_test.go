package subtitles

import (
	"strings"
	"testing"
)

func TestDedupeSubtitleParts(t *testing.T) {
	parts := dedupeSubtitleParts([]string{"他站在天台。", "他站在天台。", "他缓缓回头。"})
	if len(parts) != 2 {
		t.Fatalf("parts=%v", parts)
	}
}

func TestSplitNarrationLines_commaBoundary(t *testing.T) {
	parts := splitNarrationLines("屏幕亮起，消息刺眼。他按下发送，转身走向黑暗。")
	if len(parts) < 2 {
		t.Fatalf("expected split parts, got %v", parts)
	}
	for _, p := range parts {
		if isPunctuationOnly(p) {
			t.Fatalf("punctuation-only part: %q", p)
		}
	}
}

func TestStyleForResolution(t *testing.T) {
	land := StyleForResolution("1920x1080")
	if land.MaxCharsPerLine != 20 || land.MarginV != 80 {
		t.Fatalf("landscape style=%+v", land)
	}
	port := StyleForResolution("1080x1920")
	if port.MaxCharsPerLine != 14 {
		t.Fatalf("portrait style=%+v", port)
	}
	w, h := PlayResForResolution("1920x1080")
	if w != 1920 || h != 1080 {
		t.Fatalf("playres=%dx%d", w, h)
	}
}

func TestWrapVerticalLines(t *testing.T) {
	got := WrapVerticalLines("她转身离去，复仇从下集开始，真相正在浮出水面。", 14, 3)
	if !strings.Contains(got, `\N`) {
		t.Fatalf("expected multiline, got %q", got)
	}
}
