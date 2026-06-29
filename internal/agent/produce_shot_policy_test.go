package agent

import (
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestShotProducePolicy_tiered(t *testing.T) {
	vid := config.StackVideoConfig{
		KeyframeMode:          "tiered",
		WMRewardBoNEnabled:    true,
		WMRewardBoNCandidates: 3,
		WMRewardBoNHeroOnly:   true,
		QualityModel:          "wan2.6-i2v",
		UseQualityModelFor:    "hero",
		HeroResolution:        "1080P",
		Model:                 "wan2.6-i2v-flash",
		Resolution:            "720P",
	}
	hero := artifacts.Shot{ID: "s01", Tier: "hero"}
	std := artifacts.Shot{ID: "s02", Tier: "standard"}

	hp := ShotProducePolicyFor(vid, hero)
	if !hp.MultiKeyframe || !hp.BoNEnabled {
		t.Fatalf("hero: multi=%v bon=%v", hp.MultiKeyframe, hp.BoNEnabled)
	}
	if hp.I2VModel != "wan2.6-i2v" || hp.Resolution != "1080P" {
		t.Fatalf("hero quality: model=%s res=%s", hp.I2VModel, hp.Resolution)
	}

	sp := ShotProducePolicyFor(vid, std)
	if sp.MultiKeyframe || sp.BoNEnabled {
		t.Fatalf("standard: multi=%v bon=%v", sp.MultiKeyframe, sp.BoNEnabled)
	}
}

func TestShotProducePolicy_single(t *testing.T) {
	vid := config.StackVideoConfig{
		KeyframeMode:          "single",
		WMRewardBoNEnabled:    true,
		WMRewardBoNCandidates: 3,
		Model:                 "wan2.6-i2v-flash",
	}
	p := ShotProducePolicyFor(vid, artifacts.Shot{ID: "s01"})
	if p.MultiKeyframe {
		t.Fatal("expected single keyframe")
	}
}
