package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// IsTimeout 是否为客户端/上下文超时（含 awaiting headers）。
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "timeout") ||
		strings.Contains(s, "deadline exceeded") ||
		err == context.DeadlineExceeded
}

// IsRetryable 判断 LLM 调用是否适合重试（空响应、JSON 解码失败、限流等）。
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "unexpected end of json"):
		return true
	case strings.Contains(s, "empty response body"):
		return true
	case strings.Contains(s, "empty message content"):
		return true
	case strings.Contains(s, "empty choices"):
		return true
	case strings.Contains(s, "429"):
		return true
	case strings.Contains(s, "503"):
		return true
	case strings.Contains(s, "502"):
		return true
	case strings.Contains(s, "timeout"):
		return true
	case strings.Contains(s, "connection reset"):
		return true
	case strings.Contains(s, "eof"):
		return true
	default:
		return false
	}
}

func decodeChatResponse(provider string, raw []byte, statusCode int, status string, req CompletionRequest) (CompletionResult, error) {
	body := strings.TrimSpace(string(raw))
	if body == "" {
		return CompletionResult{}, fmt.Errorf("%s: empty response body (status %s)", provider, status)
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		snip := body
		if len(snip) > 240 {
			snip = snip[:240] + "..."
		}
		return CompletionResult{}, fmt.Errorf("%s: decode response: %w (status %s body=%q)", provider, err, status, snip)
	}
	if len(out.Choices) == 0 {
		return CompletionResult{}, fmt.Errorf("%s: empty choices (status %s)", provider, status)
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	if text == "" {
		return CompletionResult{}, fmt.Errorf("%s: empty message content (status %s)", provider, status)
	}
	usage := out.Usage
	if usage.Total() == 0 {
		usage = EstimateTokens(req.System+req.User, text)
	}
	return CompletionResult{Text: text, Usage: usage}, nil
}
