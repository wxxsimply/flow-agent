package agent

import (
	"testing"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestProduceStageTimeoutSeedanceFourShots(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Name: "micro-movie-seedance",
				Video: map[string]any{
					"enabled":             true,
					"all_shots":           true,
					"provider":            "volcengine",
					"max_parallel_shots":  1,
				},
			},
		},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{{ID: "s01"}, {ID: "s02"}, {ID: "s03"}, {ID: "s04"}},
	}
	got := produceStageTimeout(rc, sb)
	if got < 45*time.Minute {
		t.Fatalf("seedance 4 sequential shots want >=45m, got %v", got)
	}
	if got > produceTimeoutMax {
		t.Fatalf("timeout capped at %v, got %v", produceTimeoutMax, got)
	}
}

func TestProduceStageTimeoutCap5Parallel(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Name: "micro-movie-cap5",
				Video: map[string]any{
					"enabled":            true,
					"all_shots":          true,
					"provider":           "volcengine",
					"max_parallel_shots": 3,
				},
			},
		},
	}
	sb := &artifacts.Storyboard{
		Shots: []artifacts.Shot{{ID: "s01"}, {ID: "s02"}, {ID: "s03"}},
	}
	got := produceStageTimeout(rc, sb)
	if got < 15*time.Minute {
		t.Fatalf("cap5 3 parallel shots want >=15m, got %v", got)
	}
	if got == 8*time.Minute {
		t.Fatal("must not use legacy 8m cap for cap5")
	}
}

func TestProduceStageTimeoutWanQuick(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{Name: "micro-movie-wan-quick"},
		},
	}
	got := produceStageTimeout(rc, nil)
	if got != 15*time.Minute {
		t.Fatalf("wan-quick want 15m, got %v", got)
	}
}
