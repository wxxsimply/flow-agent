package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/compose/subtitles"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const defaultShotAudioPadSec = 0.15

// produceMediaContext K 阶段音画/字幕/BGM 管线状态。
type produceMediaContext struct {
	Segments    []artifacts.AudioSegment
	Timeline    *artifacts.Timeline
	SyncReport  *artifacts.SyncReport
	ComposeOpts ffmpeg.ComposeOptions
}

func defaultShotPad(rc *runctx.Context) float64 {
	if rc.Workflow == "micro-movie" {
		return 0.05
	}
	if rc.App != nil && rc.App.Stack != nil {
		if v := rc.App.Stack.ComposeConfig().ShotAudioPadSec; v > 0 {
			return v
		}
	}
	return defaultShotAudioPadSec
}

// runProduceMedia K1–K4：按镜 TTS、对齐时间轴、ASS 字幕、可选 BGM。
func runProduceMedia(ctx context.Context, rc *runctx.Context, sb *artifacts.Storyboard, ttsCfg config.StackTTSConfig) (*produceMediaContext, error) {
	if sb != nil {
		if n := sb.DedupeNarrations(); n > 0 {
			slog.Info("deduped shot narrations before TTS", "fixed", n)
		}
	}
	if err := verifyTTSReady(ctx, rc, ttsCfg); err != nil {
		return nil, err
	}
	audioDir := rc.ArtifactPath("artifacts/audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return nil, err
	}

	segments, err := synthesizePerShotAudio(ctx, rc, sb, ttsCfg, audioDir)
	if err != nil {
		return nil, err
	}

	segPaths := make([]string, len(segments))
	for i, s := range segments {
		segPaths[i] = rc.ArtifactPath(s.Path)
	}
	narrPath := rc.ArtifactPath("artifacts/narration.mp3")
	if err := os.MkdirAll(filepath.Dir(narrPath), 0o755); err != nil {
		return nil, fmt.Errorf("narration dir: %w", err)
	}
	if err := ffmpeg.ConcatAudioFiles(segPaths, narrPath); err != nil {
		return nil, fmt.Errorf("concat narration: %w", err)
	}

	audioSegs := &artifacts.AudioSegments{
		Segments:  segments,
		Narration: artifacts.CanonicalWriteRel("artifacts/narration.mp3"),
	}
	if len(segments) > 0 {
		last := segments[len(segments)-1]
		audioSegs.TotalSec = last.StartSec + last.DurationSec
	}
	if err := audioSegs.Save(rc.ArtifactPath("artifacts/audio_segments.json")); err != nil {
		return nil, err
	}

	pad := defaultShotPad(rc)
	tl := artifacts.BuildTimelineAligned(sb, rc.EpisodeNo, segments, pad)
	reconcileTimelineToTarget(rc, tl, audioSegs.TotalSec)
	if rc.Creative != nil {
		tl.Resolution = config.MediaSpecFromCreative(rc.Creative).Resolution
	}

	narrations := make(map[string]string)
	for _, shot := range sb.Shots {
		narrations[shot.ID] = shot.Narration
	}
	subStyle := subtitles.StyleForResolution(tl.Resolution)
	if rc.App != nil && rc.App.Stack != nil {
		cc := rc.App.Stack.ComposeConfig()
		if cc.SubtitleCharsPerLine > 0 {
			subStyle.MaxCharsPerLine = cc.SubtitleCharsPerLine
		}
		if cc.SubtitleMaxLines > 0 {
			subStyle.MaxLines = cc.SubtitleMaxLines
		}
	}
	assPath := rc.ArtifactPath("artifacts/subtitles.ass")
	events := subtitles.EventsFromSegments(segments, narrations, subStyle)
	playX, playY := subtitles.PlayResForResolution(tl.Resolution)
	if err := subtitles.WriteASS(assPath, events, subStyle, playX, playY); err != nil {
		return nil, fmt.Errorf("subtitles.ass: %w", err)
	}
	tl.SubtitleFile = artifacts.CanonicalWriteRel("artifacts/subtitles.ass")

	speechAudioSec := audioSegs.TotalSec
	videoTotal := tl.TotalVideoSec()
	if videoTotal > speechAudioSec+0.25 {
		if err := ffmpeg.PadAudioToDuration(narrPath, videoTotal); err != nil {
			slog.Warn("pad narration to video length failed", "err", err)
		} else if tl.Audio != nil {
			tl.Audio.TotalSec = videoTotal
		}
	} else if speechAudioSec > videoTotal+0.25 {
		if err := ffmpeg.TrimAudioToDuration(narrPath, videoTotal); err != nil {
			slog.Warn("trim narration to video length failed", "err", err)
		}
	}
	sync := buildSyncReport(tl, speechAudioSec, narrPath, pad)
	if err := sync.Save(rc.ArtifactPath("artifacts/sync-report.json")); err != nil {
		return nil, err
	}

	compose := ffmpeg.ComposeOptions{
		RunDir:       rc.RunDir,
		SubtitleASS:  assPath,
		UseGlobalASS: true,
	}
	if rc.App != nil && rc.App.Stack != nil {
		cc := rc.App.Stack.ComposeConfig()
		compose.VideoNativeOnly = cc.VideoNativeOnly || rc.App.Stack.VideoConfig().VideoNative()
		compose.ClipCrossfadeSec = cc.ClipCrossfadeSec
		compose.ClipEdgeFadeSec = cc.ClipEdgeFadeSec
		bgmVol := cc.BGMVolume
		if bgmVol <= 0 {
			bgmVol = 0.22
		}
		bgmPath := ResolveBGMPathForCompose(rc)
		if bgmPath == "" && cc.BGMEnabled {
			bgm := cc.BGMPath
			if bgm != "" && !filepath.IsAbs(bgm) {
				bgm = filepath.Join(rc.App.Root, bgm)
			}
			bgmPath = bgm
		}
		if bgmPath != "" && fileExistsMedia(bgmPath) {
			compose.BGMPath = bgmPath
			compose.BGMVolume = bgmVol
			if plan, err := artifacts.LoadBGMPlan(rc.ArtifactPath("artifacts/bgm-plan.json")); err == nil && plan.Volume > 0 {
				compose.BGMVolume = plan.Volume
			}
		} else if cc.BGMEnabled || (rc.Creative != nil && rc.Creative.BGMMode == "auto") {
			slog.Warn("bgm enabled but no track found", "hint", "add assets/bgm/<mood>.mp3 — see assets/bgm/README.md")
		}
	}

	return &produceMediaContext{
		Segments:    segments,
		Timeline:    tl,
		SyncReport:  sync,
		ComposeOpts: compose,
	}, nil
}

func synthesizePerShotAudio(ctx context.Context, rc *runctx.Context, sb *artifacts.Storyboard, ttsCfg config.StackTTSConfig, audioDir string) ([]artifacts.AudioSegment, error) {
	var segments []artifacts.AudioSegment
	var start float64
	totalChars := 0

	for i, shot := range sb.Shots {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(800 * time.Millisecond):
			}
		}
		text := strings.TrimSpace(shot.Narration)
		relPath := filepath.Join("artifacts/audio", shot.ID+".mp3")
		outPath := rc.ArtifactPath(relPath)

		if rc.DryRun {
			if err := ffmpeg.GenerateSilentMP3(outPath, shot.DurationSec); err != nil {
				return nil, err
			}
		} else if err := synthesizeShotText(ctx, rc, ttsCfg, text, outPath); err != nil {
			if tts.IsNonRetryableError(err) && !tts.UseSilentOnBillingFailure() {
				hint := tts.UserHint(err)
				if hint != "" {
					return nil, fmt.Errorf("shot %s tts: %w (%s)", shot.ID, err, hint)
				}
				return nil, fmt.Errorf("shot %s tts: %w", shot.ID, err)
			}
			slog.Warn("shot tts unavailable, using silent audio", "shot", shot.ID)
			if err := ffmpeg.GenerateSilentMP3(outPath, shot.DurationSec); err != nil {
				return nil, err
			}
		} else {
			totalChars += len([]rune(text))
		}

		dur, err := ffmpeg.ProbeAudioDurationSec(outPath)
		if err != nil || dur <= 0 {
			dur = shot.DurationSec
		}
		segments = append(segments, artifacts.AudioSegment{
			ShotID:      shot.ID,
			Path:        relPath,
			StartSec:    start,
			DurationSec: dur,
		})
		start += dur
		sb.Shots[i].DurationSec = dur
	}
	sb.SyncTotalNarrationSec()
	rc.RecordTTS(totalChars)
	return segments, nil
}

func shouldUseSilentTTS(err error) bool {
	return err != nil && tts.IsNonRetryableError(err) && tts.UseSilentOnBillingFailure()
}

func ttsConfigureHint(provider string) string {
	base := config.ProviderConfigHintZh()
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "volcengine":
		return base + "（volcengine.app_id + access_key）"
	case "dashscope", "bailian", "qwen":
		return base + "（dashscope.api_key）"
	default:
		if p == "" {
			p = "tts"
		}
		return fmt.Sprintf("%s（%s）", base, p)
	}
}

func verifyTTSReady(ctx context.Context, rc *runctx.Context, ttsCfg config.StackTTSConfig) error {
	if rc == nil || rc.DryRun {
		return nil
	}
	if rc.Providers == nil || rc.Providers.TTS == nil {
		return fmt.Errorf("tts not configured")
	}
	if rc.App != nil && strings.EqualFold(strings.TrimSpace(ttsCfg.Provider), "volcengine") &&
		!rc.App.Providers.VolcengineTTSConfigured() {
		return fmt.Errorf("%s tts not configured: %s", strings.TrimSpace(ttsCfg.Provider), ttsConfigureHint(ttsCfg.Provider))
	}
	voice, speed := ResolveNarratorForRun(rc, ttsCfg)
	resourceID := tts.ResolveVolcResourceID(voice, ttsCfg.Product, ttsCfg.ResourceID)
	_, err := rc.Providers.TTS.Synthesize(ctx, tts.SynthesizeRequest{
		SSML:       "旁白可用性检测",
		Voice:      voice,
		Format:     ttsCfg.Format,
		SpeedRatio: speed,
		ResourceID: resourceID,
	})
	if err == nil {
		return nil
	}
	if shouldUseSilentTTS(err) {
		slog.Info("TTS unavailable for this run; video will continue without narration", "hint", tts.UserHint(err))
		return nil
	}
	if hint := tts.UserHint(err); hint != "" {
		return fmt.Errorf("tts unavailable: %w (%s)", err, hint)
	}
	if tts.IsNonRetryableError(err) {
		return fmt.Errorf("tts unavailable: %w", err)
	}
	return nil
}

func synthesizeShotText(ctx context.Context, rc *runctx.Context, cfg config.StackTTSConfig, text, outPath string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty narration")
	}
	if rc.Providers == nil || rc.Providers.TTS == nil {
		return fmt.Errorf("tts not configured")
	}
	if rc.App != nil && strings.EqualFold(strings.TrimSpace(cfg.Provider), "volcengine") &&
		!rc.App.Providers.VolcengineTTSConfigured() {
		return fmt.Errorf("%s tts not configured: %s", strings.TrimSpace(cfg.Provider), ttsConfigureHint(cfg.Provider))
	}
	voice, speed := ResolveNarratorForRun(rc, cfg)
	req := tts.SynthesizeRequest{
		SSML:       text,
		Voice:      voice,
		Format:     cfg.Format,
		SpeedRatio: speed,
		ResourceID: tts.ResolveVolcResourceID(voice, cfg.Product, cfg.ResourceID),
	}
	audio, err := rc.Providers.TTS.Synthesize(ctx, req)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, audio, 0o644)
}

func fileExistsMedia(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
