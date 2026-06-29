package agent

import (
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestReconcileTimelineToTarget_scalesShortAudio(t *testing.T) {
	rc := &runctx.Context{
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 36},
		App: &config.App{
			Stack: &config.Stack{Name: "micro-movie-seedance", TargetDurationSec: 30},
		},
	}
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{DurationSec: 5},
			{DurationSec: 5},
			{DurationSec: 6},
		},
	}
	reconcileTimelineToTarget(rc, tl, 14)
	got := tl.TotalVideoSec()
	if got < 34 || got > 38 {
		t.Fatalf("expected ~36s video slots, got %.2f", got)
	}
}

func TestReconcileTimelineToTarget_scalesDownLongAudio(t *testing.T) {
	rc := &runctx.Context{
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 30},
		App: &config.App{
			Stack: &config.Stack{Name: "micro-movie-seedance", TargetDurationSec: 30},
		},
	}
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
			{DurationSec: 11, AudioDurationSec: 11},
		},
	}
	reconcileTimelineToTarget(rc, tl, 77)
	got := tl.TotalVideoSec()
	if got < 28 || got > 32 {
		t.Fatalf("expected ~30s video slots after scale down, got %.2f", got)
	}
}

func TestProduceShotCapForTarget(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stack, err := config.LoadStack(filepath.Join(root, "config", "stacks", "micro-movie-seedance.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rc := &runctx.Context{
		Creative: &artifacts.CreativeOptions{TargetDurationSec: 36},
		App:      &config.App{Stack: stack},
	}
	if got := produceShotCapForTarget(rc); got != 8 {
		t.Fatalf("expected 8 shots for 36s/5s clip, got %d", got)
	}
}
