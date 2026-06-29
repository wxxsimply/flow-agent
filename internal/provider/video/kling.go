package video

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/flow-agent/flow-agent/internal/config"
)

const (
	defaultKlingBaseURL = "https://api-beijing.klingai.com"
	globalKlingBaseURL  = "https://api.klingai.com"
	klingImage2VideoCreate = "/v1/videos/image2video"
	klingImage2VideoPoll   = "/v1/videos/image2video/%s"
	klingText2VideoCreate  = "/v1/videos/text2video"
	klingText2VideoPoll    = "/v1/videos/text2video/%s"
)

// Kling 可灵视频（JWT：Access Key + Secret Key）。
type Kling struct {
	accessKey  string
	secretKey  string
	bases      []string
	imageModel string
	textModel  string
	http       *http.Client
	pollHTTP   *http.Client
}

// NewKling 构造可灵客户端；imageModel / textModel 可为 stack 别名（文生与图生映射不同）。
func NewKling(p config.Providers, imageModel, textModel string) *Kling {
	ak := strings.TrimSpace(p.Kling.AccessKey)
	sk := strings.TrimSpace(p.Kling.SecretKey)
	if ak == "" || sk == "" {
		return nil
	}
	if strings.TrimSpace(textModel) == "" {
		textModel = imageModel
	}
	if strings.TrimSpace(imageModel) == "" {
		imageModel = textModel
	}
	bases := klingBaseURLs(p.Kling.BaseURL)
	return &Kling{
		accessKey:  ak,
		secretKey:  sk,
		bases:      bases,
		imageModel: NormalizeKlingImage2VideoModel(imageModel),
		textModel:  NormalizeKlingText2VideoModel(textModel),
		http:       &http.Client{Timeout: 60 * time.Second},
		pollHTTP:   &http.Client{Timeout: 30 * time.Second},
	}
}

func klingBaseURLs(configured string) []string {
	custom := strings.TrimRight(strings.TrimSpace(configured), "/")
	seen := make(map[string]bool)
	var bases []string
	add := func(u string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		bases = append(bases, u)
	}
	if custom != "" {
		add(custom)
	}
	add(defaultKlingBaseURL)
	add(globalKlingBaseURL)
	return bases
}

// ImageToVideo 图生视频；返回本地 mp4 路径。
func (k *Kling) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	if k == nil {
		return "", fmt.Errorf("kling: not configured (set access_key and secret_key)")
	}
	imgB64, err := imageToBase64(req.ImagePath)
	if err != nil {
		return "", err
	}
	body := map[string]any{
		"model_name": k.imageModel,
		"image":      imgB64,
		"duration":   klingDurationStr(req.DurationSec),
		"mode":       orDefault(req.Mode, "std"),
	}
	if p := strings.TrimSpace(req.Prompt); p != "" {
		body["prompt"] = p
	}
	out := strings.TrimSuffix(req.ImagePath, filepath.Ext(req.ImagePath)) + ".mp4"
	return k.submitAndDownload(ctx, klingImage2VideoCreate, klingImage2VideoPoll, body, out)
}

// TextToVideo 文生视频；返回本地 mp4 路径。
func (k *Kling) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	if k == nil {
		return "", fmt.Errorf("kling: not configured (set access_key and secret_key)")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("kling text2video: empty prompt")
	}
	out := strings.TrimSpace(req.OutPath)
	if out == "" {
		return "", fmt.Errorf("kling text2video: out_path required")
	}
	duration := klingDurationStr(req.DurationSec)
	mode := orDefault(req.Mode, "std")
	aspect := orDefault(req.AspectRatio, "9:16")

	var lastErr error
	for _, model := range Text2VideoModelsToTry(k.textModel) {
		body := map[string]any{
			"model_name":   model,
			"prompt":       prompt,
			"duration":     duration,
			"mode":         mode,
			"aspect_ratio": aspect,
		}
		slog.Debug("kling text2video", "model_name", model, "duration", duration)
		path, err := k.submitAndDownload(ctx, klingText2VideoCreate, klingText2VideoPoll, body, out)
		if err == nil {
			if model != k.textModel {
				slog.Info("kling text2video model fallback", "from", k.textModel, "to", model)
			}
			return path, nil
		}
		lastErr = err
		if !isKlingModelError(err) {
			return "", err
		}
		slog.Warn("kling text2video model rejected", "model_name", model, "err", err)
	}
	if lastErr != nil {
		return "", fmt.Errorf("kling text2video: no supported model_name (tried %v): %w", Text2VideoModelsToTry(k.textModel), lastErr)
	}
	return "", fmt.Errorf("kling text2video: no supported model_name")
}

func (k *Kling) submitAndDownload(ctx context.Context, createPath, pollFmt string, body map[string]any, out string) (string, error) {
	taskID, base, err := k.createTask(ctx, createPath, body)
	if err != nil {
		return "", err
	}
	videoURL, err := k.waitTask(ctx, base, pollFmt, taskID)
	if err != nil {
		return "", err
	}
	if err := k.download(ctx, videoURL, out); err != nil {
		return "", err
	}
	return out, nil
}

func (k *Kling) createTask(ctx context.Context, createPath string, body map[string]any) (taskID, base string, err error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}
	var lastErr error
	for _, b := range k.bases {
		taskID, err = k.createTaskAt(ctx, b, createPath, data)
		if err == nil {
			return taskID, b, nil
		}
		lastErr = err
		if !isKlingAuthError(err) {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("kling: all endpoints failed auth: %w", lastErr)
}

func (k *Kling) createTaskAt(ctx context.Context, base, createPath string, data []byte) (string, error) {
	url := base + createPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	k.setAuth(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := k.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kling create (%s): http %d: %s", base, resp.StatusCode, truncate(string(raw), 400))
	}
	var result klingAPIResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("kling create (%s): code=%d msg=%s", base, result.Code, result.Message)
	}
	if result.Data.TaskID == "" {
		return "", fmt.Errorf("kling create: empty task_id")
	}
	return result.Data.TaskID, nil
}

func (k *Kling) waitTask(ctx context.Context, base, pollFmt, taskID string) (string, error) {
	url := base + fmt.Sprintf(pollFmt, taskID)
	deadline := time.Now().Add(10 * time.Minute)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("kling: task timeout %s", taskID)
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		k.setAuth(httpReq)
		resp, err := k.pollHTTP.Do(httpReq)
		if err != nil {
			return "", err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("kling poll: http %d: %s", resp.StatusCode, truncate(string(raw), 400))
		}
		var result klingAPIResponse
		if err := json.Unmarshal(raw, &result); err != nil {
			return "", err
		}
		if result.Code != 0 {
			return "", fmt.Errorf("kling poll: code=%d msg=%s", result.Code, result.Message)
		}
		switch result.Data.TaskStatus {
		case "succeed":
			videos := result.Data.TaskResult.Videos
			if len(videos) == 0 || videos[0].URL == "" {
				return "", fmt.Errorf("kling: no video url in result")
			}
			return videos[0].URL, nil
		case "failed":
			return "", fmt.Errorf("kling task failed: %s", result.Data.TaskStatusMsg)
		case "submitted", "processing":
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(5 * time.Second):
			}
		default:
			return "", fmt.Errorf("kling: unknown status %q", result.Data.TaskStatus)
		}
	}
}

func (k *Kling) download(ctx context.Context, videoURL, out string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return err
	}
	resp, err := k.pollHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kling download: http %d", resp.StatusCode)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func (k *Kling) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+k.jwtToken())
}

func (k *Kling) jwtToken() string {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": k.accessKey,
		"exp": now + 1800,
		"nbf": now - 5,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(k.secretKey))
	if err != nil {
		return ""
	}
	return s
}

func isKlingAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "401") ||
		strings.Contains(s, "1000") ||
		strings.Contains(s, "1001") ||
		strings.Contains(s, "1002") ||
		strings.Contains(s, "access key not found") ||
		strings.Contains(s, "身份验证")
}

func isKlingModelError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "1201") ||
		strings.Contains(s, "model is not supported") ||
		strings.Contains(s, "model_name") && strings.Contains(s, "invalid") ||
		strings.Contains(s, "invalid model")
}

type klingAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID        string `json:"task_id"`
		TaskStatus    string `json:"task_status"`
		TaskStatusMsg string `json:"task_status_msg"`
		TaskResult    struct {
			Videos []struct {
				URL string `json:"url"`
			} `json:"videos"`
		} `json:"task_result"`
	} `json:"data"`
}

func imageToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func klingDurationStr(sec int) string {
	if sec >= 8 {
		return "10"
	}
	return "5"
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// KlingConfigured 是否已配置可灵 AK/SK。
func KlingConfigured(p config.Providers) bool {
	return strings.TrimSpace(p.Kling.AccessKey) != "" && strings.TrimSpace(p.Kling.SecretKey) != ""
}
