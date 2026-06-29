package image

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"
)

// Volcengine 火山方舟 Seedream 文生图。
type Volcengine struct {
	ark   *ve.Client
	model string
}

// NewVolcengine 构造 Seedream 客户端。
func NewVolcengine(p config.Providers, model string) *Volcengine {
	ark := ve.New(p)
	if ark == nil {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = "doubao-seedream-4-0-250828"
	}
	return &Volcengine{ark: ark, model: model}
}

// Generate 文生图，返回 PNG/JPEG 二进制。
func (v *Volcengine) Generate(ctx context.Context, req GenerateRequest) ([]byte, error) {
	if v == nil || v.ark == nil {
		return nil, fmt.Errorf("volcengine image: not configured (set volcengine.api_key)")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("volcengine image: empty prompt")
	}
	body := map[string]any{
		"model":                           v.model,
		"prompt":                          prompt,
		"size":                            ve.SeedreamSize(req.Width, req.Height, req.AspectRatio),
		"response_format":                 "url",
		"watermark":                       false,
		"sequential_image_generation":     "disabled",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	raw, err := v.ark.PostJSON(ctx, "/images/generations", data)
	if err != nil {
		return nil, err
	}
	url, err := parseSeedreamImageURL(raw)
	if err != nil {
		return nil, err
	}
	return ve.FetchBytes(ctx, url)
}

func parseSeedreamImageURL(raw []byte) (string, error) {
	var out struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", fmt.Errorf("seedream: %s", out.Error.Message)
	}
	if len(out.Data) == 0 || out.Data[0].URL == "" {
		return "", fmt.Errorf("seedream: no image url in response")
	}
	return out.Data[0].URL, nil
}
