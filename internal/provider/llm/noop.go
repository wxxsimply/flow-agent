package llm

import "context"

// Noop 无 API Key 时的占位 LLM，便于本地干跑流水线。
type Noop struct{}

// Complete 返回带 [noop] 前缀的固定响应。
func (Noop) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	_ = ctx
	text := "[noop] " + req.User
	return CompletionResult{Text: text, Usage: EstimateTokens(req.System+req.User, text)}, nil
}

// Stream 整段作为单 chunk 回调。
func (Noop) Stream(ctx context.Context, req CompletionRequest, onChunk func(string) error) (TokenUsage, error) {
	_ = ctx
	text := "[noop] " + req.User
	if err := onChunk(text); err != nil {
		return TokenUsage{}, err
	}
	return EstimateTokens(req.System+req.User, text), nil
}
