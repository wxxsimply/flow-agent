package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
)

// DeepSeek 调用 OpenAI 兼容的 /v1/chat/completions。
type DeepSeek struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewDeepSeek 从配置构造客户端。
func NewDeepSeek(cfg config.DeepSeekConfig) *DeepSeek {
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.deepseek.com"
	}
	return &DeepSeek{apiKey: cfg.APIKey, baseURL: strings.TrimRight(base, "/"), http: NewHTTPClient(true)}
}

// Complete 非流式对话补全。
func (d *DeepSeek) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	body := buildChatPayload(req, false)
	data, err := json.Marshal(body)
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.http.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return CompletionResult{}, fmt.Errorf("deepseek: %s: %s", resp.Status, string(raw))
	}
	return decodeChatResponse("deepseek", raw, resp.StatusCode, resp.Status, req)
}

// Stream 使用 SSE 流式补全，按 chunk 回调。
func (d *DeepSeek) Stream(ctx context.Context, req CompletionRequest, onChunk func(string) error) (TokenUsage, error) {
	body := buildChatPayload(req, true)
	data, err := json.Marshal(body)
	if err != nil {
		return TokenUsage{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return TokenUsage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := d.http.Do(httpReq)
	if err != nil {
		return TokenUsage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return TokenUsage{}, fmt.Errorf("deepseek stream: %s: %s", resp.Status, string(raw))
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

type chatPayload struct {
	Model          string           `json:"model"`
	Messages       []chatMessage    `json:"messages"`
	MaxTokens      int              `json:"max_tokens,omitempty"`
	Temperature    float64          `json:"temperature,omitempty"`
	Stream         bool             `json:"stream,omitempty"`
	ResponseFormat *responseFormat  `json:"response_format,omitempty"`
	Thinking       *thinkingConfig  `json:"thinking,omitempty"`
}

type thinkingConfig struct {
	Type string `json:"type"` // enabled | disabled
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
		Delta   chatMessage `json:"delta"`
	} `json:"choices"`
	Usage TokenUsage `json:"usage"`
}

// buildChatPayload 组装请求体，默认模型 deepseek-v4-flash。
func buildChatPayload(req CompletionRequest, stream bool) chatPayload {
	model := req.Model
	if model == "" {
		model = "deepseek-v4-flash"
	}
	msgs := []chatMessage{}
	if req.System != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: req.System})
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: req.User})
	out := chatPayload{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      stream,
		ResponseFormat: func() *responseFormat {
			if req.JSONMode && !stream {
				return &responseFormat{Type: "json_object"}
			}
			return nil
		}(),
	}
	if req.JSONMode && !stream && strings.Contains(strings.ToLower(model), "deepseek-v4") {
		out.Thinking = &thinkingConfig{Type: "disabled"}
	}
	return out
}

// parseSSEStream 解析 OpenAI 兼容 SSE 流。
func parseSSEStream(r io.Reader, onChunk func(string) error) error {
	scanner := bufio.NewScanner(r)
	// 允许较大的 SSE 行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		chunk, err := extractDeltaContent(data)
		if err != nil {
			return err
		}
		if chunk == "" {
			continue
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func extractDeltaContent(data string) (string, error) {
	var out chatResponse
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return "", fmt.Errorf("sse json: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	c := out.Choices[0]
	if c.Delta.Content != "" {
		return c.Delta.Content, nil
	}
	return c.Message.Content, nil
}
