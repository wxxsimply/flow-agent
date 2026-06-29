package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// KlingText2VideoModelProbe 探测某 model_name 是否可被 text2video 接受（提交成功即视为可用）。
type KlingText2VideoModelProbe struct {
	Model   string
	OK      bool
	Message string
}

// text2video 候选 model_name（按可灵开放平台常见顺序；账号开通情况不同）。
var klingText2VideoCandidates = []string{
	"kling-v1-6",
	"kling-v1-5",
	"kling-v2-6",
	"kling-v2-1-master",
	"kling-v2-master",
	"kling-v3-0",
	"kling-v2-5",
	"kling-v2-5-turbo",
	"kling-v2-1",
}

// ProbeKlingText2VideoModels 在指定 base 上依次尝试文生视频 model_name。
func ProbeKlingText2VideoModels(ctx context.Context, p Providers, base string) []KlingText2VideoModelProbe {
	ak := strings.TrimSpace(p.Kling.AccessKey)
	sk := strings.TrimSpace(p.Kling.SecretKey)
	if ak == "" || sk == "" {
		return nil
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		for _, r := range ProbeKling(ctx, p) {
			if r.OK {
				base = r.BaseURL
				break
			}
		}
	}
	if base == "" {
		base = "https://api.klingai.com"
	}
	token, err := signKlingJWT(ak, sk)
	if err != nil {
		return []KlingText2VideoModelProbe{{Model: "(jwt)", Message: err.Error()}}
	}
	client := &http.Client{Timeout: 25 * time.Second}
	var out []KlingText2VideoModelProbe
	for _, model := range klingText2VideoCandidates {
		if ctx.Err() != nil {
			break
		}
		r := probeText2VideoModel(ctx, client, base, token, model)
		out = append(out, r)
		if r.OK {
			break
		}
	}
	return out
}

func probeText2VideoModel(ctx context.Context, client *http.Client, base, token, model string) KlingText2VideoModelProbe {
	r := KlingText2VideoModelProbe{Model: model}
	body, err := json.Marshal(map[string]any{
		"model_name":   model,
		"prompt":       "flowagent model probe, vertical 9:16, short clip",
		"duration":     "5",
		"mode":         "std",
		"aspect_ratio": "9:16",
	})
	if err != nil {
		r.Message = err.Error()
		return r
	}
	url := base + "/v1/videos/text2video"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		r.Message = err.Error()
		return r
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		r.Message = err.Error()
		return r
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if isKlingAuthHTTP(resp.StatusCode, string(raw)) {
		r.Message = "auth failed"
		return r
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if envelope.Code == 0 && envelope.Data.TaskID != "" {
		r.OK = true
		r.Message = "task_id=" + envelope.Data.TaskID
		return r
	}
	if envelope.Message != "" {
		r.Message = fmt.Sprintf("code=%d %s", envelope.Code, envelope.Message)
	} else {
		r.Message = fmt.Sprintf("HTTP %d %s", resp.StatusCode, truncateKlingBody(string(raw)))
	}
	return r
}

// FirstWorkingKlingText2VideoModel 返回第一个探测成功的 model_name，无则空字符串。
func FirstWorkingKlingText2VideoModel(ctx context.Context, p Providers) (model, base string) {
	var baseURL string
	for _, r := range ProbeKling(ctx, p) {
		if r.OK {
			baseURL = r.BaseURL
			break
		}
	}
	for _, r := range ProbeKlingText2VideoModels(ctx, p, baseURL) {
		if r.OK {
			return r.Model, baseURL
		}
	}
	return "", baseURL
}

// FormatKlingText2VideoProbeReport 人类可读报告。
func FormatKlingText2VideoProbeReport(results []KlingText2VideoModelProbe) string {
	var b strings.Builder
	b.WriteString("Kling text2video model_name 探测:\n\n")
	for _, r := range results {
		mark := "[ ]"
		if r.OK {
			mark = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %-22s %s\n", mark, r.Model, r.Message))
	}
	for _, r := range results {
		if r.OK {
			b.WriteString(fmt.Sprintf("\n建议在 config/stacks/video-native-short.yaml 设置:\n  text_model: %s\n", r.Model))
			return b.String()
		}
	}
	b.WriteString("\n全部 model_name 不可用。请在可灵控制台确认已开通文生视频 API，或改用 strategy: image2video。\n")
	return b.String()
}
