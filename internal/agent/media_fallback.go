package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/image"
	"github.com/flow-agent/flow-agent/internal/provider/video"
	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

// ErrMediaAccountOverdue 火山方舟与百炼均因欠费/余额不可用。
var ErrMediaAccountOverdue = errors.New("媒体 API 账户欠费或余额不足（火山方舟/百炼），请充值后重试")

func isArrearage(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "arrearage") ||
		strings.Contains(msg, "accountoverdue") ||
		strings.Contains(msg, "overdue balance") ||
		strings.Contains(msg, "account has an overdue")
}

func dashScopeAvailable(rc *runctx.Context) bool {
	return rc != nil && rc.App != nil && strings.TrimSpace(rc.App.Providers.DashScope.APIKey) != ""
}

func usesVolcengineMedia(rc *runctx.Context) bool {
	if rc == nil || rc.App == nil || rc.App.Stack == nil {
		return false
	}
	img := strings.ToLower(strings.TrimSpace(rc.App.Stack.ImageConfig().Provider))
	vid := strings.ToLower(strings.TrimSpace(rc.App.Stack.VideoConfig().Provider))
	return isVolcengineProvider(img) || isVolcengineProvider(vid)
}

func isVolcengineProvider(name string) bool {
	switch name {
	case "volcengine", "seedream", "seedance", "jimeng", "ark":
		return true
	default:
		return false
	}
}

func shouldMediaFallback(rc *runctx.Context, err error) bool {
	return err != nil && usesVolcengineMedia(rc) && ve.IsVolcengineFatal(err) && dashScopeAvailable(rc)
}

func dashScopeImageModel(imgCfg config.StackImageConfig) string {
	if m := strings.TrimSpace(imgCfg.Model); strings.HasPrefix(m, "wan") {
		return m
	}
	return "wan2.6-t2i"
}

func generateImageWithFallback(
	ctx context.Context,
	rc *runctx.Context,
	imgCfg config.StackImageConfig,
	req image.GenerateRequest,
) ([]byte, error) {
	if rc.Providers == nil || rc.Providers.Image == nil {
		return nil, fmt.Errorf("image provider not initialized")
	}
	data, err := rc.Providers.Image.Generate(ctx, req)
	if err == nil {
		return data, nil
	}
	if !shouldMediaFallback(rc, err) {
		return nil, err
	}
	model := dashScopeImageModel(imgCfg)
	slog.Debug("volcengine image unavailable, trying dashscope fallback", "model", model, "err", err)
	fb := image.NewDashScope(rc.App.Providers, model)
	if fb == nil {
		return nil, err
	}
	data, fbErr := fb.Generate(ctx, req)
	if fbErr != nil {
		if isArrearage(err) && isArrearage(fbErr) {
			return nil, fmt.Errorf("%w: volcengine=%s; dashscope=%s", ErrMediaAccountOverdue, trimErr(err), trimErr(fbErr))
		}
		return nil, fmt.Errorf("volcengine: %v; dashscope: %w", err, fbErr)
	}
	slog.Debug("image generated via dashscope fallback")
	return data, nil
}

func imageToVideoWithFallback(
	ctx context.Context,
	rc *runctx.Context,
	vidCfg config.StackVideoConfig,
	req video.ImageToVideoRequest,
) (string, error) {
	if shouldSkipI2V(rc) {
		return "", ErrUseKenBurns
	}
	if rc.Providers == nil || rc.Providers.Video == nil {
		return "", fmt.Errorf("video provider not initialized")
	}
	out, err := rc.Providers.Video.ImageToVideo(ctx, req)
	if err == nil {
		return out, nil
	}
	noteProduceAPIError(rc, "", err)
	if !shouldMediaFallback(rc, err) {
		if imageToVideoShouldKenBurns(rc, err) {
			return "", ErrUseKenBurns
		}
		return "", err
	}
	if !video.WanConfigured(rc.App.Providers) {
		if imageToVideoShouldKenBurns(rc, err) {
			return "", ErrUseKenBurns
		}
		return "", err
	}
	wan := video.NewWan(rc.App.Providers, "wan2.6-i2v-flash", "", vidCfg.Resolution, vidCfg.SilentAudio, vidCfg.FastPoll)
	if wan == nil {
		if imageToVideoShouldKenBurns(rc, err) {
			return "", ErrUseKenBurns
		}
		return "", err
	}
	out, fbErr := wan.ImageToVideo(ctx, req)
	if fbErr == nil {
		slog.Info("video generated via wan i2v fallback")
		return out, nil
	}
	noteProduceAPIError(rc, "", fbErr)
	if isArrearage(err) && isArrearage(fbErr) {
		return "", fmt.Errorf("%w: volcengine=%s; wan=%s", ErrMediaAccountOverdue, trimErr(err), trimErr(fbErr))
	}
	if imageToVideoShouldKenBurns(rc, fbErr) || imageToVideoShouldKenBurns(rc, err) {
		return "", ErrUseKenBurns
	}
	return "", fmt.Errorf("volcengine: %v; wan: %w", err, fbErr)
}

func trimErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
