package tts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
)

type ProbeResult struct {
	Voice string
	OK    bool
	Bytes int
	Err   string
}

// ProbeVoices tries several 2.0 voice types to diagnose 403/resource errors.
func ProbeVoices(p config.Providers) []ProbeResult {
	client := NewVolcengine(p)
	if client == nil {
		return []ProbeResult{{
			Voice: "-",
			Err:   "volcengine tts not configured (app_id + access_key required)",
		}}
	}
	voices := []string{
		"zh_male_m191_uranus_bigtts",
		"zh_male_liufei_uranus_bigtts",
		"zh_female_vv_uranus_bigtts",
		"zh_female_xiaohe_uranus_bigtts",
	}
	out := make([]ProbeResult, 0, len(voices))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resourceID := "seed-tts-2.0"
	for _, voice := range voices {
		audio, err := client.Synthesize(ctx, SynthesizeRequest{
			Voice:      voice,
			SSML:       "测试语音合成",
			Format:     "mp3",
			ResourceID: resourceID,
		})
		res := ProbeResult{Voice: voice}
		if err != nil {
			res.Err = err.Error()
		} else {
			res.OK = true
			res.Bytes = len(audio)
		}
		out = append(out, res)
	}
	return out
}

func FormatProbeReport(p config.Providers, results []ProbeResult) string {
	var b strings.Builder
	b.WriteString("--- Volcengine TTS probe (语音合成 2.0 / seed-tts-2.0) ---\n")
	if !p.VolcengineTTSConfigured() {
		b.WriteString("volcengine_tts: NOT configured\n")
		b.WriteString("  set volcengine.app_id + access_key (OpenSpeech Token) in providers.local.yaml\n")
		b.WriteString("  access_key 必须是语音 Token，不能填 ark- 方舟 Key\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("app_id: %s (len=%d)\n", maskProbe(p.Volcengine.AppID), len(strings.TrimSpace(p.Volcengine.AppID))))
	b.WriteString(fmt.Sprintf("access_key: %s (len=%d, ark=%v)\n",
		maskProbe(p.Volcengine.AccessKey),
		len(strings.TrimSpace(p.Volcengine.AccessKey)),
		strings.HasPrefix(strings.TrimSpace(p.Volcengine.AccessKey), "ark-"),
	))
	for _, r := range results {
		if r.OK {
			b.WriteString(fmt.Sprintf("[OK] voice=%s bytes=%d\n", r.Voice, r.Bytes))
		} else {
			b.WriteString(fmt.Sprintf("[FAIL] voice=%s err=%s\n", r.Voice, r.Err))
		}
	}
	b.WriteString("\n若全部失败且含 resource not granted：\n")
	b.WriteString("  1. 打开 https://console.volcengine.com/speech/service 确认「语音合成2.0」已开通\n")
	b.WriteString("  2. 应用管理里确认 AppID/Token 与 2.0 服务绑定（建议新建应用并勾选 2.0）\n")
	b.WriteString("  3. 音色需为 2.0 系列（*_uranus_bigtts），勿用 BV*_streaming（1.0）\n")
	b.WriteString("  旁白默认走 seed-tts-2.0；TTS 失败默认阻断 produce。\n")
	b.WriteString("  需静音继续出视频时设 FLOWAGENT_TTS_ALLOW_SILENT=1\n")
	return b.String()
}

func maskProbe(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
