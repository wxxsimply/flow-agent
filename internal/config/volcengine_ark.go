package config

import "strings"

const DefaultArkBaseURL = "https://ark.cn-beijing.volces.com/api/v3"

// ArkBaseURL 火山方舟 API 根地址。
func (v VolcengineConfig) ArkBaseURL() string {
	if u := strings.TrimSpace(v.BaseURL); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultArkBaseURL
}

// ArkAPIKey 方舟 API Key（出图/视频）；与 OpenSpeech TTS token 分离。
func (p Providers) ArkAPIKey() string {
	if k := strings.TrimSpace(p.Volcengine.APIKey); k != "" {
		return k
	}
	// 兼容：部分用户把 ark Key 写在 access_key
	if k := strings.TrimSpace(p.Volcengine.AccessKey); strings.HasPrefix(k, "ark-") {
		return k
	}
	return ""
}

// VolcengineArkConfigured 是否可调用方舟 Seedream/Seedance。
func (p Providers) VolcengineArkConfigured() bool {
	return p.ArkAPIKey() != ""
}

// VolcengineTTSConfigured 是否可调用火山豆包语音 OpenSpeech TTS。
// 需要 app_id + access_key（OpenSpeech Token）；勿将方舟 ark- Key 填在 access_key。
func (p Providers) VolcengineTTSConfigured() bool {
	if strings.TrimSpace(p.Volcengine.AppID) == "" {
		return false
	}
	token := strings.TrimSpace(p.Volcengine.AccessKey)
	if token == "" || strings.HasPrefix(token, "ark-") {
		return false
	}
	return true
}

// MediaProduceConfigured stack 出图/视频所需凭证是否就绪。
func (p Providers) MediaProduceConfigured(stack *Stack) bool {
	if stack == nil {
		return strings.TrimSpace(p.DashScope.APIKey) != "" || p.VolcengineArkConfigured()
	}
	img := strings.ToLower(strings.TrimSpace(stack.ImageConfig().Provider))
	vid := strings.ToLower(strings.TrimSpace(stack.VideoConfig().Provider))
	if isOpenAIVideoProvider(img) && isOpenAIVideoProvider(vid) {
		return p.OpenAIEnabled()
	}
	if isGeminiVideoProvider(img) && isGeminiVideoProvider(vid) {
		return p.GeminiEnabled()
	}
	if img == "kling" && vid == "kling" {
		return p.KlingEnabled()
	}
	if isDashscopeMediaProvider(img) && isDashscopeMediaProvider(vid) {
		return strings.TrimSpace(p.DashScope.APIKey) != ""
	}
	if isOpenAIVideoProvider(vid) && !p.OpenAIEnabled() {
		return false
	}
	if isGeminiVideoProvider(vid) && !p.GeminiEnabled() {
		return false
	}
	if isVolcengineMediaProvider(img) || isVolcengineMediaProvider(vid) {
		return p.VolcengineArkConfigured() || strings.TrimSpace(p.DashScope.APIKey) != ""
	}
	if isOpenAIVideoProvider(vid) {
		return p.OpenAIEnabled() && (p.VolcengineArkConfigured() || strings.TrimSpace(p.DashScope.APIKey) != "")
	}
	return strings.TrimSpace(p.DashScope.APIKey) != ""
}

func isDashscopeMediaProvider(name string) bool {
	switch name {
	case "dashscope", "wan", "bailian":
		return true
	default:
		return strings.Contains(name, "dashscope") || strings.Contains(name, "wan")
	}
}

func isOpenAIVideoProvider(name string) bool {
	switch name {
	case "openai", "sora":
		return true
	default:
		return false
	}
}

func isGeminiVideoProvider(name string) bool {
	switch name {
	case "gemini", "veo", "google":
		return true
	default:
		return false
	}
}

func isVolcengineMediaProvider(name string) bool {
	switch name {
	case "volcengine", "seedream", "seedance", "jimeng", "ark":
		return true
	default:
		return false
	}
}
