package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/provider/llm"
)

const maxLLMAttempts = 3

// completeJSONWithRetry 调用 LLM 并要求 JSON 输出；对空响应/解码失败自动重试。
func completeJSONWithRetry(ctx context.Context, client llm.Client, req llm.CompletionRequest) (llm.CompletionResult, error) {
	req.JSONMode = true
	var last error
	for attempt := 0; attempt < maxLLMAttempts; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt*attempt) * time.Second
			if last != nil && llm.IsTimeout(last) {
				wait = 15 * time.Second
			}
			select {
			case <-ctx.Done():
				return llm.CompletionResult{}, ctx.Err()
			case <-time.After(wait):
			}
			slog.Warn("llm retry", "attempt", attempt+1, "model", req.Model, "err", last)
		}
		res, err := client.Complete(ctx, req)
		if err != nil {
			last = err
			if llm.IsRetryable(err) {
				continue
			}
			return llm.CompletionResult{}, err
		}
		if strings.TrimSpace(res.Text) == "" {
			last = fmt.Errorf("empty llm content")
			continue
		}
		return res, nil
	}
	return llm.CompletionResult{}, fmt.Errorf("llm failed after %d attempts: %w", maxLLMAttempts, last)
}
