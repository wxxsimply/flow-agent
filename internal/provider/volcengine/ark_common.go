package volcengine

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

// Client 火山方舟 HTTP 客户端（Seedream / Seedance 共用）。
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New 构造方舟客户端。
func New(p config.Providers) *Client {
	key := p.ArkAPIKey()
	if key == "" {
		return nil
	}
	return NewWithBaseURL(key, p.Volcengine.ArkBaseURL())
}

// NewWithBaseURL 使用显式 Key 与 Base URL 构造 HTTP 客户端（OpenAI 兼容网关复用）。
func NewWithBaseURL(apiKey, baseURL string) *Client {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &Client{
		apiKey:  key,
		baseURL: base,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Configured 是否已配置方舟 Key。
func Configured(p config.Providers) bool {
	return p.VolcengineArkConfigured()
}

func (c *Client) postJSON(ctx context.Context, path string, body []byte) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("volcengine ark: not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("volcengine ark: %s: %s", resp.Status, truncate(string(raw), 500))
	}
	return raw, nil
}

func (c *Client) getJSON(ctx context.Context, path string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("volcengine ark: not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("volcengine ark: %s: %s", resp.Status, truncate(string(raw), 500))
	}
	return raw, nil
}

// ImageDataURI 本地 PNG/JPEG → data URI，供 Seedance 图生视频。
func ImageDataURI(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty image path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "image/png"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, b64), nil
}

// SeedreamSize 将宽高/比例映射为 Seedream size 参数。
func SeedreamSize(width, height int, aspect string) string {
	if width > 0 && height > 0 {
		return fmt.Sprintf("%dx%d", width, height)
	}
	switch aspect {
	case "16:9":
		return "2560x1440"
	case "1:1":
		return "2048x2048"
	default:
		return "1440x2560" // 9:16 竖屏
	}
}

// SeedanceRatio 视频画幅。
func SeedanceRatio(aspect string) string {
	switch strings.TrimSpace(aspect) {
	case "16:9":
		return "16:9"
	case "1:1":
		return "1:1"
	default:
		return "9:16"
	}
}

// SeedanceModelFallbacks 按优先级返回可尝试的 Seedance 模型（ModelNotOpen 时依次回退）。
func SeedanceModelFallbacks(primary string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(m string) {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			return
		}
		seen[m] = true
		out = append(out, m)
	}
	add(primary)
	add("doubao-seedance-1-5-pro-251215")
	add("doubao-seedance-1-0-pro-250528")
	add("doubao-seedance-1-0-pro-fast-251015")
	add("doubao-seedance-2-0-fast-260128")
	return out
}

// IsModelNotOpen 方舟返回模型未开通。
func IsModelNotOpen(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "modelnotopen") ||
		strings.Contains(msg, "has not activated the model") ||
		strings.Contains(msg, "model is not open")
}

// IsVolcengineFatal 方舟/Seedream/Seedance 不可重试的错误（欠费、鉴权、配额等）。
func IsVolcengineFatal(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "accountoverdueerror") ||
		strings.Contains(msg, "overdue balance") ||
		strings.Contains(msg, "insufficient balance") ||
		strings.Contains(msg, "account frozen") ||
		strings.Contains(msg, "payment required") ||
		strings.Contains(msg, "authenticate request") ||
		strings.Contains(msg, "access denied") {
		return true
	}
	if strings.Contains(msg, "403 forbidden") &&
		(strings.Contains(msg, "overdue") || strings.Contains(msg, "forbidden\":")) {
		return true
	}
	return false
}

// IsVolcengineRetryable i2v/出图是否值得重试。
func IsVolcengineRetryable(err error) bool {
	if err == nil {
		return false
	}
	if IsVolcengineFatal(err) || IsModelNotOpen(err) {
		return false
	}
	return true
}

// ClampSeedanceDuration Seedance 时长：2.0 标准 4–15s，2.0 fast / 1.x 4–12s。
func ClampSeedanceDuration(model string, sec int) int {
	max := 12
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(m, "seedance-2-0") && !strings.Contains(m, "fast") {
		max = 15
	}
	if sec < 4 {
		return 4
	}
	if sec > max {
		return max
	}
	return sec
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch %s: %s", resp.Status, truncate(string(b), 200))
	}
	return io.ReadAll(resp.Body)
}
