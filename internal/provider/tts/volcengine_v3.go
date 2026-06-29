package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func speechRateFromRatio(ratio float64) int {
	if ratio <= 0 {
		ratio = 1.0
	}
	r := int((ratio - 1.0) * 100)
	if r < -50 {
		return -50
	}
	if r > 100 {
		return 100
	}
	return r
}

func (v *Volcengine) synthesizeV3(ctx context.Context, text, voiceType, format string, speed float64, resourceID string) ([]byte, error) {
	if resourceID == "" {
		resourceID = "seed-tts-2.0"
	}
	if format == "" {
		format = "mp3"
	}
	reqParams := map[string]any{
		"text":    text,
		"speaker": voiceType,
		"audio_params": map[string]any{
			"format":      format,
			"sample_rate": 24000,
			"speech_rate": speechRateFromRatio(speed),
		},
	}
	if strings.HasPrefix(strings.TrimSpace(voiceType), "S_") {
		additions, _ := json.Marshal(map[string]any{"model_type": 4})
		reqParams["additions"] = string(additions)
	}
	payload := map[string]any{
		"user":       map[string]any{"uid": "flowagent"},
		"req_params": reqParams,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	reqID := uuid.New().String()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openspeech.bytedance.com/api/v3/tts/unidirectional", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-App-Id", v.appID)
	httpReq.Header.Set("X-Api-Access-Key", v.accessToken)
	httpReq.Header.Set("X-Api-Resource-Id", resourceID)
	httpReq.Header.Set("X-Api-Request-Id", reqID)

	resp, err := v.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("volcengine tts v3: %s: %s", resp.Status, string(raw))
	}

	var audio []byte
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk struct {
			Code    int     `json:"code"`
			Message string  `json:"message"`
			Data    *string `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Code != 0 && chunk.Code != 20000000 {
			if chunk.Message != "" {
				return nil, fmt.Errorf("volcengine tts v3: code=%d: %s", chunk.Code, chunk.Message)
			}
		}
		if chunk.Data != nil && *chunk.Data != "" {
			part, err := decodeBase64Audio(*chunk.Data)
			if err != nil {
				return nil, err
			}
			audio = append(audio, part...)
		}
		if chunk.Code == 20000000 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("volcengine tts v3: empty audio data")
	}
	return audio, nil
}
