package agent

import (
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/video"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestI2vAPIRequestDurationSec_Sora(t *testing.T) {
	vid := config.StackVideoConfig{Provider: "openai"}
	ts := artifacts.TimelineShot{AudioDurationSec: 5.66}
	got := i2vAPIRequestDurationSec(ts, vid)
	if int(got) != video.SoraDurationSeconds(6) {
		t.Fatalf("5.66s audio -> %v, want sora bucket for 6", got)
	}
}

func TestComposeTimelineShotDurationSec_SoraUsesAudio(t *testing.T) {
	vid := config.StackVideoConfig{Provider: "openai"}
	ts := artifacts.TimelineShot{AudioDurationSec: 5.66, VideoDurationSec: 8.3}
	got := composeTimelineShotDurationSec(ts, vid, 0.05)
	if got < 5.6 || got > 5.8 {
		t.Fatalf("compose should follow audio, got %v", got)
	}
}

func TestIsSoraVideoProvider(t *testing.T) {
	if !isSoraVideoProvider(config.StackVideoConfig{Provider: "openai"}) {
		t.Fatal("expected openai")
	}
	if isSoraVideoProvider(config.StackVideoConfig{Provider: "wan"}) {
		t.Fatal("wan should not be sora")
	}
}
