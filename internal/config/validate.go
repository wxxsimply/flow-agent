package config

import (
	"fmt"
	"strings"
)

// KeyStatus 单个 Provider 的配置状态。
type KeyStatus struct {
	Name    string
	OK      bool
	Hint    string
	Phase   string // 对应 IMPLEMENTATION_ROADMAP 阶段
}

// ValidateProviders 检查密钥是否已配置（不校验网络连通性）。
func ValidateProviders(p Providers) []KeyStatus {
	return []KeyStatus{
		{
			Name:  "deepseek",
			OK:    strings.TrimSpace(p.DeepSeek.APIKey) != "",
			Hint:  "deepseek.api_key 或环境变量 DEEPSEEK_API_KEY",
			Phase: "B（Plan/Write）— 建议最先配置",
		},
		{
			Name:  "dashscope",
			OK:    strings.TrimSpace(p.DashScope.APIKey) != "",
			Hint:  "百炼控制台 sk- → dashscope.api_key（https://bailian.console.aliyun.com/）",
			Phase: "C/D/E（Qwen/万相，原 DashScope）",
		},
		{
			Name:  "volcengine_tts",
			OK:    p.VolcengineTTSConfigured(),
			Hint:  "OpenSpeech app_id + access_key；运行 flowagent config test-volcengine-tts 诊断",
			Phase: "E（旁白 TTS，cap5/seedance 栈）",
		},
		{
			Name: "volcengine",
			OK: strings.TrimSpace(p.Volcengine.SecretKey) != "" &&
				(p.VolcengineTTSConfigured() || p.VolcengineArkConfigured()),
			Hint:  "语音 secret_key；TTS 见 volcengine_tts；出图/视频见 volcengine_ark",
			Phase: "E（火山语音 + 方舟）",
		},
		{
			Name: "volcengine_ark",
			OK:    p.VolcengineArkConfigured(),
			Hint:  "volcengine.api_key 或 VOLCENGINE_API_KEY（方舟 Seedream/Seedance）",
			Phase: "E（出图/视频，micro-movie-seedance stack）",
		},
		{
			Name:  "kling",
			OK:    p.KlingEnabled(),
			Hint:  "kling.access_key + secret_key 或 KLING_ACCESS_KEY / KLING_SECRET_KEY（可选，无则 Ken Burns）",
			Phase: "K2（图生视频，可选）",
		},
		{
			Name:  "gemini",
			OK:    p.GeminiEnabled(),
			Hint:  "gemini.api_key 或 GEMINI_API_KEY（可选，micro-movie-veo-lite stack）",
			Phase: "E（Gemini Veo i2v，国外备选）",
		},
		{
			Name:  "openai",
			OK:    p.OpenAIEnabled(),
			Hint:  "openai.api_key 或 OPENAI_API_KEY（Sora Videos API，micro-movie-sora stack）",
			Phase: "E（OpenAI Sora i2v）",
		},
	}
}

// FormatCheckReport 生成人类可读的检查报告。
func FormatCheckReport(path string, statuses []KeyStatus) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("config: %s\n\n", path))
	ok, total := 0, 0
	for _, s := range statuses {
		total++
		mark := "[ ]"
		if s.OK {
			mark = "[x]"
			ok++
		}
		optional := ""
		if s.Name == "kling" || s.Name == "gemini" || s.Name == "openai" {
			optional = " (optional)"
		}
		b.WriteString(fmt.Sprintf("%s %-12s %s\n    阶段: %s\n    配置: %s\n\n", mark, s.Name+optional, "", s.Phase, s.Hint))
	}
	b.WriteString(fmt.Sprintf("summary: %d/%d configured\n", ok, total))
	if ok == 0 {
		b.WriteString("\n提示: 复制 config/providers.local.yaml.example → config/providers.local.yaml 并填入密钥\n")
	}
	return b.String()
}
