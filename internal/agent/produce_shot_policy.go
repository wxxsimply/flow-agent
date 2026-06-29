package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ShotProducePolicy 单镜 produce 策略（由 stack + tier 解析）。
type ShotProducePolicy struct {
	MultiKeyframe bool
	BoNEnabled    bool
	BoNCandidates int
	I2VModel      string
	Resolution    string
}

// ShotProducePolicyFor 解析本镜应使用的关键帧/BoN/模型配置。
func ShotProducePolicyFor(vidCfg config.StackVideoConfig, shot artifacts.Shot) ShotProducePolicy {
	p := ShotProducePolicy{
		BoNEnabled:    vidCfg.WMRewardBoNEnabled,
		BoNCandidates: vidCfg.WMRewardBoNCandidates,
		I2VModel:      vidCfg.Model,
		Resolution:    vidCfg.Resolution,
	}
	mode := strings.ToLower(strings.TrimSpace(vidCfg.KeyframeMode))
	if mode == "" {
		mode = "multi"
	}

	switch mode {
	case "single":
		p.MultiKeyframe = false
	case "tiered":
		if artifacts.IsHeroShot(shot) {
			p.MultiKeyframe = true
			if vidCfg.WMRewardBoNHeroOnly || vidCfg.WMRewardBoNEnabled {
				p.BoNEnabled = true
			}
			if vidCfg.HeroBonCandidates > 0 {
				p.BoNCandidates = vidCfg.HeroBonCandidates
			}
			p.applyQuality(vidCfg)
		} else {
			p.MultiKeyframe = false
			if vidCfg.WMRewardBoNHeroOnly {
				p.BoNEnabled = false
			}
		}
	default: // multi
		p.MultiKeyframe = true
		if artifacts.IsHeroShot(shot) {
			p.applyQuality(vidCfg)
		}
	}

	if strings.EqualFold(vidCfg.UseQualityModelFor, "always") {
		p.applyQuality(vidCfg)
	}
	return p
}

func (p *ShotProducePolicy) applyQuality(vidCfg config.StackVideoConfig) {
	if q := strings.TrimSpace(vidCfg.QualityModel); q != "" {
		p.I2VModel = q
	}
	if r := strings.TrimSpace(vidCfg.HeroResolution); r != "" {
		p.Resolution = r
	}
}

// VidCfgForShot 将 stack 视频配置与镜级策略合并为有效配置副本。
func VidCfgForShot(base config.StackVideoConfig, pol ShotProducePolicy) config.StackVideoConfig {
	out := base
	out.Model = pol.I2VModel
	out.Resolution = pol.Resolution
	out.WMRewardBoNEnabled = pol.BoNEnabled
	out.WMRewardBoNCandidates = pol.BoNCandidates
	return out
}

// UseMultiKeyframeForShot 是否对本镜使用多关键帧路径。
func UseMultiKeyframeForShot(rc *runctx.Context, vidCfg config.StackVideoConfig, shot artifacts.Shot) bool {
	if (IsDirectorRun(rc) || shot.UserSource) && !shot.Expanded {
		return false
	}
	if !isMicroMovieStack(rc) {
		return false
	}
	return ShotProducePolicyFor(vidCfg, shot).MultiKeyframe
}
