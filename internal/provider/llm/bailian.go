package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
)

// Bailian 阿里云百炼 OpenAI 兼容 Chat API（Qwen 等）。
type Bailian struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewBailian 从 DashScope/百炼配置构造客户端。
func NewBailian(p config.Providers) *Bailian {
	return &Bailian{
		apiKey:  p.DashScope.APIKey,
		baseURL: strings.TrimRight(p.ResolveDashScopeBaseURL(), "/"),
		http:    NewHTTPClient(true),
	}
}

// Complete 非流式对话补全。
func (b *Bailian) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	body := buildChatPayload(req, false)
	data, err := json.Marshal(body)
	if err != nil {
		return CompletionResult{}, err
	}
	url := b.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return CompletionResult{}, fmt.Errorf("bailian: %s: %s", resp.Status, string(raw))
	}
	return decodeChatResponse("bailian", raw, resp.StatusCode, resp.Status, req)
}

// Stream 流式补全（SSE）。
func (b *Bailian) Stream(ctx context.Context, req CompletionRequest, onChunk func(string) error) (TokenUsage, error) {
	body := buildChatPayload(req, true)
	data, err := json.Marshal(body)
	if err != nil {
		return TokenUsage{}, err
	}
	url := b.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return TokenUsage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return TokenUsage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return TokenUsage{}, fmt.Errorf("bailian stream: %s: %s", resp.Status, string(raw))
	}
	var out strings.Builder
	err = parseSSEStream(resp.Body, func(chunk string) error {
		out.WriteString(chunk)
		return onChunk(chunk)
	})
	if err != nil {
		return TokenUsage{}, err
	}
	return EstimateTokens(req.System+req.User, out.String()), nil
}
