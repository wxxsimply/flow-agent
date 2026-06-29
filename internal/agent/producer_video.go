package agent



import (

	"context"

	"errors"

	"fmt"

	"log/slog"

	"os"

	"path/filepath"

	"strings"

	"time"



	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"

	"github.com/flow-agent/flow-agent/internal/config"

	"github.com/flow-agent/flow-agent/internal/provider/image"

	"github.com/flow-agent/flow-agent/internal/provider/video"

	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"

	"github.com/flow-agent/flow-agent/internal/runctx"

	"github.com/flow-agent/flow-agent/pkg/artifacts"

)



func produceVisualAssets(

	ctx context.Context,

	rc *runctx.Context,

	sb *artifacts.Storyboard,

	tl *artifacts.Timeline,

	imgCfg config.StackImageConfig,

	vidCfg config.StackVideoConfig,

	assetsDir string,

	dryRun bool,

) error {
	resetProducePrompts()
	resetProduceTimings()
	resetBonScores()
	defer func() {
		_ = flushProducePrompts(rc)
		_ = flushProduceTimings(rc)
		if !dryRun && sb != nil && tl != nil {
			_ = flushProduceDegradation(rc, sb, tl)
		}
	}()

	if dryRun {

		return produceVisualAssetsDryRun(sb, vidCfg, assetsDir)

	}
	seedFirstShotFromTurnaround(rc, imgCfg, assetsDir)

	if vidCfg.VideoNative() {

		if rc.Providers == nil || rc.Providers.Video == nil {
			vidProv := "kling"
			if rc.App != nil && rc.App.Stack != nil {
				vidProv = rc.App.Stack.VideoConfig().Provider
			}
			switch strings.ToLower(vidProv) {
			case "bailian", "dashscope", "wan":
				return fmt.Errorf("video-native stack requires dashscope api_key (flowagent config test-wan-video)")
			case "volcengine", "seedance", "ark":
				return fmt.Errorf("video-native stack requires volcengine.api_key (方舟 Seedance，flowagent test-shot --stack micro-movie-wan-flash)")
			default:
				return fmt.Errorf("video-native stack requires kling access_key + secret_key (flowagent config test-kling)")
			}
		}

		return produceVideoNative(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir)

	}

	return produceHybridKenBurns(ctx, rc, sb, tl, imgCfg, vidCfg, assetsDir)

}



func produceVisualAssetsDryRun(sb *artifacts.Storyboard, vidCfg config.StackVideoConfig, assetsDir string) error {

	for _, shot := range sb.Shots {

		if vidCfg.VideoNative() || shot.AIVideoBudget {

			vidPath := filepath.Join(assetsDir, shot.ID+".mp4")

			sec := int(shot.DurationSec + 0.5)

			if sec < 1 {

				sec = 5

			}

			if err := ffmpeg.RunBlackClip(vidPath, sec); err != nil {

				return err

			}

			continue

		}

		imgPath := filepath.Join(assetsDir, shot.ID+".png")

		if err := ffmpeg.GeneratePlaceholderPNG(imgPath, shot.ID); err != nil {

			return err

		}

	}

	return nil

}



func motionVideoPrompt(rc *runctx.Context, vidCfg config.StackVideoConfig, shot artifacts.Shot, prevShot *artifacts.Shot) string {
	return shotMotionPrompt(rc, vidCfg, shot, prevShot)
}



// produceMotionClipTextFirst 文生视频；失败则必须图生视频成功。

func produceMotionClipTextFirst(

	ctx context.Context,

	rc *runctx.Context,

	imgCfg config.StackImageConfig,

	vidCfg config.StackVideoConfig,

	shot artifacts.Shot,

	prompt string,

	durSec float64,

	assetsDir, vidPath string,

) error {

	if t2vErr := klingText2Video(ctx, rc, vidCfg, prompt, durSec, vidPath); t2vErr == nil {
		return nil
	}
	slog.Warn("text2video failed, using image2video", "shot", shot.ID)

	return produceMotionClipImage2Video(ctx, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, 2, nil)

}



func produceMotionClipImage2Video(

	ctx context.Context,

	rc *runctx.Context,

	imgCfg config.StackImageConfig,

	vidCfg config.StackVideoConfig,

	shot artifacts.Shot,

	prompt string,

	durSec float64,

	assetsDir, vidPath string,

	maxAttempts int,

	opts *produceClipOpts,

) error {

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {

		if attempt > 1 {

			select {

			case <-ctx.Done():

				return ctx.Err()

			case <-time.After(12 * time.Second):

			}

			slog.Warn("image2video retry", "shot", shot.ID, "attempt", attempt)

		}

		lastErr = produceMotionClipForShot(ctx, rc, imgCfg, vidCfg, shot, durSec, assetsDir, vidPath, true, opts)

		if lastErr == nil {

			return nil

		}

		if errors.Is(lastErr, ErrUseKenBurns) {

			return lastErr

		}

		if !ve.IsVolcengineRetryable(lastErr) {

			return lastErr

		}

	}

	return lastErr

}



func produceHybridKenBurns(

	ctx context.Context,

	rc *runctx.Context,

	sb *artifacts.Storyboard,

	tl *artifacts.Timeline,

	imgCfg config.StackImageConfig,

	vidCfg config.StackVideoConfig,

	assetsDir string,

) error {

	imageOK := 0

	for i, shot := range sb.Shots {

		if i > 0 {

			select {

			case <-ctx.Done():

				return ctx.Err()

			case <-time.After(3 * time.Second):

			}

		}

		imgPath := filepath.Join(assetsDir, shot.ID+".png")

		imgBytes, err := generateImageWithFallback(ctx, rc, imgCfg, image.GenerateRequest{

			Prompt:      shotImagePrompt(rc, shot),

			AspectRatio: imgCfg.AspectRatio,

			Width:       imgCfg.Width,

			Height:      imgCfg.Height,

		})

		if err != nil {

			slog.Warn("image generate failed, using placeholder", "shot", shot.ID, "err", err)

			if err := ffmpeg.GeneratePlaceholderPNG(imgPath, shot.ID); err != nil {

				return err

			}

			continue

		}

		if err := os.WriteFile(imgPath, imgBytes, 0o644); err != nil {

			return err

		}

		imageOK++

	}

	rc.RecordImage(imageOK)



	if !vidCfg.Enabled || rc.Providers.Video == nil {

		slog.Info("produce mode", "video", "ken_burns_only")

		return nil

	}



	videoSec := 0.0

	klingOK := 0

	for i, shot := range sb.Shots {

		if !shot.AIVideoBudget {

			continue

		}

		if klingOK > 0 {

			select {

			case <-ctx.Done():

				return ctx.Err()

			case <-time.After(8 * time.Second):

			}

		}

		durSec := shot.DurationSec
		if i < len(tl.Shots) {
			durSec = i2vRequestDurationSec(tl.Shots[i])
		}

		vidPath := filepath.Join(assetsDir, shot.ID+".mp4")

		prompt := strings.TrimSpace(imgCfg.StylePrefix + shot.VisualPrompt)

		err := klingImage2VideoWithOptionalImage(ctx, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, false)

		if err != nil {

			slog.Warn("kling retry", "shot", shot.ID, "attempt", 2)

			select {

			case <-ctx.Done():

				return ctx.Err()

			case <-time.After(12 * time.Second):

			}

			err = klingImage2VideoWithOptionalImage(ctx, rc, imgCfg, vidCfg, shot, prompt, durSec, assetsDir, vidPath, false)

		}

		if err != nil {

			slog.Warn("kling skipped, ken_burns fallback", "shot", shot.ID, "err", err)

			continue

		}

		klingOK++

		if i < len(tl.Shots) {

			tl.Shots[i].VisualType = "ai_video"

		}

		if probed, perr := ffmpeg.ProbeVideoDurationSec(vidPath); perr == nil && probed > 0 {
			videoSec += probed
			if i < len(tl.Shots) {
				tl.Shots[i].VideoDurationSec = probed
			}
		} else {
			videoSec += durSec
		}

	}

	if klingOK > 0 {

		slog.Info("kling clips ready", "count", klingOK, "model", vidCfg.Model)

	}

	rc.RecordVideo(videoSec)

	return nil

}



func klingText2Video(ctx context.Context, rc *runctx.Context, vidCfg config.StackVideoConfig, prompt string, durSec float64, outPath string) error {

	_, err := rc.Providers.Video.TextToVideo(ctx, video.TextToVideoRequest{

		OutPath:     outPath,

		Prompt:      prompt,

		DurationSec: int(durSec + 0.5),

		Mode:        vidCfg.Mode,

		AspectRatio: vidCfg.AspectRatio,

	})

	return err

}



func klingImage2VideoWithOptionalImage(

	ctx context.Context,

	rc *runctx.Context,

	imgCfg config.StackImageConfig,

	vidCfg config.StackVideoConfig,

	shot artifacts.Shot,

	prompt string,

	durSec float64,

	assetsDir, vidPath string,

	mustImage bool,

) error {
	_ = prompt
	return produceMotionClipForShot(ctx, rc, imgCfg, vidCfg, shot, durSec, assetsDir, vidPath, mustImage, nil)
}


