package llm

import (
	"net/http"
	"time"
)

const (
	// HTTPTimeoutDefault 常规分镜/剧本调用。
	HTTPTimeoutDefault = 3 * time.Minute
	// HTTPTimeoutLong 长文 JSON 生成（如镜头语言扩写），需覆盖服务端排队+生成时间。
	HTTPTimeoutLong = 12 * time.Minute
)

// NewHTTPClient 构造带总超时的 HTTP 客户端（含读响应体）。
func NewHTTPClient(longRunning bool) *http.Client {
	t := HTTPTimeoutDefault
	if longRunning {
		t = HTTPTimeoutLong
	}
	return &http.Client{Timeout: t}
}
