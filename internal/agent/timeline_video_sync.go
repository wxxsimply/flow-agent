package agent

import (
	"log/slog"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// SyncTimelineFromVideoAssets 探测各镜 mp4 时长并回写 timeline（供合成与诊断）。
func SyncTimelineFromVideoAssets(tl *artifacts.Timeline, assetsDir string, padSec float64, vidCfg config.StackVideoConfig) {
	if tl == nil {
		return
	}
	for i := range tl.Shots {
		vidPath := filepath.Join(assetsDir, tl.Shots[i].ID+".mp4")
		probed, err := ffmpeg.ProbeVideoDurationSec(vidPath)
		if err != nil || probed <= 0 {
			continue
		}
		tl.Shots[i].VideoDurationSec = probed
		if isSoraVideoProvider(vidCfg) && tl.Shots[i].AudioDurationSec > 0 {
			tl.Shots[i].DurationSec = composeTimelineShotDurationSec(tl.Shots[i], vidCfg, padSec)
			slog.Debug("timeline video probed",
				"shot", tl.Shots[i].ID,
				"video_sec", probed,
				"audio_sec", tl.Shots[i].AudioDurationSec,
				"compose_sec", tl.Shots[i].DurationSec,
			)
			continue
		}
		target := composeShotDurationSec(tl.Shots[i])
		if probed+padSec > target && tl.Shots[i].AudioDurationSec > 0 {
			target = tl.Shots[i].AudioDurationSec + padSec
		}
		if probed >= target-0.05 {
			tl.Shots[i].DurationSec = probed + padSec
		}
		slog.Debug("timeline video probed",
			"shot", tl.Shots[i].ID,
			"video_sec", probed,
			"audio_sec", tl.Shots[i].AudioDurationSec,
			"compose_sec", tl.Shots[i].DurationSec,
		)
	}
}
