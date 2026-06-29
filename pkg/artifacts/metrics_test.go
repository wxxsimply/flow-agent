package artifacts

import (
	"strings"
	"testing"
)

func TestPublishMetricsNormalize(t *testing.T) {
	m := &PublishMetrics{EpisodeNo: 1, CompletionRate: 0.5}
	if err := m.Normalize(); err != nil {
		t.Fatal(err)
	}
	if m.Platform != "douyin" {
		t.Fatalf("platform=%q", m.Platform)
	}
	text := m.FormatForPlanner()
	if text == "" || !strings.Contains(text, "完播") {
		t.Fatalf("format: %s", text)
	}
}

func TestPublishMetricsCompletionRange(t *testing.T) {
	m := &PublishMetrics{EpisodeNo: 1, CompletionRate: 1.5}
	if err := m.Normalize(); err == nil {
		t.Fatal("expected error for completion > 1")
	}
}
