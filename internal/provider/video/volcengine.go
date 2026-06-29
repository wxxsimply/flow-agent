package video

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"
)

// Volcengine 火山方舟 Seedance 图生/文生视频。
type Volcengine struct {
	ark         *ve.Client
	model       string
	aspectRatio string
	silentAudio bool
	fastPoll    bool
}

// NewVolcengine 构造 Seedance 客户端。
func NewVolcengine(p config.Providers, model, aspectRatio string, silentAudio, fastPoll bool) *Volcengine {
	ark := ve.New(p)
	if ark == nil {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = "doubao-seedance-1-5-pro-251215"
	}
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = "9:16"
	}
	return &Volcengine{
		ark:         ark,
		model:       model,
		aspectRatio: aspectRatio,
		silentAudio: silentAudio,
		fastPoll:    fastPoll,
	}
}

// VolcengineConfigured 是否已配置方舟 Key。
func VolcengineConfigured(p config.Providers) bool {
	return ve.Configured(p)
}

// ImageToVideo 首帧图生视频。
func (v *Volcengine) ImageToVideo(ctx context.Context, req ImageToVideoRequest) (string, error) {
	if v == nil || v.ark == nil {
		return "", fmt.Errorf("volcengine video: not configured (set volcengine.api_key)")
	}
	imgURL, err := ve.ImageDataURI(req.ImagePath)
	if err != nil {
		return "", err
	}
	out := strings.TrimSuffix(req.ImagePath, filepath.Ext(req.ImagePath)) + ".mp4"
	model := v.model
	if m := strings.TrimSpace(req.Model); m != "" {
		model = m
	}
	ratio := v.aspectRatio
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		ratio = ar
	}
	return v.synthesize(ctx, model, imgURL, strings.TrimSpace(req.Prompt), req.DurationSec, out, ratio)
}

// TextToVideo 文生视频。
func (v *Volcengine) TextToVideo(ctx context.Context, req TextToVideoRequest) (string, error) {
	if v == nil || v.ark == nil {
		return "", fmt.Errorf("volcengine video: not configured")
	}
	out := strings.TrimSpace(req.OutPath)
	if out == "" {
		return "", fmt.Errorf("volcengine text2video: out_path required")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("volcengine text2video: empty prompt")
	}
	return v.synthesize(ctx, v.model, "", prompt, req.DurationSec, out, strings.TrimSpace(req.AspectRatio))
}

func (v *Volcengine) synthesize(ctx context.Context, model, imgDataURI, prompt string, durSec int, outPath, aspectRatio string) (string, error) {
	models := ve.SeedanceModelFallbacks(model)
	var lastErr error
	for i, m := range models {
		out, err := v.synthesizeModel(ctx, m, imgDataURI, prompt, durSec, outPath, aspectRatio)
		if err == nil {
			if i > 0 {
				slog.Info("seedance model fallback", "from", model, "to", m)
			}
			return out, nil
		}
		lastErr = err
		if !ve.IsModelNotOpen(err) {
			return "", err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("seedance: no models to try")
	}
	return "", lastErr
}

func (v *Volcengine) synthesizeModel(ctx context.Context, model, imgDataURI, prompt string, durSec int, outPath, aspectRatio string) (string, error) {
	dur := ve.ClampSeedanceDuration(model, durSec)
	ratio := v.aspectRatio
	if ar := strings.TrimSpace(aspectRatio); ar != "" {
		ratio = ar
	}
	content := []map[string]any{}
	if prompt != "" {
		content = append(content, map[string]any{
			"type": "text",
			"text": prompt,
		})
	}
	if imgDataURI != "" {
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": imgDataURI,
			},
		})
	}
	body := map[string]any{
		"model":          model,
		"content":        content,
		"ratio":          ve.SeedanceRatio(ratio),
		"duration":       dur,
		"generate_audio": !v.silentAudio,
		"watermark":      false,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	raw, err := v.ark.PostJSON(ctx, "/contents/generations/tasks", data)
	if err != nil {
		return "", err
	}
	taskID, err := parseTaskID(raw)
	if err != nil {
		return "", err
	}
	videoURL, err := v.pollTask(ctx, taskID)
	if err != nil {
		return "", err
	}
	videoBytes, err := ve.FetchBytes(ctx, videoURL)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, videoBytes, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func parseTaskID(raw []byte) (string, error) {
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
		return "", fmt.Errorf("seedance create: %s", out.Error.Message)
	}
	if out.ID == "" {
		return "", fmt.Errorf("seedance: empty task id")
	}
	return out.ID, nil
}

func (v *Volcengine) pollTask(ctx context.Context, taskID string) (string, error) {
	interval := 10 * time.Second
	if v.fastPoll {
		interval = 8 * time.Second
	}
	deadline := time.Now().Add(20 * time.Minute)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("seedance: task timeout %s", taskID)
		}
		raw, err := v.ark.GetJSON(ctx, "/contents/generations/tasks/"+taskID)
		if err != nil {
			return "", err
		}
		status, videoURL, failMsg, err := parseTaskResult(raw)
		if err != nil {
			return "", err
		}
		switch strings.ToLower(status) {
		case "succeeded":
			if videoURL == "" {
				return "", fmt.Errorf("seedance: succeeded but no video_url")
			}
			return videoURL, nil
		case "failed", "expired", "cancelled":
			if failMsg == "" {
				failMsg = status
			}
			return "", fmt.Errorf("seedance task failed: %s", failMsg)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}
}

func parseTaskResult(raw []byte) (status, videoURL, failMsg string, err error) {
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Content struct {
			VideoURL string `json:"video_url"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", "", out.Error.Message, fmt.Errorf("seedance poll: %s", out.Error.Message)
	}
	return out.Status, out.Content.VideoURL, out.Message, nil
}
