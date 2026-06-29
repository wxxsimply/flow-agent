package config

import "strings"

// BailianRegion 百炼 API Key 所属地域，须与控制台创建 Key 时选择的地域一致。
// 文档：https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope
type BailianRegion string

const (
	RegionCNBeijing BailianRegion = "cn-beijing" // 华北2（北京）中国内地
	RegionIntl      BailianRegion = "intl"       // 新加坡（国际版控制台常见）
	RegionUS        BailianRegion = "us"         // 美国（弗吉尼亚）
	RegionHK        BailianRegion = "hk"         // 中国香港
)

// BailianNativeBaseURL 返回百炼原生 API 根地址（万相/语音异步任务等，非 compatible-mode）。
func BailianNativeBaseURL(region BailianRegion) string {
	switch normalizeRegion(region) {
	case RegionIntl:
		return "https://dashscope-intl.aliyuncs.com"
	case RegionUS:
		return "https://dashscope-us.aliyuncs.com"
	case RegionHK:
		return "https://cn-hongkong.dashscope.aliyuncs.com"
	default:
		return "https://dashscope.aliyuncs.com"
	}
}

// BailianCompatibleBaseURL 返回 OpenAI 兼容模式 base_url（用于 Qwen Chat）。
func BailianCompatibleBaseURL(region BailianRegion) string {
	switch normalizeRegion(region) {
	case RegionIntl:
		return "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
	case RegionUS:
		return "https://dashscope-us.aliyuncs.com/compatible-mode/v1"
	case RegionHK:
		return "https://cn-hongkong.dashscope.aliyuncs.com/compatible-mode/v1"
	default:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
}

// BailianChatCompletionsURL 完整 chat/completions 地址（用于连通性测试）。
func BailianChatCompletionsURL(region BailianRegion) string {
	return BailianCompatibleBaseURL(region) + "/chat/completions"
}

// ResolveDashScopeBaseURL 优先使用配置里的 base_url，否则按 region 选择官方端点。
func (p Providers) ResolveDashScopeBaseURL() string {
	if strings.TrimSpace(p.DashScope.BaseURL) != "" {
		return strings.TrimSpace(p.DashScope.BaseURL)
	}
	return BailianCompatibleBaseURL(BailianRegion(p.DashScope.Region))
}

// NormalizeBailianRegion 导出地域规范化（供 TTS/Image 等使用）。
func NormalizeBailianRegion(r BailianRegion) BailianRegion {
	return normalizeRegion(r)
}

func normalizeRegion(r BailianRegion) BailianRegion {
	switch strings.ToLower(strings.TrimSpace(string(r))) {
	case "intl", "sg", "singapore", "国际", "新加坡":
		return RegionIntl
	case "us", "virginia", "美国":
		return RegionUS
	case "hk", "hongkong", "香港":
		return RegionHK
	case "cn", "cn-beijing", "beijing", "北京", "华北", "":
		return RegionCNBeijing
	default:
		return BailianRegion(strings.ToLower(string(r)))
	}
}

// AllBailianRegions 用于 test-api 依次探测。
func AllBailianRegions() []BailianRegion {
	return []BailianRegion{RegionCNBeijing, RegionIntl, RegionUS, RegionHK}
}
