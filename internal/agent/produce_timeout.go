package agent

import (
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const (
	produceTimeoutMin = 15 * time.Minute
	produceTimeoutMax = 120 * time.Minute
)

// produceStageTimeout 按栈、镜数与并行度估算 produce 墙钟上限。
// 旧逻辑对 seedance/cap5 固定 8 分钟，远小于单镜 Seedance 轮询上限，易触发 context deadline exceeded。
func produceStageTimeout(rc *runctx.Context, sb *artifacts.Storyboard) time.Duration {
	shots := 1
	if sb != nil && len(sb.Shots) > 0 {
		shots = len(sb.Shots)
	}

	vidCfg := config.StackVideoConfig{}
	stackName := ""
	if rc != nil && rc.App != nil && rc.App.Stack != nil {
		vidCfg = rc.App.Stack.VideoConfig()
		stackName = rc.App.Stack.Name
	}

	if stackName == "micro-movie-wan-quick" {
		return clampDuration(15*time.Minute, produceTimeoutMin, produceTimeoutMax)
	}

	parallel := vidCfg.MaxParallelShots
	if parallel <= 0 {
		parallel = 1
	}
	batches := (shots + parallel - 1) / parallel

	perShot := perShotProduceEstimate(vidCfg)
	overhead := 5 * time.Minute // TTS、关键帧批处理、FFmpeg 合成
	timeout := overhead + time.Duration(batches)*perShot

	return clampDuration(timeout, produceTimeoutMin, produceTimeoutMax)
}

func perShotProduceEstimate(vidCfg config.StackVideoConfig) time.Duration {
	if !vidCfg.VideoNative() {
		return 6 * time.Minute
	}
	switch strings.ToLower(strings.TrimSpace(vidCfg.Provider)) {
	case "volcengine", "seedance", "ark":
		return 12 * time.Minute
	case "openai", "sora":
		return 15 * time.Minute
	case "kling":
		return 12 * time.Minute
	case "bailian", "dashscope", "wan":
		if strings.Contains(strings.ToLower(vidCfg.Model), "flash") {
			return 8 * time.Minute
		}
		return 10 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func clampDuration(d, min, max time.Duration) time.Duration {
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}
