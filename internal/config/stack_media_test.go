package config

import (
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestStackVideoNativePolicy(t *testing.T) {
	s := &Stack{
		TargetDurationSec: 90,
		Video: map[string]any{
			"enabled":    true,
			"all_shots":  true,
			"strategy":   "text2video",
			"skip_image": true,
		},
	}
	vid := s.VideoConfig()
	if !vid.VideoNative() {
		t.Fatal("expected VideoNative")
	}
	if !vid.SkipImage {
		t.Fatal("expected skip_image default true for all_shots")
	}
	pol := s.StoryboardPolicy()
	if !pol.VideoNative() {
		t.Fatal("expected video native storyboard policy")
	}
	if pol.MinShots != artifacts.VideoNativeShortPolicy().MinShots {
		t.Fatalf("policy min shots mismatch")
	}
}
