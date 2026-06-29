package cost

import (
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestCompareTargets(t *testing.T) {
	ledger := &artifacts.CostLedger{
		LLMCNY: 7, TTSCNY: 4, ImageCNY: 15, VideoCNY: 0, OtherCNY: 0,
	}
	ledger.Recalc()
	targets := map[string][]float64{
		"llm":   {5, 10},
		"tts":   {3, 6},
		"image": {12, 22},
		"total": {25, 60},
	}
	checks := CompareTargets(ledger, targets)
	if len(checks) != 6 {
		t.Fatalf("got %d checks", len(checks))
	}
	for _, c := range checks {
		if c.Category == "llm" && !c.InRange {
			t.Fatalf("llm should be in range: %+v", c)
		}
		if c.Category == "video" && !c.InRange {
			// video 0 with no target in map - HasTarget false, InRange true
		}
	}
	ledger.ImageCNY = 99
	ledger.Recalc()
	checks = CompareTargets(ledger, targets)
	for _, c := range checks {
		if c.Category == "image" && c.InRange {
			t.Fatal("image should be out of range")
		}
	}
}
