package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func produceKenBurnsFallbackClip(shot artifacts.Shot, durSec float64, assetsDir, vidPath string, shotIndex int, resolution string) error {
	imgPath := filepath.Join(assetsDir, shot.ID+".png")
	if _, err := os.Stat(imgPath); err != nil {
		return fmt.Errorf("ken burns fallback: no keyframe png: %w", err)
	}
	if durSec <= 0 {
		durSec = 5
	}
	kbIndex := shotIndex
	if strings.TrimSpace(shot.HeldProps.String()) != "" {
		kbIndex = 0 // 持物镜用最小位移 Ken Burns，减道具边缘 warp
	}
	if err := ffmpeg.KenBurnsClipSized(imgPath, vidPath, durSec, 30, "", kbIndex, resolution); err != nil {
		return err
	}
	slog.Debug("using ken burns fallback clip", "shot", shot.ID)
	return nil
}

func recordKenBurnsFallbackClip(tl *artifacts.Timeline, i int, vidPath string, durSec float64, videoSec *float64) {
	if i < len(tl.Shots) {
		tl.Shots[i].VisualType = "ken_burns"
		tl.Shots[i].AIVideoBudget = false
	}
	if probed, perr := ffmpeg.ProbeVideoDurationSec(vidPath); perr == nil && probed > 0 {
		*videoSec += probed
		if i < len(tl.Shots) {
			tl.Shots[i].VideoDurationSec = probed
		}
	} else {
		*videoSec += durSec
	}
}
