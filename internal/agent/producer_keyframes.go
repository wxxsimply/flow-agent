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
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// produceClipOpts 镜间衔接选项（上一镜末帧 + 上一镜分镜）。
type produceClipOpts struct {
	seedKeyframe string
	prevShot     *artifacts.Shot
}

// produceMotionClipForShot 单镜视频：按 stack/tier 选择单帧或多关键帧路径。
func produceMotionClipForShot(
	ctx context.Context,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	shot artifacts.Shot,
	durSec float64,
	assetsDir, vidPath string,
	mustImage bool,
	opts *produceClipOpts,
) error {
	if shouldSkipExistingShot(vidPath) {
		return nil
	}
	start := time.Now()
	pol := ShotProducePolicyFor(vidCfg, shot)
	effective := VidCfgForShot(vidCfg, pol)
	var err error
	var prev *artifacts.Shot
	seed := ""
	if opts != nil {
		prev = opts.prevShot
		seed = opts.seedKeyframe
	}
	if UseMultiKeyframeForShot(rc, vidCfg, shot) {
		err = produceMotionClipMultiKeyframe(ctx, rc, imgCfg, effective, shot, durSec, assetsDir, vidPath, mustImage, prev, seed)
	} else {
		err = produceMotionClipSingleKeyframe(ctx, rc, imgCfg, effective, shot, durSec, assetsDir, vidPath, mustImage, prev, seed)
	}
	recordProduceTiming(shot, pol, time.Since(start), err)
	markShotCheckpoint(rc, shot.ID, vidPath, err)
	return err
}

func produceMotionClipSingleKeyframe(
	ctx context.Context,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	shot artifacts.Shot,
	durSec float64,
	assetsDir, vidPath string,
	mustImage bool,
	prevShot *artifacts.Shot,
	seedKeyframe string,
) error {
	imgPath := filepath.Join(assetsDir, shot.ID+".png")
	directorUnexpanded := (IsDirectorRun(rc) || shot.UserSource) && !shot.Expanded
	beat := ""
	if directorUnexpanded {
		beat = shot.VisualPrompt
	}
	hardChain := strings.TrimSpace(seedKeyframe) != ""
	kfPrompt := ""
	if beat != "" {
		kfPrompt = ShotKeyframeImagePrompt(rc, shot, beat, prevShot, hardChain)
	} else if shot.Expanded && len(shot.ActionBeats) > 0 {
		kfBeat := keyframeBeatForShot(shot, false)
		kfPrompt = ShotKeyframeImagePrompt(rc, shot, kfBeat, prevShot, hardChain)
		if held := strings.TrimSpace(shot.HeldProps.String()); held != "" {
			kfPrompt += "，与 visual_prompt 握持一致，单帧定格于：" + held
		}
	}
	if err := ensureKeyframeImage(ctx, rc, imgCfg, shot, beat, imgPath, mustImage, kfPrompt, seedKeyframe); err != nil {
		return err
	}
	if shouldSkipI2V(rc) {
		return ErrUseKenBurns
	}
	i2vPrompt := shotMotionPrompt(rc, vidCfg, shot, prevShot)
	appendProducePromptEntry(ProducePromptEntry{
		ShotID:             shot.ID,
		KeyframePrompts:    compactPromptList(kfPrompt),
		SingleMotionPrompt: i2vPrompt,
		BoNEnabled:         vidCfg.WMRewardBoNEnabled && vidCfg.WMRewardBoNCandidates >= 2 && !rc.DryRun,
	})
	return imageToVideoFileBoN(ctx, rc, vidCfg, imgPath, i2vPrompt, durSec, vidPath, &shot)
}

func produceMotionClipMultiKeyframe(
	ctx context.Context,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	vidCfg config.StackVideoConfig,
	shot artifacts.Shot,
	durSec float64,
	assetsDir, vidPath string,
	mustImage bool,
	prevShot *artifacts.Shot,
	seedKeyframe string,
) error {
	beats := actionBeatsForShot(shot)
	if len(beats) < 2 {
		return produceMotionClipSingleKeyframe(ctx, rc, imgCfg, vidCfg, shot, durSec, assetsDir, vidPath, mustImage, prevShot, seedKeyframe)
	}

	tmpDir := filepath.Join(assetsDir, "_kf_"+shot.ID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	var kfPrompts []string
	for i, beat := range beats {
		kfPath := filepath.Join(assetsDir, fmt.Sprintf("%s-k%02d.png", shot.ID, i+1))
		p := ShotKeyframeImagePrompt(rc, shot, beat, prevShot, strings.TrimSpace(seedKeyframe) != "")
		kfPrompts = append(kfPrompts, p)
		seed := ""
		if i == 0 {
			seed = seedKeyframe
		}
		if err := ensureKeyframeImage(ctx, rc, imgCfg, shot, beat, kfPath, mustImage, p, seed); err != nil {
			return err
		}
	}
	firstKF := filepath.Join(assetsDir, shot.ID+"-k01.png")
	mainPNG := filepath.Join(assetsDir, shot.ID+".png")
	if data, err := os.ReadFile(firstKF); err == nil {
		_ = os.WriteFile(mainPNG, data, 0o644)
	}
	if shouldSkipI2V(rc) {
		return ErrUseKenBurns
	}

	nSeg := len(beats) - 1
	segDur := durSec / float64(nSeg)
	if segDur < 2 {
		segDur = 2
	}

	var segPaths []string
	var segPrompts []string
	bon := vidCfg.WMRewardBoNEnabled && vidCfg.WMRewardBoNCandidates >= 2 && !rc.DryRun
	for i := 0; i < nSeg; i++ {
		segPath := filepath.Join(tmpDir, fmt.Sprintf("seg-%02d.mp4", i+1))
		imgPath := filepath.Join(assetsDir, fmt.Sprintf("%s-k%02d.png", shot.ID, i+1))
		motion := ShotSegmentMotionPrompt(rc, vidCfg, shot, beats[i], beats[i+1])
		segPrompts = append(segPrompts, motion)
		if rc.DryRun {
			if err := ffmpeg.RunBlackClip(segPath, int(segDur+0.5)); err != nil {
				return err
			}
		} else if err := imageToVideoFileBoN(ctx, rc, vidCfg, imgPath, motion, segDur, segPath, &shot); err != nil {
			if errors.Is(err, ErrUseKenBurns) {
				return err
			}
			slog.Warn("keyframe segment failed", "shot", shot.ID, "seg", i+1, "err", err)
			if kbErr := produceKenBurnsFallbackClip(shot, segDur, assetsDir, segPath, i, ""); kbErr != nil {
				if err2 := ffmpeg.RunBlackClip(segPath, int(segDur+0.5)); err2 != nil {
					return err2
				}
				slog.Warn("segment ken burns fallback failed, using black clip", "shot", shot.ID, "seg", i+1)
			}
		}
		segPaths = append(segPaths, segPath)
	}

	appendProducePromptEntry(ProducePromptEntry{
		ShotID:               shot.ID,
		KeyframePrompts:      kfPrompts,
		SegmentMotionPrompts: segPrompts,
		BoNEnabled:           bon,
	})

	merged := filepath.Join(tmpDir, "merged.mp4")
	if err := ffmpeg.ConcatVideoClips(segPaths, merged); err != nil {
		return err
	}
	return ffmpeg.TrimVideoToDuration(merged, vidPath, durSec)
}

func ensureKeyframeImage(
	ctx context.Context,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	shot artifacts.Shot,
	beat, outPath string,
	mustImage bool,
	promptOverride string,
	seedKeyframe string,
) error {
	if _, err := os.Stat(outPath); err == nil {
		return nil
	}
	if seed := strings.TrimSpace(seedKeyframe); seed != "" {
		if data, err := os.ReadFile(seed); err == nil && len(data) > 0 {
			return os.WriteFile(outPath, data, 0o644)
		}
	}
	if rc.DryRun {
		label := shot.ID
		if beat != "" {
			label = beat
		}
		return ffmpeg.GeneratePlaceholderPNG(outPath, label)
	}
	if rc.Providers == nil || rc.Providers.Image == nil {
		if mustImage {
			return fmt.Errorf("image provider not initialized")
		}
		return ffmpeg.GeneratePlaceholderPNG(outPath, shot.ID)
	}
	prompt := strings.TrimSpace(promptOverride)
	if prompt == "" {
		prompt = shotImagePrompt(rc, shot)
		if beat != "" {
			prompt = ShotKeyframeImagePrompt(rc, shot, beat, nil, false)
		}
	}
	imgBytes, genErr := generateImageWithFallback(ctx, rc, imgCfg, image.GenerateRequest{
		Prompt:      prompt,
		AspectRatio: imgCfg.AspectRatio,
		Width:       imgCfg.Width,
		Height:      imgCfg.Height,
	})
	if genErr != nil {
		if mustImage {
			return fmt.Errorf("keyframe %s: %w", shot.ID, genErr)
		}
		return ffmpeg.GeneratePlaceholderPNG(outPath, shot.ID)
	}
	return os.WriteFile(outPath, imgBytes, 0o644)
}

func compactPromptList(p string) []string {
	if strings.TrimSpace(p) == "" {
		return nil
	}
	return []string{p}
}

func imageToVideoFile(
	ctx context.Context,
	rc *runctx.Context,
	vidCfg config.StackVideoConfig,
	imgPath, prompt string,
	durSec float64,
	outPath string,
) error {
	if rc.DryRun {
		return ffmpeg.RunBlackClip(outPath, int(durSec+0.5))
	}
	if rc.Providers == nil || rc.Providers.Video == nil {
		return fmt.Errorf("video provider not initialized")
	}
	rc.RecordVideoAPICall(1)
	out, err := imageToVideoWithFallback(ctx, rc, vidCfg, video.ImageToVideoRequest{
		ImagePath:   imgPath,
		Prompt:      prompt,
		DurationSec: int(durSec + 0.5),
		Mode:        vidCfg.Mode,
		Model:       vidCfg.Model,
		Resolution:  vidCfg.Resolution,
		AspectRatio: vidCfg.AspectRatio,
	})
	if err != nil {
		return err
	}
	if out != "" && out != outPath {
		if data, readErr := os.ReadFile(out); readErr == nil {
			return os.WriteFile(outPath, data, 0o644)
		}
	}
	return nil
}

func actionBeatsForShot(shot artifacts.Shot) []string {
	if len(shot.ActionBeats) >= 2 {
		out := make([]string, 0, len(shot.ActionBeats))
		for _, b := range shot.ActionBeats {
			b = strings.TrimSpace(b)
			if b != "" {
				out = append(out, b)
			}
		}
		if len(out) >= 2 {
			return out
		}
	}
	base := strings.TrimSpace(shot.VisualPrompt)
	if base == "" {
		base = "电影镜头"
	}
	return []string{
		base + "，动作起始：角色静止或准备姿态，肢体稳定，符合人体结构",
		base + "，动作进行：单一清晰动作的中间关键帧，小幅位移，禁止肢体交叠",
		base + "，动作结束：动作完成后的稳定姿态，与起始姿态空间一致",
	}
}

func characterAppearanceBlock(rc *runctx.Context) string {
	sheets, err := artifacts.LoadCharacterSheets(rc.ArtifactPath("artifacts/character-sheets.json"))
	if err != nil {
		return ""
	}
	return sheets.AppearanceBlock()
}

func characterViewLockBlock(rc *runctx.Context, visual string) string {
	sheets, err := artifacts.LoadCharacterSheets(rc.ArtifactPath("artifacts/character-sheets.json"))
	if err != nil {
		return ""
	}
	return sheets.ViewLockBlock(visual)
}

func propSheetsForRun(rc *runctx.Context) *artifacts.PropSheets {
	sheets, err := artifacts.LoadPropSheets(rc.ArtifactPath("artifacts/prop-sheets.json"))
	if err != nil {
		return nil
	}
	return sheets
}

func propAppearanceBlock(rc *runctx.Context, shot artifacts.Shot) string {
	sheets := propSheetsForRun(rc)
	if sheets == nil {
		return ""
	}
	props := sheets.PropsForShot(shot)
	return sheets.AppearanceBlock(props)
}

func propViewLockBlock(rc *runctx.Context, shot artifacts.Shot) string {
	sheets := propSheetsForRun(rc)
	if sheets == nil {
		return ""
	}
	return sheets.PropViewLockBlock(shot.VisualPrompt, shot.PropRefs)
}
