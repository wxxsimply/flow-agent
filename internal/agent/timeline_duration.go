package agent

import "github.com/flow-agent/flow-agent/pkg/artifacts"

// i2vRequestDurationSec 图生视频请求时长：对齐旁白实测，不用镜尾 pad。
func i2vRequestDurationSec(ts artifacts.TimelineShot) float64 {
	if ts.AudioDurationSec > 0 {
		return ts.AudioDurationSec
	}
	if ts.DurationSec > 0 {
		return ts.DurationSec
	}
	return 5
}

// composeShotDurationSec 合成单镜目标时长（含镜间 pad）。
func composeShotDurationSec(ts artifacts.TimelineShot) float64 {
	if ts.DurationSec > 0 {
		return ts.DurationSec
	}
	if ts.AudioDurationSec > 0 {
		return ts.AudioDurationSec
	}
	return 3
}
