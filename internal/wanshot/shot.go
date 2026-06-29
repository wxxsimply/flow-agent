package wanshot

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
)

// Result 单镜链路测试结果（stack 决定 t2i / i2v Provider）。
type Result struct {
	OutDir    string
	ImagePath string
	VideoPath string
	Model     string
	Provider  string
	Duration  int
	Err       error
}

// Run 生成一张关键帧 + 一段图生视频。animStyle: 2d | 3d。
func Run(ctx context.Context, app *config.App, animStyle, imagePrompt, motionPrompt string, durationSec int, outDir string) Result {
	res := Result{Duration: durationSec}
	if app == nil || app.Stack == nil {
		res.Err = fmt.Errorf("stack required")
		return res
	}
	if durationSec < 2 {
		res.Duration = 5
	}
	if strings.TrimSpace(outDir) == "" {
		outDir = filepath.Join(os.TempDir(), "flowagent-shot-test")
	}
	res.OutDir = outDir
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		res.Err = err
		return res
	}

	imgCfg := app.Stack.ImageConfig()
	vidCfg := app.Stack.VideoConfig()
	res.Provider = vidCfg.Provider
	res.Model = vidCfg.Model
	preset := config.AnimationStylePreset(animStyle)
	if strings.TrimSpace(imagePrompt) == "" {
		imagePrompt = "竖屏9:16，深夜办公室，年轻程序员面对发光显示器，电影感光影"
	}
	if strings.TrimSpace(motionPrompt) == "" {
		motionPrompt = "镜头缓慢推进，人物转头看向屏幕，表情惊讶"
		if sfx := strings.TrimSpace(vidCfg.MotionPromptSuffix); sfx != "" {
			motionPrompt += sfx
		}
	}

	bundle := provider.NewBundle(app)
	if bundle.Image == nil {
		res.Err = fmt.Errorf("image provider not configured for stack %s", app.Stack.Name)
		return res
	}
	fullImage := preset.ImageStylePrefix + imagePrompt + " [NEG] " + preset.ImageNegative
	imgBytes, err := generateImage(ctx, app, imgCfg, fullImage)
	if err != nil {
		res.Err = fmt.Errorf("t2i: %w", err)
		return res
	}
	res.ImagePath = filepath.Join(outDir, "shot.png")
	if err := os.WriteFile(res.ImagePath, imgBytes, 0o644); err != nil {
		res.Err = err
		return res
	}

	if bundle.Video == nil {
		res.Err = fmt.Errorf("video provider not configured for stack %s (check providers.local.yaml)", app.Stack.Name)
		return res
	}
	res.VideoPath = filepath.Join(outDir, "shot.mp4")
	fullMotion := imagePrompt + preset.VisualGuard + config.GlobalVisualGuard +
		preset.VideoMotionSuffix + "，" + motionPrompt
	dur := res.Duration
	switch strings.ToLower(strings.TrimSpace(vidCfg.Provider)) {
	case "gemini", "veo", "google":
		dur = video.VeoDurationSeconds(res.Duration)
		res.Duration = dur
	case "openai", "sora":
		dur = video.SoraDurationSeconds(res.Duration)
		res.Duration = dur
	}
	out, err := bundle.Video.ImageToVideo(ctx, video.ImageToVideoRequest{
		ImagePath:   res.ImagePath,
		Prompt:      fullMotion,
		DurationSec: dur,
		Model:       vidCfg.Model,
		Resolution:  vidCfg.Resolution,
		AspectRatio: vidCfg.AspectRatio,
	})
	if err != nil {
		res.Err = fmt.Errorf("i2v (%s): %w", vidCfg.Provider, err)
		return res
	}
	if out != "" && out != res.VideoPath {
		if data, readErr := os.ReadFile(out); readErr == nil {
			_ = os.WriteFile(res.VideoPath, data, 0o644)
		} else {
			res.VideoPath = out
		}
	}
	return res
}

// FormatReport 格式化单镜测试输出。
func FormatReport(r Result, elapsed time.Duration) string {
	var b strings.Builder
	b.WriteString("=== 单镜测试 (t2i + i2v) ===\n")
	if r.Err != nil {
		fmt.Fprintf(&b, "FAIL: %v\n", r.Err)
		return b.String()
	}
	fmt.Fprintf(&b, "OK (%s)\n", elapsed.Round(time.Second))
	if r.Provider != "" {
		fmt.Fprintf(&b, "provider: %s\n", r.Provider)
	}
	fmt.Fprintf(&b, "model: %s\n", r.Model)
	fmt.Fprintf(&b, "duration: %ds\n", r.Duration)
	fmt.Fprintf(&b, "image: %s\n", r.ImagePath)
	fmt.Fprintf(&b, "video: %s\n", r.VideoPath)
	fmt.Fprintf(&b, "dir: %s\n", r.OutDir)
	return b.String()
}

func generateImage(ctx context.Context, app *config.App, imgCfg config.StackImageConfig, prompt string) ([]byte, error) {
	bundle := provider.NewBundle(app)
	if bundle.Image == nil {
		return nil, fmt.Errorf("image provider not configured")
	}
	req := image.GenerateRequest{
		Prompt:      prompt,
		AspectRatio: imgCfg.AspectRatio,
		Width:       imgCfg.Width,
		Height:      imgCfg.Height,
	}
	data, err := bundle.Image.Generate(ctx, req)
	if err == nil {
		return data, nil
	}
	imgProv := strings.ToLower(strings.TrimSpace(imgCfg.Provider))
	if !isVolcengineImageProvider(imgProv) || strings.TrimSpace(app.Providers.DashScope.APIKey) == "" {
		return nil, fmt.Errorf("t2i: %w", err)
	}
	fb := image.NewDashScope(app.Providers, dashScopeImageModel(imgCfg))
	if fb == nil {
		return nil, fmt.Errorf("t2i: %w", err)
	}
	data, fbErr := fb.Generate(ctx, req)
	if fbErr != nil {
		return nil, fmt.Errorf("t2i: %w; dashscope fallback: %v", err, fbErr)
	}
	return data, nil
}

func isVolcengineImageProvider(name string) bool {
	switch name {
	case "volcengine", "seedream", "jimeng", "ark":
		return true
	default:
		return false
	}
}

func dashScopeImageModel(imgCfg config.StackImageConfig) string {
	if m := strings.TrimSpace(imgCfg.Model); strings.HasPrefix(m, "wan") {
		return m
	}
	return "wan2.6-t2i"
}
