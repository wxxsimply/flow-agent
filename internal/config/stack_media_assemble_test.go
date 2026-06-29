package config

import "testing"

func TestAssembleConfig_briefRunes(t *testing.T) {
	s := &Stack{
		Assemble: map[string]any{
			"brief_runes_min":     2000,
			"brief_runes_max":     2400,
			"brief_runes_floor":   1800,
			"brief_segment_count": 2,
			"quick_assemble":      false,
		},
	}
	cfg := s.AssembleConfig()
	if cfg.BriefRunesMin != 2000 || cfg.BriefRunesMax != 2400 || cfg.BriefRunesFloor != 1800 || cfg.BriefSegmentCount != 2 {
		t.Fatalf("unexpected: %+v", cfg)
	}
	if cfg.QuickAssemble {
		t.Fatal("expected quick_assemble false")
	}
}
