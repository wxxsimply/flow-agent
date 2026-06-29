package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

const (
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	geminiDefaultModel   = "veo-3.1-lite-generate-preview"
)

// GeminiVeo Google Gemini API Veo 图生/文生视频（异步 predictLongRunning）。
type GeminiVeo struct {
	apiKey      string
	baseURL     string
	model       string
	aspectRatio string
	resolution  string
	fastPoll    bool
	http        *http.Client
	pollHTTP    *http.Client
}

// NewGeminiVeo 构造 Veo 客户端。
func NewGeminiVeo(p config.Providers, model, aspectRatio, resolution string, fastPoll bool) *GeminiVeo {
	key := strings.TrimSpace(p.Gemini.APIKey)
	if key == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(p.Gemini.BaseURL), "/")
	if base == "" {
		base = geminiDefaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = geminiDefaultModel
	}
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = "9:16"
	}
	if strings.TrimSpace(resolution) == "" {
		resolution = "720p"
	}
	return &GeminiVeo{
		apiKey:      key,
		baseURL:     base,
		model:       model,
		aspectRatio: aspectRatio,
		resolution:  normalizeVeoResolution(resolution),
		fastPoll:    fastPoll,
		http:        &http.Client{Timeout: 120 * time.Second},
		pollHTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

// GeminiConfigured 是否已配置 Gemini API Key。
func GeminiConfigured(p config.Providers) bool {
	return strings.TrimSpace(p.Gemini.APIKey) != ""
}

// ImageToVideo 首帧图生视频；可选 LastFramePath 做首尾帧插值。
func (g *GeminiVeo) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	if g == nil {
		return "", fmt.Errorf("gemini veo: not configured (set gemini.api_key)")
	}
	imgInline, err := imageInlineForVeo(req.ImagePath)
	if err != nil {
		return "", err
	}
	out := strings.TrimSuffix(req.ImagePath, filepath.Ext(req.ImagePath)) + ".mp4"
	if strings.TrimSpace(req.ImagePath) == "" {
		return "", fmt.Errorf("gemini veo i2v: image_path required")
	}
	model := g.model
	if m := strings.TrimSpace(req.Model); m != "" {
		model = m
	}
	ratio := g.aspectRatio
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		ratio = ar
	}
	res := g.resolution
	if r := strings.TrimSpace(req.Resolution); r != "" {
		res = normalizeVeoResolution(r)
	}
	var lastInline map[string]any
	if lf := strings.TrimSpace(req.LastFramePath); lf != "" {
		lastInline, err = imageInlineForVeo(lf)
		if err != nil {
			return "", err
		}
	}
	return g.synthesize(ctx, model, imgInline, lastInline, strings.TrimSpace(req.Prompt), req.DurationSec, ratio, res, out)
}

// TextToVideo 文生视频。
func (g *GeminiVeo) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	if g == nil {
		return "", fmt.Errorf("gemini veo: not configured")
	}
	out := strings.TrimSpace(req.OutPath)
	if out == "" {
		return "", fmt.Errorf("gemini veo t2v: out_path required")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("gemini veo t2v: empty prompt")
	}
	ratio := g.aspectRatio
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		ratio = ar
	}
	return g.synthesize(ctx, g.model, nil, nil, prompt, req.DurationSec, ratio, g.resolution, out)
}

func (g *GeminiVeo) synthesize(
	ctx context.Context,
	model string,
	imageInline, lastInline map[string]any,
	prompt string,
	durSec int,
	aspectRatio, resolution, outPath string,
) (string, error) {
	instance := map[string]any{}
	if prompt != "" {
		instance["prompt"] = prompt
	}
	if imageInline != nil {
		instance["image"] = imageInline
	}
	if lastInline != nil {
		instance["lastFrame"] = lastInline
	}
	if len(instance) == 0 {
		return "", fmt.Errorf("gemini veo: prompt or image required")
	}

	params := map[string]any{
		"aspectRatio":      aspectRatio,
		"durationSeconds":  clampVeoDuration(durSec),
		"resolution":       resolution,
		"personGeneration": "allow_adult",
	}
	body := map[string]any{
		"instances":  []map[string]any{instance},
		"parameters": params,
	}

	opName, err := g.createOperation(ctx, model, body)
	if err != nil {
		return "", err
	}
	videoURI, err := g.pollOperation(ctx, opName)
	if err != nil {
		return "", err
	}
	if err := g.downloadVideo(ctx, videoURI, outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func (g *GeminiVeo) createOperation(ctx context.Context, model string, body map[string]any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/models/%s:predictLongRunning", g.baseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gemini veo create: http %d: %s", resp.StatusCode, truncateVeo(string(raw), 500))
	}
	var out struct {
		Name  string `json:"name"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error.Message != "" {
		return "", fmt.Errorf("gemini veo create: %s", out.Error.Message)
	}
	if out.Name == "" {
		return "", fmt.Errorf("gemini veo create: empty operation name")
	}
	return out.Name, nil
}

func (g *GeminiVeo) pollOperation(ctx context.Context, opName string) (string, error) {
	url := g.baseURL + "/" + strings.TrimPrefix(opName, "/")
	deadline := time.Now().Add(15 * time.Minute)
	interval := 10 * time.Second
	if g.fastPoll {
		interval = 6 * time.Second
	}
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("gemini veo: operation timeout %s", opName)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("x-goog-api-key", g.apiKey)
		resp, err := g.pollHTTP.Do(req)
		if err != nil {
			return "", err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("gemini veo poll: http %d: %s", resp.StatusCode, truncateVeo(string(raw), 400))
		}
		var out struct {
			Done  bool `json:"done"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
			Response struct {
				GenerateVideoResponse struct {
					GeneratedSamples []struct {
						Video struct {
							URI string `json:"uri"`
						} `json:"video"`
					} `json:"generatedSamples"`
				} `json:"generateVideoResponse"`
				GeneratedVideos []struct {
					Video struct {
						URI string `json:"uri"`
					} `json:"video"`
				} `json:"generatedVideos"`
			} `json:"response"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		if out.Error.Message != "" {
			return "", fmt.Errorf("gemini veo operation failed: %s", out.Error.Message)
		}
		if out.Done {
			if uri := extractVeoVideoURI(out.Response); uri != "" {
				return uri, nil
			}
			return "", fmt.Errorf("gemini veo: no video uri in completed operation")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}
}

func extractVeoVideoURI(resp struct {
	GenerateVideoResponse struct {
		GeneratedSamples []struct {
			Video struct {
				URI string `json:"uri"`
			} `json:"video"`
		} `json:"generatedSamples"`
	} `json:"generateVideoResponse"`
	GeneratedVideos []struct {
		Video struct {
			URI string `json:"uri"`
		} `json:"video"`
	} `json:"generatedVideos"`
}) string {
	for _, s := range resp.GenerateVideoResponse.GeneratedSamples {
		if s.Video.URI != "" {
			return s.Video.URI
		}
	}
	for _, s := range resp.GeneratedVideos {
		if s.Video.URI != "" {
			return s.Video.URI
		}
	}
	return ""
}

func (g *GeminiVeo) downloadVideo(ctx context.Context, uri, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.pollHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gemini veo download: http %d", resp.StatusCode)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func imageInlineForVeo(path string) (map[string]any, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil, fmt.Errorf("gemini veo: remote image URLs not supported yet: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return map[string]any{
		"inlineData": map[string]any{
			"mimeType": mime,
			"data":     b64,
		},
	}, nil
}

func clampVeoDuration(sec int) string {
	switch {
	case sec <= 4:
		return "4"
	case sec <= 6:
		return "6"
	default:
		return "8"
	}
}

func normalizeVeoResolution(r string) string {
	r = strings.TrimSpace(strings.ToLower(r))
	switch r {
	case "", "720p", "720":
		return "720p"
	case "1080p", "1080":
		return "1080p"
	case "4k":
		return "4k"
	default:
		if strings.HasSuffix(r, "p") {
			return r
		}
		return "720p"
	}
}

func truncateVeo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// VeoDurationSeconds 将 stack 秒数映射为 Veo API 支持的 4/6/8。
func VeoDurationSeconds(sec int) int {
	n, _ := strconv.Atoi(clampVeoDuration(sec))
	return n
}
