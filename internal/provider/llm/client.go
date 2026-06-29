// Package llm 定义大语言模型统一调用接口。
package llm

import "context"

// CompletionRequest 单次补全/对话请求。
type CompletionRequest struct {
	Model       string
	System      string
	User        string
	MaxTokens   int
	Temperature float64
	JSONMode    bool // 为 true 时请求 response_format=json_object（DeepSeek/百炼兼容）
}

// Client 支持一次性完成与流式回调。
type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error)
	Stream(ctx context.Context, req CompletionRequest, onChunk func(string) error) (TokenUsage, error)
}
