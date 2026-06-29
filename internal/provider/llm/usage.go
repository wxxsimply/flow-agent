package llm

import "unicode/utf8"

// TokenUsage LLM 用量（与 OpenAI usage 字段对齐）。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Total 返回总 token 数。
func (u TokenUsage) Total() int {
	return u.PromptTokens + u.CompletionTokens
}

// CompletionResult 非流式补全结果。
type CompletionResult struct {
	Text  string
	Usage TokenUsage
}

// EstimateTokens 在 API 未返回 usage 时按字符粗估（中文约 1.5 字符/token）。
func EstimateTokens(prompt, completion string) TokenUsage {
	const charsPerToken = 1.5
	p := utf8.RuneCountInString(prompt)
	c := utf8.RuneCountInString(completion)
	return TokenUsage{
		PromptTokens:     int(float64(p) / charsPerToken),
		CompletionTokens: int(float64(c) / charsPerToken),
	}
}

// MergeUsage 合并多次调用的用量。
func MergeUsage(a, b TokenUsage) TokenUsage {
	return TokenUsage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
	}
}
