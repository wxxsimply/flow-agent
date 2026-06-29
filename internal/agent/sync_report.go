package agent

import (
	"math"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const syncDriftToleranceSec = 0.5

func sumTimelineSpeechSec(tl *artifacts.Timeline) float64 {
	if tl == nil {
		return 0
	}
	var sum float64
	for _, s := range tl.Shots {
		if s.AudioDurationSec > 0 {
			sum += s.AudioDurationSec
		}
	}
	return sum
}

// buildSyncReport 生成音画对齐报告：内容 drift（TTS 分段一致性）与合成 drift（槽位 vs 旁白文件）。
func buildSyncReport(tl *artifacts.Timeline, speechAudioSec float64, narrPath string, padSec float64) *artifacts.SyncReport {
	videoTotal := tl.TotalVideoSec()
	contentDrift := math.Abs(sumTimelineSpeechSec(tl) - speechAudioSec)

	audioFileSec := speechAudioSec
	if narrPath != "" {
		if probed, err := ffmpeg.ProbeAudioDurationSec(narrPath); err == nil && probed > 0 {
			audioFileSec = probed
		}
	}
	composeDrift := math.Abs(videoTotal - audioFileSec)
	maxDrift := math.Max(contentDrift, composeDrift)

	return &artifacts.SyncReport{
		VideoTotalSec:   videoTotal,
		AudioTotalSec:   audioFileSec,
		SpeechAudioSec:  speechAudioSec,
		MaxDriftSec:     maxDrift,
		Aligned:         maxDrift <= syncDriftToleranceSec,
		ShotAudioPadSec: padSec,
	}
}

func speechAudioSecFromSegments(segments []artifacts.AudioSegment) float64 {
	if len(segments) == 0 {
		return 0
	}
	last := segments[len(segments)-1]
	return last.StartSec + last.DurationSec
}

// refreshSyncReport 在视频探测后按最新 timeline 与旁白文件重算并写入 sync-report.json。
func refreshSyncReport(rc *runctx.Context, tl *artifacts.Timeline, padSec, speechAudioSec float64) (*artifacts.SyncReport, error) {
	narrPath := ""
	if rc != nil {
		narrPath = rc.ArtifactPath("artifacts/narration.mp3")
	}
	report := buildSyncReport(tl, speechAudioSec, narrPath, padSec)
	if rc != nil {
		outPath := rc.ArtifactPath("artifacts/sync-report.json")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return nil, err
		}
		if err := report.Save(outPath); err != nil {
			return nil, err
		}
	}
	return report, nil
}
