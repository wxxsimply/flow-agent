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

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/config"
)

const (
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	soraDefaultModel     = "sora-2"
)

// Sora OpenAI Videos API（Sora 2 / Sora 2 Pro）图生/文生视频。
type Sora struct {
	apiKey      string
	baseURL     string
	model       string
	aspectRatio string
	size        string
	fastPoll    bool
	http        *http.Client
	pollHTTP    *http.Client
}

// NewSora 构造 Sora 客户端。
func NewSora(p config.Providers, model, aspectRatio, resolution string, fastPoll bool) *Sora {
	key := strings.TrimSpace(p.OpenAI.APIKey)
	if key == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(p.OpenAI.BaseURL), "/")
	if base == "" {
		base = openAIDefaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = soraDefaultModel
	}
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = "9:16"
	}
	size := normalizeSoraSize(resolution, aspectRatio, model)
	return &Sora{
		apiKey:      key,
		baseURL:     base,
		model:       model,
		aspectRatio: aspectRatio,
		size:        size,
		fastPoll:    fastPoll,
		http:        &http.Client{Timeout: 180 * time.Second},
		pollHTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

// SoraConfigured 是否已配置 OpenAI API Key（Videos API）。
func SoraConfigured(p config.Providers) bool {
	return strings.TrimSpace(p.OpenAI.APIKey) != ""
}

// ImageToVideo 首帧图生视频（input_reference）。
func (s *Sora) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	if s == nil {
		return "", fmt.Errorf("sora: not configured (set openai.api_key)")
	}
	if strings.TrimSpace(req.ImagePath) == "" {
		return "", fmt.Errorf("sora i2v: image_path required")
	}
	out := strings.TrimSuffix(req.ImagePath, filepath.Ext(req.ImagePath)) + ".mp4"
	model := s.model
	if m := strings.TrimSpace(req.Model); m != "" {
		model = m
	}
	size := s.size
	if sz := soraSizeFromRequest(req.Resolution, req.AspectRatio, model); sz != "" {
		size = sz
	} else if ar := strings.TrimSpace(req.AspectRatio); ar != "" && ar != s.aspectRatio {
		size = normalizeSoraSize(req.Resolution, ar, model)
	}
	prepared, cleanup, err := prepareSoraInputImage(req.ImagePath, size)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}
	seconds := clampSoraDuration(req.DurationSec)
	return s.render(ctx, model, size, seconds, strings.TrimSpace(req.Prompt), prepared, out)
}

// TextToVideo 文生视频。
func (s *Sora) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	if s == nil {
		return "", fmt.Errorf("sora: not configured")
	}
	out := strings.TrimSpace(req.OutPath)
	if out == "" {
		return "", fmt.Errorf("sora t2v: out_path required")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("sora t2v: empty prompt")
	}
	model := s.model
	size := s.size
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		size = normalizeSoraSize("", ar, model)
	}
	return s.render(ctx, model, size, clampSoraDuration(req.DurationSec), prompt, "", out)
}

func (s *Sora) render(ctx context.Context, model, size, seconds, prompt, imagePath, outPath string) (string, error) {
	videoID, err := s.createJob(ctx, model, size, seconds, prompt, imagePath)
	if err != nil {
		return "", err
	}
	if err := s.pollJob(ctx, videoID); err != nil {
		return "", err
	}
	if err := s.downloadContent(ctx, videoID, outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

type soraInputReference struct {
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

type soraCreateRequest struct {
	Prompt         string              `json:"prompt"`
	Model          string              `json:"model"`
	Size           string              `json:"size"`
	Seconds        string              `json:"seconds"`
	InputReference *soraInputReference `json:"input_reference,omitempty"`
}

func (s *Sora) createJob(ctx context.Context, model, size, seconds, prompt, imagePath string) (string, error) {
	body := soraCreateRequest{
		Prompt:  prompt,
		Model:   model,
		Size:    size,
		Seconds: seconds,
	}
	if imagePath != "" {
		dataURL, err := soraImageDataURL(imagePath)
		if err != nil {
			return "", err
		}
		body.InputReference = &soraInputReference{ImageURL: dataURL}
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/videos", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("sora create: http %d: %s", resp.StatusCode, truncateSora(string(raw), 500))
	}
	var out struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", fmt.Errorf("sora create: %s", out.Error.Message)
	}
	if out.ID == "" {
		return "", fmt.Errorf("sora create: empty video id")
	}
	return out.ID, nil
}

func (s *Sora) pollJob(ctx context.Context, videoID string) error {
	url := s.baseURL + "/videos/" + videoID
	deadline := time.Now().Add(20 * time.Minute)
	interval := 12 * time.Second
	if s.fastPoll {
		interval = 8 * time.Second
	}
	pollNetRetries := 0
	const maxPollNetRetries = 6
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("sora: job timeout %s", videoID)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
		resp, err := s.pollHTTP.Do(req)
		if err != nil {
			if pollNetRetries < maxPollNetRetries && soraRetryableNetErr(err) {
				pollNetRetries++
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(interval):
				}
				continue
			}
			return err
		}
		pollNetRetries = 0
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("sora poll: http %d: %s", resp.StatusCode, truncateSora(string(raw), 400))
		}
		var out struct {
			Status string `json:"status"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return err
		}
		switch out.Status {
		case "completed":
			return nil
		case "failed":
			msg := "video generation failed"
			if out.Error != nil && out.Error.Message != "" {
				msg = out.Error.Message
			}
			return fmt.Errorf("sora job %s: %s", videoID, msg)
		case "queued", "in_progress", "":
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}
		default:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}
		}
	}
}

func (s *Sora) downloadContent(ctx context.Context, videoID, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	url := s.baseURL + "/videos/" + videoID + "/content"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.pollHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sora download: http %d", resp.StatusCode)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func prepareSoraInputImage(src, size string) (path string, cleanup func(), err error) {
	w, h, ok := parseSoraSize(size)
	if !ok {
		return src, nil, nil
	}
	if ffmpeg.ImageMatchesSize(src, w, h) {
		return src, nil, nil
	}
	if !ffmpeg.Available() {
		return src, nil, nil
	}
	tmp := src + ".sora-" + size + filepath.Ext(src)
	if err := ffmpeg.ScaleImageToSize(src, tmp, w, h); err != nil {
		return src, nil, nil
	}
	return tmp, func() { _ = os.Remove(tmp) }, nil
}

func soraSizeFromRequest(resolution, aspectRatio, model string) string {
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return ""
	}
	if strings.Contains(resolution, "x") {
		return strings.ToLower(resolution)
	}
	return normalizeSoraSize(resolution, aspectRatio, model)
}

func normalizeSoraSize(resolution, aspectRatio, model string) string {
	if sz := strings.TrimSpace(resolution); strings.Contains(sz, "x") {
		return strings.ToLower(sz)
	}
	ratio := strings.TrimSpace(aspectRatio)
	pro := strings.Contains(strings.ToLower(model), "pro")
	switch ratio {
	case "9:16", "9/16":
		if pro {
			return "1080x1920"
		}
		return "720x1280"
	case "16:9", "16/9":
		if pro {
			return "1920x1080"
		}
		return "1280x720"
	default:
		if pro {
			return "1080x1920"
		}
		return "720x1280"
	}
}

func parseSoraSize(size string) (w, h int, ok bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

func clampSoraDuration(sec int) string {
	switch {
	case sec <= 4:
		return "4"
	case sec <= 8:
		return "8"
	default:
		return "12"
	}
}

// SoraDurationSeconds 将 stack 秒数映射为 Sora API 支持的 4/8/12。
func SoraDurationSeconds(sec int) int {
	n, _ := strconv.Atoi(clampSoraDuration(sec))
	return n
}

func soraImageDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("sora input image: %w", err)
	}
	mime := soraImageMIME(path)
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + mime + ";base64," + encoded, nil
}

func soraImageMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func truncateSora(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func soraRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "tls:")
}
