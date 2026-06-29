package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
	"golang.org/x/sync/errgroup"
)

// produceVideoNative 整片动效：每一镜产出 mp4；hard=末帧衔接，soft=同场景弱关联+可换场景，off=各镜独立。
func produceVideoNative(
	ctx context.Context,
	rc *runctx.Context,
	sb *artifacts.Storyboard,
	tl *artifacts.Timeline,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	assetsDir string,
) error {
	initProduceState(rc)
	defer clearProduceState(rc)
	switch vidCfg.ChainMode() {
	case "hard":
		return produceVideoNativeChained(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir, true)
	case "soft":
		return produceVideoNativeParallel(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir, true)
	default:
		return produceVideoNativeParallel(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir, false)
	}
}

func produceVideoNativeParallel(
	ctx context.Context,
	rc *runctx.Context,
	sb *artifacts.Storyboard,
	tl *artifacts.Timeline,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	assetsDir string,
	softContinuity bool,
) error {
	strategy := strings.ToLower(strings.TrimSpace(vidCfg.Strategy))
	if strategy == "" {
		strategy = "image2video"
	}
	parallel := vidCfg.MaxParallelShots
	if parallel <= 0 {
		parallel = 1
	}
	wallStart := time.Now()
	var mu sync.Mutex
	var missing []string
	clipsOK := 0
	imageOK := 0
	videoSec := 0.0

	eg, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, parallel)

	for i := range sb.Shots {
		i, shot := i, sb.Shots[i]
		eg.Go(func() error {
			if checkAccountOverdue(rc) {
				return ErrMediaAccountOverdue
			}
			select {
			case <-gctx.Done():
				return gctx.Err()
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			if parallel <= 1 && i > 0 && vidCfg.InterShotDelaySec > 0 {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case <-time.After(time.Duration(vidCfg.InterShotDelaySec) * time.Second):
				}
			}

			durSec := shotDurationSec(tl, i, shot, vidCfg)
			vidPath := filepath.Join(assetsDir, shot.ID+".mp4")
			var prev *artifacts.Shot
			var opts *produceClipOpts
			if softContinuity && i > 0 {
				p := sb.Shots[i-1]
				if shouldLinkToPreviousShot(p, shot) {
					prev = &p
					opts = &produceClipOpts{prevShot: prev}
				}
			}
			prompt := motionVideoPrompt(rc, vidCfg, shot, prev)
			err := produceShotMotionClip(gctx, strategy, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, opts)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if errors.Is(err, ErrMediaAccountOverdue) {
					return err
				}
				if !errors.Is(err, ErrUseKenBurns) {
					slog.Warn("motion clip failed", "shot", shot.ID, "strategy", strategy, "err", err)
				}
				if vidCfg.KenBurnsFallback {
					if kbErr := produceKenBurnsFallbackClip(shot, durSec, assetsDir, vidPath, i, tl.Resolution); kbErr == nil {
						clipsOK++
						if _, statErr := os.Stat(filepath.Join(assetsDir, shot.ID+".png")); statErr == nil {
							imageOK++
						}
						recordKenBurnsFallbackClip(tl, i, vidPath, durSec, &videoSec)
						return nil
					} else {
						slog.Warn("ken burns fallback failed", "shot", shot.ID, "err", kbErr)
					}
				}
				missing = append(missing, shot.ID)
				return nil
			}
			clipsOK++
			if _, statErr := os.Stat(filepath.Join(assetsDir, shot.ID+".png")); statErr == nil {
				imageOK++
			}
			recordTimelineClip(tl, i, vidPath, durSec, &videoSec)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	rc.RecordImage(imageOK)
	rc.RecordVideo(videoSec)
	slog.Info("produce video native done",
		"shots", len(sb.Shots),
		"clips_ok", clipsOK,
		"parallel", parallel,
		"chained", softContinuity,
		"wall_sec", int(time.Since(wallStart).Seconds()),
		"missing", len(missing),
	)
	if clipsOK < len(sb.Shots) {
		return fmt.Errorf("full-motion: %d/%d shots missing mp4 (failed: %s)",
			len(sb.Shots)-clipsOK, len(sb.Shots), strings.Join(missing, ", "))
	}
	return nil
}

func produceVideoNativeChained(
	ctx context.Context,
	rc *runctx.Context,
	sb *artifacts.Storyboard,
	tl *artifacts.Timeline,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	assetsDir string,
	hardChain bool,
) error {
	strategy := strings.ToLower(strings.TrimSpace(vidCfg.Strategy))
	if strategy == "" {
		strategy = "image2video"
	}
	wallStart := time.Now()
	var missing []string
	clipsOK := 0
	imageOK := 0
	videoSec := 0.0
	chainTail := ""

	for i := range sb.Shots {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if i > 0 && vidCfg.InterShotDelaySec > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(vidCfg.InterShotDelaySec) * time.Second):
			}
		}

		shot := sb.Shots[i]
		var prev *artifacts.Shot
		if i > 0 {
			p := sb.Shots[i-1]
			if shouldLinkToPreviousShot(p, shot) {
				prev = &p
			}
		}
		opts := &produceClipOpts{prevShot: prev}
		if hardChain {
			opts.seedKeyframe = chainTail
		}
		durSec := shotDurationSec(tl, i, shot, vidCfg)
		vidPath := filepath.Join(assetsDir, shot.ID+".mp4")
		prompt := motionVideoPrompt(rc, vidCfg, shot, prev)
		err := produceShotMotionClip(ctx, strategy, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, opts)
		if err != nil {
			if !errors.Is(err, ErrUseKenBurns) {
				slog.Warn("motion clip failed", "shot", shot.ID, "strategy", strategy, "err", err, "chained", true)
			}
			if vidCfg.KenBurnsFallback {
				if kbErr := produceKenBurnsFallbackClip(shot, durSec, assetsDir, vidPath, i, tl.Resolution); kbErr == nil {
					clipsOK++
					if _, statErr := os.Stat(filepath.Join(assetsDir, shot.ID+".png")); statErr == nil {
						imageOK++
					}
					recordKenBurnsFallbackClip(tl, i, vidPath, durSec, &videoSec)
					continue
				} else {
					slog.Warn("ken burns fallback failed", "shot", shot.ID, "err", kbErr)
				}
			}
			missing = append(missing, shot.ID)
			continue
		}
		clipsOK++
		if _, statErr := os.Stat(filepath.Join(assetsDir, shot.ID+".png")); statErr == nil {
			imageOK++
		}
		recordTimelineClip(tl, i, vidPath, durSec, &videoSec)

		if hardChain {
			tailPath := filepath.Join(assetsDir, "_chain_tail.png")
			if extractErr := ffmpeg.ExtractVideoLastFrame(vidPath, tailPath); extractErr == nil {
				chainTail = tailPath
			} else {
				slog.Warn("chain tail extract failed", "shot", shot.ID, "err", extractErr)
			}
		}
	}

	rc.RecordImage(imageOK)
	rc.RecordVideo(videoSec)
	slog.Info("produce video native done",
		"shots", len(sb.Shots),
		"clips_ok", clipsOK,
		"parallel", 1,
		"chained", vidCfg.ChainMode(),
		"wall_sec", int(time.Since(wallStart).Seconds()),
		"missing", len(missing),
	)
	if clipsOK < len(sb.Shots) {
		return fmt.Errorf("full-motion: %d/%d shots missing mp4 (failed: %s)",
			len(sb.Shots)-clipsOK, len(sb.Shots), strings.Join(missing, ", "))
	}
	return nil
}

func shotDurationSec(tl *artifacts.Timeline, i int, shot artifacts.Shot, vidCfg config.StackVideoConfig) float64 {
	if i < len(tl.Shots) {
		return i2vAPIRequestDurationSec(tl.Shots[i], vidCfg)
	}
	return float64(shot.DurationSec)
}

func produceShotMotionClip(
	ctx context.Context,
	strategy string,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	shot artifacts.Shot,
	prompt string,
	durSec float64,
	assetsDir, vidPath string,
	opts *produceClipOpts,
) error {
	switch strategy {
	case "text2video", "all_motion", "motion":
		return produceMotionClipTextFirst(ctx, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath)
	default:
		return produceMotionClipImage2Video(ctx, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, 2, opts)
	}
}

func recordTimelineClip(tl *artifacts.Timeline, i int, vidPath string, durSec float64, videoSec *float64) {
	if i < len(tl.Shots) {
		tl.Shots[i].VisualType = "ai_video"
		tl.Shots[i].AIVideoBudget = true
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
