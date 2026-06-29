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

// Volcengine 火山豆包语音（OpenSpeech HTTP API）。
type Volcengine struct {
	appID       string
	accessToken string
	http        *http.Client
}

// NewVolcengine 构造火山 TTS 客户端。
func NewVolcengine(p config.Providers) *Volcengine {
	token := strings.TrimSpace(p.Volcengine.AccessKey)
	if token == "" {
		return nil
	}
	return &Volcengine{
		appID:       p.Volcengine.AppID,
		accessToken: token,
		http:        http.DefaultClient,
	}
}

// Synthesize 调用 v3 API（默认 seed-tts-2.0）；1.0 资源未开通时可回退 v1。
func (v *Volcengine) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("volcengine tts: not configured")
	}
	text := stripSSML(req.SSML)
	if text == "" {
		return nil, fmt.Errorf("volcengine tts: empty text")
	}
	speed := req.SpeedRatio
	if speed <= 0 {
		speed = 1.0
	}
	voiceType := orDefault(req.Voice, "zh_male_m191_uranus_bigtts")
	format := orDefault(req.Format, "mp3")
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = ResolveVolcResourceID(voiceType, "", "")
	}

	if audio, err := v.synthesizeV3(ctx, text, voiceType, format, speed, resourceID); err == nil {
		return audio, nil
	} else if IsVolcengineV2Resource(resourceID) {
		return nil, err
	} else if !IsVolcengineResourceGrantError(err) {
		return nil, err
	}

	return v.synthesizeV1(ctx, text, voiceType, format, speed)
}

func (v *Volcengine) synthesizeV1(ctx context.Context, text, voiceType, format string, speed float64) ([]byte, error) {
	cluster := "volcano_tts"
	payload := map[string]any{
		"app": map[string]any{
			"appid":   v.appID,
			"token":   v.accessToken,
			"cluster": cluster,
		},
		"user": map[string]any{"uid": "flowagent"},
		"audio": map[string]any{
			"voice_type":  voiceType,
			"encoding":    format,
			"speed_ratio": speed,
		},
		"request": map[string]any{
			"reqid":     fmt.Sprintf("fa-%d", time.Now().UnixNano()),
			"text":      text,
			"operation": "query",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openspeech.bytedance.com/api/v1/tts", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer;"+v.accessToken)

	resp, err := v.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("volcengine tts: %s: %s", resp.Status, string(raw))
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Data == "" {
		return nil, fmt.Errorf("volcengine tts: empty audio data")
	}
	return decodeBase64Audio(out.Data)
}
