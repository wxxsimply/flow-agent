package volcengine

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// PostJSON POST JSON 到方舟 API。
func (c *Client) PostJSON(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.postJSON(ctx, path, body)
}

// GetJSON GET 方舟 API。
func (c *Client) GetJSON(ctx context.Context, path string) ([]byte, error) {
	return c.getJSON(ctx, path)
}

// FetchBytes 下载 URL 内容。
func FetchBytes(ctx context.Context, url string) ([]byte, error) {
	return fetchURL(ctx, url)
}

// DecodeDataURL 解析 data:image/...;base64,... 为二进制。
func DecodeDataURL(dataURL string) ([]byte, error) {
	const prefix = "data:"
	if !strings.HasPrefix(dataURL, prefix) {
		return nil, fmt.Errorf("invalid data url")
	}
	rest := dataURL[len(prefix):]
	idx := strings.Index(rest, ",")
	if idx < 0 {
		return nil, fmt.Errorf("invalid data url")
	}
	return base64.StdEncoding.DecodeString(rest[idx+1:])
}
