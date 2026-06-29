package tts

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/flow-agent/flow-agent/internal/config"
)

// ResourceGrantFallback tries primary (Volcengine); on resource-not-granted errors only,
// falls back to secondary (DashScope) for the rest of the run.
type ResourceGrantFallback struct {
	primary    Client
	secondary  Client
	mu         sync.Mutex
	usePrimary bool
	secondaryErr error // cached after first non-retryable secondary failure
}

// NewResourceGrantFallback wraps Volcengine with DashScope for account permission gaps.
func NewResourceGrantFallback(primary, secondary Client) Client {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}
	return &ResourceGrantFallback{primary: primary, secondary: secondary, usePrimary: true}
}

// IsVolcengineResourceGrantError reports Volcengine 403 when TTS service/voice is not subscribed.
func IsVolcengineResourceGrantError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "resource not granted") ||
		strings.Contains(s, "requested resource not granted") ||
		strings.Contains(s, "code\":3001") ||
		strings.Contains(s, "45000030") ||
		strings.Contains(s, "access denied")
}

func (f *ResourceGrantFallback) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	f.mu.Lock()
	tryPrimary := f.usePrimary
	cached := f.secondaryErr
	f.mu.Unlock()

	if cached != nil {
		return nil, cached
	}

	if tryPrimary {
		data, err := f.primary.Synthesize(ctx, req)
		if err == nil {
			return data, nil
		}
		if !IsVolcengineResourceGrantError(err) {
			return nil, err
		}
		f.mu.Lock()
		f.usePrimary = false
		f.mu.Unlock()
		slog.Warn("volcengine tts not subscribed, falling back to dashscope for narration",
			"hint", "enable 语音合成1.0 in https://console.volcengine.com/speech/service",
			"err", err.Error())
	}

	sec := req
	sec.Voice = config.DashScopeVoiceFor(req.Voice)
	data, err := f.secondary.Synthesize(ctx, sec)
	if err != nil && IsNonRetryableError(err) {
		f.mu.Lock()
		f.secondaryErr = err
		f.mu.Unlock()
	}
	return data, err
}
