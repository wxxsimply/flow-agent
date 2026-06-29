package artifacts

import "testing"

func TestCreativeNormalize_preserves30sTarget(t *testing.T) {
	c := &CreativeOptions{TargetDurationSec: 30}
	c.Normalize()
	if c.TargetDurationSec != 30 {
		t.Fatalf("expected 30s target preserved, got %d", c.TargetDurationSec)
	}
}

func TestCreativeNormalize_clampsExtremes(t *testing.T) {
	low := &CreativeOptions{TargetDurationSec: 5}
	low.Normalize()
	if low.TargetDurationSec != 15 {
		t.Fatalf("expected min 15s, got %d", low.TargetDurationSec)
	}
	high := &CreativeOptions{TargetDurationSec: 300}
	high.Normalize()
	if high.TargetDurationSec != 180 {
		t.Fatalf("expected max 180s, got %d", high.TargetDurationSec)
	}
}
