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
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

const (
	wanVideoSynthesisPath = "/api/v1/services/aigc/video-generation/video-synthesis"
)

// Wan 百炼万相图生/文生视频（异步任务）。
type Wan struct {
	apiKey      string
	baseURL     string
	model       string
	qualityModel string
	resolution  string
	silentAudio  bool
	fastPoll     bool
	http         *http.Client
	pollHTTP     *http.Client
}

// NewWan 构造万相视频客户端。
func NewWan(p config.Providers, model, qualityModel, resolution string, silentAudio, fastPoll bool) *Wan {
	if strings.TrimSpace(p.DashScope.APIKey) == "" {
		return nil
	}
	region := config.NormalizeBailianRegion(config.BailianRegion(p.DashScope.Region))
	base := config.BailianNativeBaseURL(region)
	if custom := strings.TrimSpace(p.DashScope.BaseURL); custom != "" {
		custom = strings.TrimRight(custom, "/")
		custom = strings.Replace(custom, "/compatible-mode/v1", "", 1)
		if strings.Contains(custom, "dashscope") {
			base = custom
		}
	}
	if strings.TrimSpace(model) == "" {
		model = "wan2.6-i2v-flash"
	}
	if strings.TrimSpace(resolution) == "" {
		resolution = "720P"
	}
	return &Wan{
		apiKey:       p.DashScope.APIKey,
		baseURL:      strings.TrimRight(base, "/"),
		model:        model,
		qualityModel: strings.TrimSpace(qualityModel),
		resolution:   normalizeWanResolution(resolution),
		silentAudio:  silentAudio,
		fastPoll:     fastPoll,
		http:         &http.Client{Timeout: 90 * time.Second},
		pollHTTP:     &http.Client{Timeout: 60 * time.Second},
	}
}

// WanConfigured 是否已配置百炼 Key（万相视频）。
func WanConfigured(p config.Providers) bool {
	return strings.TrimSpace(p.DashScope.APIKey) != ""
}

// ImageToVideo 图生视频；返回本地 mp4 路径。
func (w *Wan) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	if w == nil {
		return "", fmt.Errorf("wan: not configured (set dashscope api_key)")
	}
	imgURL, err := imageInputForWan(req.ImagePath)
	if err != nil {
		return "", err
	}
	out := strings.TrimSuffix(req.ImagePath, filepath.Ext(req.ImagePath)) + ".mp4"
	if strings.TrimSpace(req.ImagePath) == "" {
		return "", fmt.Errorf("wan image2video: image_path required")
	}
	dur := clampWanDuration(req.DurationSec)
	model := w.model
	if m := strings.TrimSpace(req.Model); m != "" {
		model = m
	}
	res := w.resolution
	if r := strings.TrimSpace(req.Resolution); r != "" {
		res = normalizeWanResolution(r)
	}
	return w.synthesizeWithResolution(ctx, model, res, imgURL, strings.TrimSpace(req.Prompt), dur, out)
}

// TextToVideo 文生视频（万相 t2v）；无首帧时使用 prompt 生成。
func (w *Wan) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	if w == nil {
		return "", fmt.Errorf("wan: not configured")
	}
	out := strings.TrimSpace(req.OutPath)
	if out == "" {
		return "", fmt.Errorf("wan text2video: out_path required")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("wan text2video: empty prompt")
	}
	dur := clampWanDuration(req.DurationSec)
	// 文生视频：无 img_url，使用 video-synthesis 仅 prompt（部分模型支持）
	model := strings.TrimPrefix(w.model, "i2v")
	if !strings.Contains(model, "t2v") {
		model = "wan2.6-t2v-flash"
	}
	return w.synthesize(ctx, model, "", prompt, dur, out)
}

func (w *Wan) synthesize(ctx context.Context, model, imgURL, prompt string, dur int, out string) (string, error) {
	return w.synthesizeWithResolution(ctx, model, w.resolution, imgURL, prompt, dur, out)
}

func (w *Wan) synthesizeWithResolution(ctx context.Context, model, resolution, imgURL, prompt string, dur int, out string) (string, error) {
	input := map[string]any{}
	if prompt != "" {
		input["prompt"] = prompt
	}
	if imgURL != "" {
		input["img_url"] = imgURL
	}
	params := map[string]any{
		"resolution":    resolution,
		"duration":      dur,
		"prompt_extend": true,
	}
	if w.silentAudio && strings.Contains(model, "flash") {
		params["audio"] = false
	}
	body := map[string]any{
		"model":      model,
		"input":      input,
		"parameters": params,
	}
	if imgURL == "" && prompt == "" {
		return "", fmt.Errorf("wan: prompt or img_url required")
	}

	taskID, err := w.createTask(ctx, body)
	if err != nil {
		// 质量档回退（仅 i2v）
		if w.qualityModel != "" && w.qualityModel != model && imgURL != "" {
			return w.synthesize(ctx, w.qualityModel, imgURL, prompt, dur, out)
		}
		return "", err
	}
	videoURL, err := w.pollTask(ctx, taskID)
	if err != nil {
		return "", err
	}
	if err := downloadFile(ctx, w.pollHTTP, videoURL, out); err != nil {
		return "", err
	}
	return out, nil
}

func (w *Wan) createTask(ctx context.Context, body map[string]any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := w.baseURL + wanVideoSynthesisPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("X-DashScope-Async", "enable")

	resp, err := w.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("wan create: http %d: %s", resp.StatusCode, truncateWan(string(raw), 500))
	}
	var out struct {
		Output struct {
			TaskID string `json:"task_id"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Code != "" {
		return "", fmt.Errorf("wan create: %s: %s", out.Code, out.Message)
	}
	if out.Output.TaskID == "" {
		return "", fmt.Errorf("wan create: empty task_id")
	}
	return out.Output.TaskID, nil
}

func (w *Wan) pollTask(ctx context.Context, taskID string) (string, error) {
	url := w.baseURL + "/api/v1/tasks/" + taskID
	deadline := time.Now().Add(12 * time.Minute)
	interval := 15 * time.Second
	if w.fastPoll {
		interval = 8 * time.Second
	}
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("wan: task timeout %s", taskID)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+w.apiKey)
		resp, err := w.pollHTTP.Do(req)
		if err != nil {
			return "", err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("wan poll: http %d: %s", resp.StatusCode, truncateWan(string(raw), 400))
		}
		var out struct {
			Output struct {
				TaskStatus string `json:"task_status"`
				VideoURL   string `json:"video_url"`
				Code       string `json:"code"`
				Message    string `json:"message"`
			} `json:"output"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		status := out.Output.TaskStatus
		switch status {
		case "SUCCEEDED":
			if out.Output.VideoURL != "" {
				return out.Output.VideoURL, nil
			}
			return "", fmt.Errorf("wan: no video_url in result")
		case "FAILED", "CANCELED":
			msg := out.Output.Message
			if msg == "" {
				msg = out.Message
			}
			return "", fmt.Errorf("wan task %s: %s", status, msg)
		case "PENDING", "RUNNING", "UNKNOWN":
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(interval):
			}
		default:
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(interval):
			}
		}
	}
}

func imageInputForWan(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".bmp":
		mime = "image/bmp"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

func downloadFile(ctx context.Context, c *http.Client, url, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wan download: http %d", resp.StatusCode)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func clampWanDuration(sec int) int {
	if sec < 2 {
		return 5
	}
	if sec > 10 {
		return 10
	}
	return sec
}

func normalizeWanResolution(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return "720P"
	}
	if strings.EqualFold(r, "720p") {
		return "720P"
	}
	if strings.EqualFold(r, "1080p") {
		return "1080P"
	}
	if strings.EqualFold(r, "480p") {
		return "480P"
	}
	return r
}

func truncateWan(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
