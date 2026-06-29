package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestBuildNextHintsFromMetrics(t *testing.T) {
	m := &artifacts.PublishMetrics{
		EpisodeNo:       1,
		Views24h:        10000,
		CompletionRate:  0.25,
		CommentKeywords: []string{"反转"},
	}
	h := BuildNextHintsFromMetrics(2, m)
	if !strings.Contains(h, "完播偏低") {
		t.Fatalf("expected low completion hint: %s", h)
	}
	if !strings.Contains(h, "反转") {
		t.Fatal("expected keywords in hints")
	}
}
