package artifacts

import (
	"strings"
	"testing"
)

func TestDedupeNarrations_removesDuplicates(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", Narration: "雨夜，他站在天台。", VisualPrompt: "少年背影，霓虹雨幕", ActionBeats: []string{"起始", "进行", "结束"}},
			{ID: "s02", Narration: "雨夜，他站在天台。", VisualPrompt: "侧脸特写，雨水滑落", ActionBeats: []string{"他缓缓回头", "进行", "结束"}},
			{ID: "s03", Narration: "雨夜，他站在天台。雨夜，他站在天台。", VisualPrompt: "特写眼神"},
		},
	}
	n := sb.DedupeNarrations()
	if n < 1 {
		t.Fatalf("expected cross-shot fix, got %d", n)
	}
	if sb.Shots[1].Narration == sb.Shots[0].Narration {
		t.Fatalf("s02 still duplicate: %q", sb.Shots[1].Narration)
	}
	if strings.Count(sb.Shots[2].Narration, "雨夜，他站在天台。") > 1 {
		t.Fatalf("s03 should dedupe inner sentence: %q", sb.Shots[2].Narration)
	}
}

func TestNarrationsTooSimilar_prefix(t *testing.T) {
	if !NarrationsTooSimilar("雨夜他独自站在天台边缘", "雨夜他独自站在天台边缘望着远方") {
		t.Fatal("expected similar")
	}
}
