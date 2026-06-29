package image

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"
)

const defaultOpenAIImageModel = "dall-e-3"

// OpenAIMedia OpenAI 兼容 /images/generations（Sora 代理等同源出图）。
type OpenAIMedia struct {
	model string
	ark   *ve.Client
}

// NewOpenAIMedia 构造 OpenAI 兼容文生图客户端。
func NewOpenAIMedia(apiKey, baseURL, model string) *OpenAIMedia {
	ark := ve.NewWithBaseURL(apiKey, baseURL)
	if ark == nil {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = defaultOpenAIImageModel
	}
	return &OpenAIMedia{model: model, ark: ark}
}

// Generate 文生图。
func (o *OpenAIMedia) Generate(ctx context.Context, req GenerateRequest) ([]byte, error) {
	if o == nil || o.ark == nil {
		return nil, fmt.Errorf("openai media image: not configured")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("openai media image: empty prompt")
	}
	body := map[string]any{
		"model":           o.model,
		"prompt":          prompt,
		"size":            ve.SeedreamSize(req.Width, req.Height, req.AspectRatio),
		"response_format": "url",
	}
	if strings.HasPrefix(strings.ToLower(o.model), "dall-e") {
		body["n"] = 1
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	raw, err := o.ark.PostJSON(ctx, "/images/generations", data)
	if err != nil {
		return nil, err
	}
	url, err := parseOpenAIImageURL(raw)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(url, "data:") {
		return ve.DecodeDataURL(url)
	}
	return ve.FetchBytes(ctx, url)
}

func parseOpenAIImageURL(raw []byte) (string, error) {
	var out struct {
		Data []struct {
			URL string `json:"url"`
			B64 string `json:"b64_json"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", fmt.Errorf("openai image: %s", out.Error.Message)
	}
	if len(out.Data) == 0 {
		return "", fmt.Errorf("openai image: empty response")
	}
	if out.Data[0].URL != "" {
		return out.Data[0].URL, nil
	}
	if out.Data[0].B64 != "" {
		return "data:image/png;base64," + out.Data[0].B64, nil
	}
	return "", fmt.Errorf("openai image: no url in response")
}
