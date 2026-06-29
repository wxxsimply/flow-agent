package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

// DashScope 百炼语音合成（Qwen-TTS / CosyVoice，与出图同 Key、同地域）。
type DashScope struct {
	apiKey  string
	baseURL string
	region  config.BailianRegion
	http    *http.Client
}

// NewDashScope 构造百炼 TTS 客户端。
func NewDashScope(p config.Providers) *DashScope {
	region := config.BailianRegion(p.DashScope.Region)
	base := config.BailianNativeBaseURL(region)
	if custom := strings.TrimSpace(p.DashScope.BaseURL); custom != "" {
		custom = strings.TrimRight(custom, "/")
		custom = strings.Replace(custom, "/compatible-mode/v1", "", 1)
		if strings.Contains(custom, "dashscope") {
			base = custom
		}
	}
	return &DashScope{
		apiKey:  p.DashScope.APIKey,
		baseURL: strings.TrimRight(base, "/"),
		region:  region,
		http:    &http.Client{Timeout: 180 * time.Second},
	}
}

// Synthesize 将 SSML/文本合成为音频（优先 Qwen-TTS，回退 CosyVoice）。
func (d *DashScope) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	text := stripSSML(req.SSML)
	if text == "" {
		return nil, fmt.Errorf("dashscope tts: empty text")
	}
	format := orDefault(req.Format, "mp3")
	speed := req.SpeedRatio
	if speed <= 0 {
		speed = 1.0
	}
	voice := config.DashScopeVoiceFor(strings.TrimSpace(req.Voice))
	if voice == "" {
		voice = "longanyang"
	}

	audio, err := d.synthesizeQwen(ctx, text, voice, speed)
	if err == nil {
		return audio, nil
	}
	qwenErr := err

	// CosyVoice 仅北京地域
	if config.NormalizeBailianRegion(d.region) == config.RegionCNBeijing {
		audio, err = d.synthesizeCosyVoice(ctx, text, format, voice, speed)
		if err == nil {
			return audio, nil
		}
		return nil, fmt.Errorf("dashscope tts: qwen failed (%v); cosyvoice failed (%w)", qwenErr, err)
	}
	return nil, fmt.Errorf("dashscope tts: qwen failed (%w); cosyvoice only available in cn-beijing", qwenErr)
}

func (d *DashScope) synthesizeQwen(ctx context.Context, text, voice string, speed float64) ([]byte, error) {
	input := map[string]any{
		"text":          text,
		"voice":         orDefault(voice, "longanyang"),
		"language_type": "Chinese",
	}
	if speed > 0 && speed != 1.0 {
		input["speech_rate"] = speed
	}
	body := map[string]any{
		"model": "qwen3-tts-flash",
		"input": input,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := d.baseURL + "/api/v1/services/aigc/multimodal-generation/generation"
	raw, err := d.post(ctx, url, data)
	if err != nil {
		return nil, err
	}
	audioURL, err := parseAudioURL(raw)
	if err != nil {
		return nil, err
	}
	return fetchURL(ctx, d.http, audioURL)
}

func (d *DashScope) synthesizeCosyVoice(ctx context.Context, text, format, voice string, speed float64) ([]byte, error) {
	in := map[string]any{
		"text":        text,
		"voice":       orDefault(voice, "longanyang"),
		"format":      format,
		"sample_rate": 24000,
	}
	if speed > 0 && speed != 1.0 {
		in["speech_rate"] = speed
	}
	body := map[string]any{
		"model": "cosyvoice-v3-flash",
		"input": in,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	// CosyVoice 非实时仅北京
	base := config.BailianNativeBaseURL(config.RegionCNBeijing)
	url := strings.TrimRight(base, "/") + "/api/v1/services/audio/tts/SpeechSynthesizer"
	raw, err := d.post(ctx, url, data)
	if err != nil {
		return nil, err
	}
	audioURL, err := parseAudioURL(raw)
	if err != nil {
		return nil, err
	}
	return fetchURL(ctx, d.http, audioURL)
}

func (d *DashScope) post(ctx context.Context, url string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	resp, err := d.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s: %s", resp.Status, string(raw))
	}
	return raw, nil
}

func parseAudioURL(raw []byte) (string, error) {
	var out struct {
		Output struct {
			Audio struct {
				URL string `json:"url"`
			} `json:"audio"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Code != "" {
		return "", fmt.Errorf("%s: %s", out.Code, out.Message)
	}
	if out.Output.Audio.URL != "" {
		return out.Output.Audio.URL, nil
	}
	return "", fmt.Errorf("no audio url in response")
}

func fetchURL(ctx context.Context, httpClient *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch audio: %s: %s", resp.Status, string(b))
	}
	return io.ReadAll(resp.Body)
}

func stripSSML(s string) string {
	s = strings.ReplaceAll(s, "<speak>", "")
	s = strings.ReplaceAll(s, "</speak>", "")
	s = strings.ReplaceAll(s, "<p>", "")
	s = strings.ReplaceAll(s, "</p>", "")
	s = strings.ReplaceAll(s, "<break time=\"500ms\"/>", " ")
	s = strings.ReplaceAll(s, "<?xml version=\"1.0\"?>", "")
	return strings.TrimSpace(s)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
