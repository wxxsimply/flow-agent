package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/video"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const soraMotionPromptMaxRunes = 350

func isSoraVideoProvider(vidCfg config.StackVideoConfig) bool {
	return strings.EqualFold(strings.TrimSpace(vidCfg.Provider), "openai")
}

// i2vAPIRequestDurationSec 图生视频 API 请求时长；Sora 对齐 4/8/12 秒档。
func i2vAPIRequestDurationSec(ts artifacts.TimelineShot, vidCfg config.StackVideoConfig) float64 {
	audio := i2vRequestDurationSec(ts)
	if isSoraVideoProvider(vidCfg) {
		return float64(video.SoraDurationSeconds(int(audio + 0.5)))
	}
	return BillableVideoSeconds(audio, vidCfg)
}

// composeTimelineShotDurationSec 合成单镜槽位时长；Sora 按旁白而非 API 返回的 mp4 全长。
func composeTimelineShotDurationSec(ts artifacts.TimelineShot, vidCfg config.StackVideoConfig, padSec float64) float64 {
	if isSoraVideoProvider(vidCfg) && ts.AudioDurationSec > 0 {
		return ts.AudioDurationSec + padSec
	}
	return composeShotDurationSec(ts)
}
