package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunProducer 生成 assets、timeline.json 并合成 master.mp4。
func RunProducer(ctx context.Context, rc *runctx.Context) error {
	if rc.DryRun {
		return runProducerDryRun(rc)
	}
	return runProducerLive(ctx, rc)
}

func runProducerDryRun(rc *runctx.Context) error {
	LoadCreativeOptionsFromRun(rc)
	sb, err := loadStoryboardForProduce(rc)
	if err != nil {
		return err
	}
	ttsCfg := config.StackTTSConfig{}
	if rc.App != nil && rc.App.Stack != nil {
		ttsCfg = rc.App.Stack.TTSConfig()
	}
	media, err := runProduceMedia(context.Background(), rc, sb, ttsCfg)
	if err != nil {
		return err
	}

	assetsDir := rc.ArtifactPath(artifacts.ShotsDir)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	vidCfg := config.StackVideoConfig{}
	if rc.App != nil && rc.App.Stack != nil {
		vidCfg = rc.App.Stack.VideoConfig()
	}
	vidCfg = config.ApplyMediaSpecVideo(vidCfg, config.MediaSpecFromCreative(rc.Creative))
	if err := produceVisualAssets(context.Background(), rc, sb, media.Timeline, config.StackImageConfig{}, vidCfg, assetsDir, true); err != nil {
		return err
	}

	if err := media.Timeline.Save(rc.ArtifactPath("artifacts/timeline.json")); err != nil {
		return err
	}
	opts := media.ComposeOpts
	opts.RunDir = rc.RunDir
	opts.DryRun = true
	opts.DurationSec = rc.TargetDurationSec()
	opts.Timeline = media.Timeline
	if err := ffmpeg.Compose(opts); err != nil {
		return err
	}
	return recordProducerArtifacts(rc, media.Timeline)
}

func runProducerLive(parent context.Context, rc *runctx.Context) error {
	LoadCreativeOptionsFromRun(rc)
	if rc.Providers == nil {
		return fmt.Errorf("providers not initialized")
	}
	if rc.App == nil || !rc.App.Providers.MediaProduceConfigured(rc.App.Stack) {
		stackName := "micro-movie-seedance"
		if rc.App != nil && rc.App.Stack != nil && strings.TrimSpace(rc.App.Stack.Name) != "" {
			stackName = rc.App.Stack.Name
		}
		return fmt.Errorf("produce requires media API keys for stack %q; %s (or use --dry-run)", stackName, config.ProviderConfigHintZh())
	}
	sb, err := loadStoryboardForProduce(rc)
	if err != nil {
		return err
	}
	if err := CheckProduceBudget(rc, sb); err != nil {
		return err
	}

	imgCfg := rc.App.Stack.ImageConfig()
	imgCfg = config.ApplyMediaSpec(imgCfg, config.MediaSpecFromCreative(rc.Creative))
	ttsCfg := rc.App.Stack.TTSConfig()

	if parent == nil {
		parent = context.Background()
	}
	timeout := produceStageTimeout(rc, sb)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	slog.Info("produce timeout budget", "timeout", timeout.String(), "shots", len(sb.Shots))

	// K1/K3/K4：按镜 TTS、对齐时间轴、ASS、sync-report
	media, err := runProduceMedia(ctx, rc, sb, ttsCfg)
	if err != nil {
		recordProduceFailure(rc, err)
		return fmt.Errorf("produce media: %w", err)
	}
	tl := media.Timeline

	if err := sb.Save(rc.ArtifactPath("artifacts/storyboard.json")); err != nil {
		return fmt.Errorf("save storyboard after TTS: %w", err)
	}
	rc.RecordArtifact("storyboard.json", "artifacts/storyboard.json", true)

	if cp := loadProduceCheckpoint(rc); len(cp.CompletedShots) > 0 {
		slog.Info("produce checkpoint", "skip_completed", len(cp.CompletedShots), "shots", cp.CompletedShots)
	}

	assetsDir := rc.ArtifactPath(artifacts.ShotsDir)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}

	vidCfg := rc.App.Stack.VideoConfig()
	vidCfg = config.ApplyMediaSpecVideo(vidCfg, config.MediaSpecFromCreative(rc.Creative))
	if err := produceVisualAssets(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir, false); err != nil {
		recordProduceFailure(rc, err)
		return err
	}
	pad := defaultShotPad(rc)
	if media.SyncReport != nil && media.SyncReport.ShotAudioPadSec > 0 {
		pad = media.SyncReport.ShotAudioPadSec
	}
	SyncTimelineFromVideoAssets(tl, assetsDir, pad, vidCfg)

	if err := tl.Save(rc.ArtifactPath("artifacts/timeline.json")); err != nil {
		return err
	}

	speechSec := speechAudioSecFromSegments(media.Segments)
	syncReport, err := refreshSyncReport(rc, tl, pad, speechSec)
	if err != nil {
		recordProduceFailure(rc, err)
		return err
	}
	media.SyncReport = syncReport

	if !ffmpeg.Available() {
		slog.Warn("ffmpeg not found", "hint", "set FFMPEG_PATH or restart terminal after winget install Gyan.FFmpeg")
	} else {
		slog.Info("ffmpeg resolved", "path", ffmpeg.BinPath())
	}
	opts := media.ComposeOpts
	opts.RunDir = rc.RunDir
	opts.DryRun = false
	opts.DurationSec = rc.TargetDurationSec()
	opts.Timeline = tl
	if err := ffmpeg.Compose(opts); err != nil {
		recordProduceFailure(rc, err)
		return err
	}
	if w, h, err := ffmpeg.ProbeVideoResolution(rc.ArtifactPath("artifacts/master.mp4")); err == nil && w > 0 && h > 0 {
		tl.Resolution = fmt.Sprintf("%dx%d", w, h)
		if err := tl.Save(rc.ArtifactPath("artifacts/timeline.json")); err != nil {
			return err
		}
	}
	slog.Info("av sync", "aligned", media.SyncReport.Aligned, "drift_sec", media.SyncReport.MaxDriftSec)
	return recordProducerArtifacts(rc, tl)
}

func recordProduceFailure(rc *runctx.Context, err error) {
	if rc == nil || rc.Manifest == nil || err == nil {
		return
	}
	rc.Manifest.LastError = err.Error()
	rc.Manifest.Stage = "failed"
	_ = rc.SaveManifest()
}

func synthesizeNarration(ctx context.Context, rc *runctx.Context, cfg config.StackTTSConfig, ssml, outPath string) error {
	if rc.Providers == nil || rc.Providers.TTS == nil {
		return fmt.Errorf("tts not configured")
	}
	voice, speed := ResolveNarratorForRun(rc, cfg)
	req := tts.SynthesizeRequest{
		SSML:       ssml,
		Voice:      voice,
		Format:     cfg.Format,
		SpeedRatio: speed,
		ResourceID: tts.ResolveVolcResourceID(voice, cfg.Product, cfg.ResourceID),
	}
	audio, err := rc.Providers.TTS.Synthesize(ctx, req)
	if err != nil {
		slog.Warn("tts api failed, generating silent track", "err", err)
		sb, _ := loadStoryboardForProduce(rc)
		dur := float64(rc.TargetDurationSec())
		if sb != nil {
			dur = sb.TotalDurationSec()
		}
		return ffmpeg.GenerateSilentMP3(outPath, dur)
	}
	return os.WriteFile(outPath, audio, 0o644)
}

func checkAudioDuration(sb *artifacts.Storyboard, narrPath string) {
	if sb == nil {
		return
	}
	dur, err := ffmpeg.ProbeAudioDurationSec(narrPath)
	if err != nil {
		return
	}
	target := sb.TotalDurationSec()
	if target <= 0 {
		return
	}
	diff := (dur - target) / target
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.05 {
		slog.Warn("tts duration mismatch", "audio_sec", dur, "target_sec", target, "diff_pct", diff*100)
	}
}

func loadStoryboardForProduce(rc *runctx.Context) (*artifacts.Storyboard, error) {
	path := rc.ArtifactPath("artifacts/storyboard.json")
	sb, err := artifacts.LoadStoryboard(path)
	if err != nil {
		return nil, fmt.Errorf("load storyboard: %w", err)
	}
	pol := StoryboardPolicyForRun(rc)
	if err := sb.Validate(rc.EpisodeNo, rc.TargetDurationSec(), pol); err != nil {
		return nil, fmt.Errorf("storyboard: %w", err)
	}
	applyStoryboardProduceCaps(rc, sb)
	return sb, nil
}

func recordProducerArtifacts(rc *runctx.Context, tl *artifacts.Timeline) error {
	rc.RecordArtifact("timeline.json", "artifacts/timeline.json", true)
	rc.RecordArtifact("master.mp4", "artifacts/master.mp4", true)
	rc.RecordArtifact("narration.mp3", "artifacts/narration.mp3", false)
	rc.RecordArtifact("assets/", "artifacts/assets/", false)
	rc.RecordArtifact("audio_segments.json", "artifacts/audio_segments.json", false)
	rc.RecordArtifact("sync-report.json", "artifacts/sync-report.json", false)
	rc.RecordArtifact("subtitles.ass", "artifacts/subtitles.ass", false)
	rc.RecordArtifact("produce-degradation.json", "artifacts/produce-degradation.json", false)
	rc.RecordArtifact("produce-checkpoint.json", "artifacts/produce-checkpoint.json", false)
	_ = tl
	return nil
}

var ssmlTagRe = regexp.MustCompile(`(?i)</?[^>]+>`)

func narrationCharCount(ssml string) int {
	plain := ssmlTagRe.ReplaceAllString(ssml, "")
	plain = strings.TrimSpace(plain)
	return utf8.RuneCountInString(plain)
}
