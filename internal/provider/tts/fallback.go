package tts

import (
	"context"
	"log/slog"
	"sync"

	"github.com/flow-agent/flow-agent/internal/config"
)

// Fallback 先尝试 primary（如火山），失败后固定走 secondary（如百炼），避免每镜重复失败请求。
type Fallback struct {
	primary   Client
	secondary Client
	mu        sync.Mutex
	usePrimary bool
}

// NewFallback 构造带音色映射的回退 TTS 客户端。
func NewFallback(primary, secondary Client) Client {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}
	return &Fallback{primary: primary, secondary: secondary, usePrimary: true}
}

func (f *Fallback) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	f.mu.Lock()
	tryPrimary := f.usePrimary
	f.mu.Unlock()

	if tryPrimary {
		data, err := f.primary.Synthesize(ctx, req)
		if err == nil {
			return data, nil
		}
		f.mu.Lock()
		f.usePrimary = false
		f.mu.Unlock()
		slog.Debug("tts fallback to dashscope", "reason", err.Error())
	}

	sec := req
	sec.Voice = config.DashScopeVoiceFor(req.Voice)
	return f.secondary.Synthesize(ctx, sec)
}
