package video

import (
	"context"
	"fmt"
)

// Noop 未配置可灵时使用。
type Noop struct{}

func (Noop) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	_ = ctx
	_ = req
	return "", fmt.Errorf("video: not configured")
}

func (Noop) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	_ = ctx
	_ = req
	return "", fmt.Errorf("video: not configured")
}
