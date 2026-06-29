package compareshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/provider/image"
	"github.com/flow-agent/flow-agent/internal/provider/video"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const defaultPropLockImagePrompt = "竖屏9:16，中景，少年侠客站立，右手横握匕首刀尖向右，左手空，电影感光影，写实"

const defaultPropLockMotionPrompt = "右手始终横握匕首，刀尖向右，左手保持空，小幅迈步，禁止道具换手、消失或变形"

// Result PropLock 同 prompt 双 Provider i2v 对比结果。
type Result struct {
	OutDir      string
	ImagePath   string
	SeedancePath string
	VeoPath     string
	MotionPrompt string
	Err         error
}

// Run 共享一张 Seedream 关键帧，分别用 Seedance 与 Veo Lite 生成 i2v，便于人工对比 PropLock。
func Run(ctx context.Context, root, outDir, imagePrompt, motionPrompt string) Result {
	res := Result{MotionPrompt: motionPrompt}
	if strings.TrimSpace(imagePrompt) == "" {
		imagePrompt = defaultPropLockImagePrompt
	}
	if strings.TrimSpace(motionPrompt) == "" {
		motionPrompt = defaultPropLockMotionPrompt
	}
	res.MotionPrompt = motionPrompt

	propBlock := artifacts.PropLockBlock("右手：匕首；左手：空")
	fullImage := imagePrompt + " " + propBlock
	fullMotion := imagePrompt + " " + propBlock + "，" + motionPrompt

	if outDir == "" {
		outDir = filepath.Join(root, "runs", "proplock-compare-"+time.Now().Format("20060102-150405"))
	}
	res.OutDir = outDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		res.Err = err
		return res
	}

	seedanceApp, err := config.Load(root, "micro-movie-seedance")
	if err != nil {
		res.Err = fmt.Errorf("load seedance stack: %w", err)
		return res
	}
	veoApp, err := config.Load(root, "micro-movie-veo-lite")
	if err != nil {
		res.Err = fmt.Errorf("load veo-lite stack: %w", err)
		return res
	}
	if !seedanceApp.Providers.VolcengineArkConfigured() {
		res.Err = fmt.Errorf("volcengine.api_key required for shared Seedream keyframe")
		return res
	}
	if !veoApp.Providers.GeminiEnabled() {
		res.Err = fmt.Errorf("gemini.api_key required for Veo Lite i2v")
		return res
	}

	imgBundle := provider.NewBundle(seedanceApp)
	if imgBundle.Image == nil {
		res.Err = fmt.Errorf("image provider not configured")
		return res
	}
	imgCfg := seedanceApp.Stack.ImageConfig()
	imgBytes, err := imgBundle.Image.Generate(ctx, image.GenerateRequest{
		Prompt:      fullImage,
		AspectRatio: imgCfg.AspectRatio,
		Width:       imgCfg.Width,
		Height:      imgCfg.Height,
	})
	if err != nil {
		res.Err = fmt.Errorf("seedream keyframe: %w", err)
		return res
	}
	res.ImagePath = filepath.Join(outDir, "keyframe.png")
	if err := os.WriteFile(res.ImagePath, imgBytes, 0o644); err != nil {
		res.Err = err
		return res
	}

	durSeedance := seedanceApp.Stack.VideoConfig().ClipDurationSec
	if durSeedance <= 0 {
		durSeedance = 5
	}
	durVeo := video.VeoDurationSeconds(durSeedance)

	seedanceBundle := provider.NewBundle(seedanceApp)
	if seedanceBundle.Video == nil {
		res.Err = fmt.Errorf("seedance video provider not configured")
		return res
	}
	res.SeedancePath = filepath.Join(outDir, "seedance.mp4")
	if out, err := seedanceBundle.Video.ImageToVideo(ctx, video.ImageToVideoRequest{
		ImagePath:   res.ImagePath,
		Prompt:      fullMotion,
		DurationSec: durSeedance,
		AspectRatio: seedanceApp.Stack.VideoConfig().AspectRatio,
	}); err != nil {
		res.Err = fmt.Errorf("seedance i2v: %w", err)
		return res
	} else if err := copyVideoOut(out, res.SeedancePath); err != nil {
		res.Err = err
		return res
	}

	veoBundle := provider.NewBundle(veoApp)
	if veoBundle.Video == nil {
		res.Err = fmt.Errorf("veo video provider not configured")
		return res
	}
	veoCfg := veoApp.Stack.VideoConfig()
	res.VeoPath = filepath.Join(outDir, "veo-lite.mp4")
	if out, err := veoBundle.Video.ImageToVideo(ctx, video.ImageToVideoRequest{
		ImagePath:   res.ImagePath,
		Prompt:      fullMotion,
		DurationSec: durVeo,
		Model:       veoCfg.Model,
		Resolution:  veoCfg.Resolution,
		AspectRatio: veoCfg.AspectRatio,
	}); err != nil {
		res.Err = fmt.Errorf("veo lite i2v: %w", err)
		return res
	} else if err := copyVideoOut(out, res.VeoPath); err != nil {
		res.Err = err
		return res
	}

	return res
}

func copyVideoOut(src, dst string) error {
	if src == "" || src == dst {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// FormatReport 格式化 PropLock 对比输出。
func FormatReport(r Result, elapsed time.Duration) string {
	var b strings.Builder
	b.WriteString("=== PropLock 对比 (Seedance vs Veo Lite) ===\n")
	if r.Err != nil {
		fmt.Fprintf(&b, "FAIL: %v\n", r.Err)
		return b.String()
	}
	fmt.Fprintf(&b, "OK (%s)\n", elapsed.Round(time.Second))
	fmt.Fprintf(&b, "keyframe: %s\n", r.ImagePath)
	fmt.Fprintf(&b, "seedance: %s\n", r.SeedancePath)
	fmt.Fprintf(&b, "veo-lite: %s\n", r.VeoPath)
	fmt.Fprintf(&b, "motion: %s\n", r.MotionPrompt)
	fmt.Fprintf(&b, "dir: %s\n", r.OutDir)
	b.WriteString("\n人工检查：右手匕首是否全程保持、无换手/消失/变形。\n")
	return b.String()
}
