package config

import (
	"fmt"
	"strings"
)

// SoraStackReadiness 检查 micro-movie-sora 所需凭证（不调用 OpenAI）。
func SoraStackReadiness(p Providers) []string {
	var missing []string
	if !p.OpenAIEnabled() {
		missing = append(missing, "openai.api_key 或 OPENAI_API_KEY（Sora Videos API）")
	}
	if strings.TrimSpace(p.DashScope.APIKey) == "" && !p.VolcengineArkConfigured() {
		missing = append(missing, "dashscope.api_key（万相出图）或 volcengine.api_key（Seedream 出图）")
	}
	if strings.TrimSpace(p.DeepSeek.APIKey) == "" {
		missing = append(missing, "deepseek.api_key（assemble 扩写）")
	}
	return missing
}

// FormatSoraReadinessReport 人类可读 Sora stack 就绪报告。
func FormatSoraReadinessReport(p Providers) string {
	missing := SoraStackReadiness(p)
	var b strings.Builder
	b.WriteString("micro-movie-sora 凭证检查\n\n")
	b.WriteString(fmt.Sprintf("  openai:    %s\n", boolMark(p.OpenAIEnabled())))
	b.WriteString(fmt.Sprintf("  dashscope: %s（万相 t2i 出图）\n", boolMark(strings.TrimSpace(p.DashScope.APIKey) != "")))
	b.WriteString(fmt.Sprintf("  volcengine_ark: %s（可选 Seedream）\n", boolMark(p.VolcengineArkConfigured())))
	b.WriteString(fmt.Sprintf("  deepseek:  %s\n", boolMark(strings.TrimSpace(p.DeepSeek.APIKey) != "")))
	if len(missing) == 0 {
		b.WriteString("\nOK 可运行 test-shot / run micro-movie --stack micro-movie-sora\n")
		return b.String()
	}
	b.WriteString("\n缺少:\n")
	for _, m := range missing {
		b.WriteString("  - " + m + "\n")
	}
	b.WriteString("\n在 config/providers.local.yaml 填入 openai.api_key，或 export OPENAI_API_KEY\n")
	b.WriteString("OpenAI Platform: https://platform.openai.com/api-keys（ChatGPT 订阅不含 API）\n")
	return b.String()
}

func boolMark(ok bool) string {
	if ok {
		return "[x]"
	}
	return "[ ]"
}
