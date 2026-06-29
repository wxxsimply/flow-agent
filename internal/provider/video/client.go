// Package video AI 视频 Provider（可灵文生/图生视频等）。
package video

import "context"

// ImageToVideoRequest 图生视频任务参数。
type ImageToVideoRequest struct {
	ImagePath     string
	LastFramePath string // 可选：Gemini Veo 首尾帧插值
	Prompt        string
	DurationSec   int
	Mode          string // std / pro
	Model         string // 可选：覆盖默认 i2v 模型
	Resolution    string // 可选：720P / 1080P
	AspectRatio   string // 可选：9:16 / 16:9 / 1:1
}

// TextToVideoRequest 文生视频任务参数。
type TextToVideoRequest struct {
	OutPath     string
	Prompt      string
	DurationSec int
	Mode        string
	AspectRatio string // 9:16, 16:9, 1:1
}

// Client 提交任务并返回本地 mp4 路径。
type Client interface {
	ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error)
	TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error)
}
