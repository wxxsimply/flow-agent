package tts

import (
	"context"
	"fmt"
)

// Noop TTS 占位，dry-run 或未配置时使用。
type Noop struct{}

func (Noop) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("tts: not configured")
}
