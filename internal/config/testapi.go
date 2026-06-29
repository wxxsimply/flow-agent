package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BailianTestResult 单次端点探测结果。
type BailianTestResult struct {
	Region     string
	BaseURL    string
	HTTPStatus int
	OK         bool
	Message    string
}

// TestBailianChat 用最小请求测试百炼 OpenAI 兼容 chat 接口。
func TestBailianChat(ctx context.Context, apiKey string, region BailianRegion) BailianTestResult {
	base := BailianCompatibleBaseURL(region)
	url := base + "/chat/completions"
	res := BailianTestResult{Region: string(normalizeRegion(region)), BaseURL: base}

	if strings.TrimSpace(apiKey) == "" {
		res.Message = "api_key empty"
		return res
	}

	body, err := json.Marshal(map[string]any{
		"model": "qwen-plus",
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 5,
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		res.Message = err.Error()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	res.HTTPStatus = resp.StatusCode

	if resp.StatusCode == http.StatusOK {
		res.OK = true
		res.Message = "ok"
		return res
	}
	// 401/403 说明网络通但 Key 或地域不对
	if len(raw) > 300 {
		raw = raw[:300]
	}
	res.Message = strings.TrimSpace(string(raw))
	return res
}

// FormatBailianTestReport 格式化 test-api 输出。
func FormatBailianTestReport(results []BailianTestResult, recommended string) string {
	var b strings.Builder
	b.WriteString("百炼 OpenAI 兼容接口探测（POST .../chat/completions）\n\n")
	for _, r := range results {
		mark := "[ ]"
		if r.OK {
			mark = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %-12s %s\n    HTTP %d  %s\n\n", mark, r.Region, r.BaseURL, r.HTTPStatus, r.Message))
	}
	if recommended != "" {
		b.WriteString(fmt.Sprintf("建议配置:\n  dashscope:\n    region: %s\n    base_url: \"\"   # 留空自动选端点\n\n", recommended))
	}
	return b.String()
}
