// Package image 文生图 Provider 接口（通义万相等，待实现）。
package image

import "context"

// GenerateRequest 单张图生成参数。
type GenerateRequest struct {
	Prompt         string
	AspectRatio    string // 如 9:16
	Width          int
	Height         int
	ReferenceImage []byte // 可选参考图（img2img / 角色锁定）
}

// Client 根据 prompt 返回图片二进制。
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) ([]byte, error)
}
