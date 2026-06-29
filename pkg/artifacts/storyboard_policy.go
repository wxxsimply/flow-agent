package artifacts

// StoryboardPolicy 分镜校验策略（由 stack video.enabled / all_shots 等决定）。
type StoryboardPolicy struct {
	MinShots             int
	MaxShots             int
	MinAIVideo           int
	MaxAIVideo           int
	AllShotsAIVideo      bool
	DurationToleranceSec float64
	RelaxDurationTarget  bool // director：总时长由 TTS 实测驱动，不强制对齐 target
}

// DefaultAIVideoPolicy 标准档：含可灵图生视频镜头（4～6 镜）。
func DefaultAIVideoPolicy() StoryboardPolicy {
	return StoryboardPolicy{
		MinShots:   4,
		MaxShots:   12,
		MinAIVideo: MinAIVideoBudgetShots,
		MaxAIVideo: MaxAIVideoBudgetShots,
	}
}

// KenBurnsShortDramaPolicy 竖屏短剧：纯 Ken Burns，镜头更密、无图生视频。
func KenBurnsShortDramaPolicy() StoryboardPolicy {
	return StoryboardPolicy{
		MinShots:   10,
		MaxShots:   18,
		MinAIVideo: 0,
		MaxAIVideo: 0,
	}
}

// VideoNativeShortPolicy 方案 B：全镜 AI 视频（文生/图生视频），成片不依赖 Ken Burns。
func VideoNativeShortPolicy() StoryboardPolicy {
	return StoryboardPolicy{
		MinShots:        9,
		MaxShots:        18,
		MinAIVideo:      9,
		MaxAIVideo:      18,
		AllShotsAIVideo: true,
	}
}

// VideoNative 是否为全镜视频策略。
func (p StoryboardPolicy) VideoNative() bool {
	return p.AllShotsAIVideo
}

// MicroMoviePolicy 微电影：全镜万相 i2v，镜数随片长增加。
func MicroMoviePolicy(stackName string) StoryboardPolicy {
	switch stackName {
	case "micro-movie-economy":
		return StoryboardPolicy{
			MinShots:        12,
			MaxShots:        20,
			MinAIVideo:      4,
			MaxAIVideo:      8,
			AllShotsAIVideo: false,
		}
	case "micro-movie-wan-hd":
		return StoryboardPolicy{
			MinShots:        18,
			MaxShots:        30,
			MinAIVideo:      18,
			MaxAIVideo:      30,
			AllShotsAIVideo: true,
		}
	case "micro-movie-wan-quick", "micro-movie-seedance", "micro-movie-cap5":
		p := StoryboardPolicy{
			MinShots:             3,
			MaxShots:             8,
			MinAIVideo:           3,
			MaxAIVideo:           8,
			AllShotsAIVideo:      true,
			DurationToleranceSec: 20,
		}
		if stackName == "micro-movie-cap5" {
			p.MinShots = 2
			p.MaxShots = 3
			p.MinAIVideo = 2
			p.MaxAIVideo = 3
			p.DurationToleranceSec = 12
		}
		if stackName == "micro-movie-seedance" {
			p.MinShots = 3
			p.MaxShots = 8
			p.MinAIVideo = 3
			p.MaxAIVideo = 8
			p.DurationToleranceSec = 15
		}
		return p
	case "micro-movie-wan-fast":
		return StoryboardPolicy{
			MinShots:             6,
			MaxShots:             12,
			MinAIVideo:           6,
			MaxAIVideo:           12,
			AllShotsAIVideo:      true,
			DurationToleranceSec: 15,
		}
	case "micro-movie-wan-premiere":
		return StoryboardPolicy{
			MinShots:             12,
			MaxShots:             18,
			MinAIVideo:           12,
			MaxAIVideo:           18,
			AllShotsAIVideo:      true,
			DurationToleranceSec: 15,
		}
	default: // micro-movie-wan-flash — 少镜头、长单镜（~10s），总时长 2-3 分钟
		return StoryboardPolicy{
			MinShots:             12,
			MaxShots:             18,
			MinAIVideo:           12,
			MaxAIVideo:           18,
			AllShotsAIVideo:      true,
			DurationToleranceSec: 15,
		}
	}
}

// DirectorPolicy 逐镜导演模式：镜数由用户决定，时长由 TTS 驱动。
func DirectorPolicy() StoryboardPolicy {
	return StoryboardPolicy{
		MinShots:             1,
		MaxShots:             30,
		MinAIVideo:           1,
		MaxAIVideo:           30,
		AllShotsAIVideo:      true,
		DurationToleranceSec: 30,
		RelaxDurationTarget:  true,
	}
}

