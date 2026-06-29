package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

// DashScope 通义万相文生图（百炼）。
type DashScope struct {
	apiKey  string
	baseURL string
	model   string
	region  config.BailianRegion
	http    *http.Client
}

// NewDashScope 构造万相客户端。
func NewDashScope(p config.Providers, model string) *DashScope {
	region := config.BailianRegion(p.DashScope.Region)
	base := config.BailianNativeBaseURL(region)
	if custom := strings.TrimSpace(p.DashScope.BaseURL); custom != "" {
		// 用户若填了 compatible-mode，剥掉路径后用作原生根地址
		custom = strings.TrimRight(custom, "/")
		custom = strings.Replace(custom, "/compatible-mode/v1", "", 1)
		if strings.Contains(custom, "dashscope") {
			base = custom
		}
	}
	if model == "" {
		model = "wan2.6-t2i"
	}
	return &DashScope{
		apiKey:  p.DashScope.APIKey,
		baseURL: strings.TrimRight(base, "/"),
		model:   model,
		region:  region,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Generate 提交文生图并返回图片二进制。
func (d *DashScope) Generate(ctx context.Context, req GenerateRequest) ([]byte, error) {
	if strings.HasPrefix(d.model, "wan2.6") || strings.HasPrefix(d.model, "wan2.5") {
		return d.generateWithRetry(ctx, func() ([]byte, error) {
			return d.generateWan26Sync(ctx, req)
		})
	}
	return d.generateWithRetry(ctx, func() ([]byte, error) {
		return d.generateLegacyAsync(ctx, req)
	})
}

func (d *DashScope) generateWan26Sync(ctx context.Context, req GenerateRequest) ([]byte, error) {
	size := wanSizeParam(req.Width, req.Height, req.AspectRatio)
	body := map[string]any{
		"model": d.model,
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"role": "user",
					"content": []map[string]string{
						{"text": req.Prompt},
					},
				},
			},
		},
		"parameters": map[string]any{
			"size":          size,
			"n":             1,
			"prompt_extend": false,
			"watermark":     false,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := d.baseURL + "/api/v1/services/aigc/multimodal-generation/generation"
	raw, err := d.postJSON(ctx, url, data, false)
	if err != nil {
		return nil, err
	}
	imgURL, err := parseWan26ImageURL(raw)
	if err != nil {
		return nil, err
	}
	return fetchBytes(ctx, d.http, imgURL)
}

func parseWan26ImageURL(raw []byte) (string, error) {
	var out struct {
		Output struct {
			Choices []struct {
				Message struct {
					Content []struct {
						Image string `json:"image"`
						Type  string `json:"type"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Code != "" {
		return "", fmt.Errorf("dashscope image: %s: %s", out.Code, out.Message)
	}
	for _, ch := range out.Output.Choices {
		for _, c := range ch.Message.Content {
			if c.Image != "" {
				return c.Image, nil
			}
		}
	}
	return "", fmt.Errorf("dashscope image: no image url in response")
}

func (d *DashScope) generateLegacyAsync(ctx context.Context, req GenerateRequest) ([]byte, error) {
	size := wanSizeParam(req.Width, req.Height, req.AspectRatio)
	body := map[string]any{
		"model": d.model,
		"input": map[string]any{
			"prompt": req.Prompt,
		},
		"parameters": map[string]any{
			"size": size,
			"n":    1,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := d.baseURL + "/api/v1/services/aigc/text2image/image-synthesis"
	raw, err := d.postJSON(ctx, url, data, true)
	if err != nil {
		return nil, err
	}
	var out struct {
		Output struct {
			TaskID string `json:"task_id"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Output.TaskID == "" {
		return nil, fmt.Errorf("dashscope image: no task_id")
	}
	return d.pollTask(ctx, out.Output.TaskID)
}

func (d *DashScope) postJSON(ctx context.Context, url string, body []byte, async bool) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	if async {
		httpReq.Header.Set("X-DashScope-Async", "enable")
	}
	resp, err := d.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 429 {
		return nil, &rateLimitError{msg: string(raw)}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dashscope image: %s: %s", resp.Status, string(raw))
	}
	return raw, nil
}

type rateLimitError struct{ msg string }

func (e *rateLimitError) Error() string { return "dashscope rate limit: " + e.msg }

func (d *DashScope) generateWithRetry(ctx context.Context, fn func() ([]byte, error)) ([]byte, error) {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt*attempt) * 2 * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
		data, err := fn()
		if err == nil {
			return data, nil
		}
		last = err
		if !isRateLimit(err) {
			return nil, err
		}
	}
	return nil, last
}

func isRateLimit(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(s, "RateQuota") || strings.Contains(s, "rate limit")
}

func (d *DashScope) pollTask(ctx context.Context, taskID string) ([]byte, error) {
	url := d.baseURL + "/api/v1/tasks/" + taskID
	for i := 0; i < 90; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+d.apiKey)
		resp, err := d.http.Do(req)
		if err != nil {
			return nil, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var out struct {
			Output struct {
				TaskStatus string `json:"task_status"`
				Results    []struct {
					URL string `json:"url"`
				} `json:"results"`
			} `json:"output"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
		switch out.Output.TaskStatus {
		case "SUCCEEDED":
			if len(out.Output.Results) > 0 && out.Output.Results[0].URL != "" {
				return fetchBytes(ctx, d.http, out.Output.Results[0].URL)
			}
			return nil, fmt.Errorf("dashscope image: no result url")
		case "FAILED", "CANCELED":
			return nil, fmt.Errorf("dashscope image task failed: %s", out.Output.TaskStatus)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("dashscope image: task timeout")
}

// wanSizeParam 万相允许的竖屏尺寸（9:16 推荐 960*1696）。
func wanSizeParam(width, height int, aspect string) string {
	if width > 0 && height > 0 {
		pixels := width * height
		if pixels >= 1280*1280 && pixels <= 1440*1440 {
			return fmt.Sprintf("%d*%d", width, height)
		}
	}
	if aspect == "9:16" || (width == 1080 && height == 1920) {
		return "960*1696"
	}
	if aspect == "16:9" || (width == 1920 && height == 1080) {
		return "1696*960"
	}
	return "1280*1280"
}

func fetchBytes(ctx context.Context, c *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch image url: %s: %s", resp.Status, string(b))
	}
	return io.ReadAll(resp.Body)
}
