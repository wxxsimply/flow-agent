// Package tts 语音合成 Provider 接口（火山豆包等，待实现）。
package tts

import "context"

// SynthesizeRequest TTS 输入。
type SynthesizeRequest struct {
	SSML       string
	Voice      string
	Format     string  // 如 mp3
	SpeedRatio float64 // 1.0=正常；0.82=更慢更低沉
	ResourceID string  // 火山 v3 X-Api-Resource-Id（如 seed-tts-2.0）
}

// Client 将 SSML/文本合成为音频字节。
type Client interface {
	Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error)
}
