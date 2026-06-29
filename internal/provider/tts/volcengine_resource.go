package tts

import "strings"

// ResolveVolcResourceID 解析 v3 API 的 X-Api-Resource-Id。
// stackProduct 来自 stack.tts.product（如 doubao-speech-2.0-emotion）；stackResource 可选显式 resource_id。
func ResolveVolcResourceID(voice, stackProduct, stackResource string) string {
	if id := strings.TrimSpace(stackResource); id != "" {
		return id
	}
	v := strings.TrimSpace(voice)
	switch {
	case strings.HasPrefix(v, "S_"):
		return "seed-icl-2.0"
	case strings.Contains(v, "_uranus_") || strings.HasPrefix(v, "saturn_"):
		return "seed-tts-2.0"
	case strings.HasPrefix(v, "BV") || strings.Contains(v, "_mars_") || strings.Contains(v, "_moon_"):
		return "seed-tts-1.0"
	}
	p := strings.ToLower(strings.TrimSpace(stackProduct))
	switch {
	case strings.Contains(p, "2.0"), strings.Contains(p, "2-0"), strings.Contains(p, "speech-2"):
		return "seed-tts-2.0"
	case strings.Contains(p, "1.0"), strings.Contains(p, "1-0"), strings.Contains(p, "speech-1"):
		return "seed-tts-1.0"
	}
	return "seed-tts-2.0"
}

// IsVolcengineV2Resource 2.0 / 复刻 2.0 无 v1 回退。
func IsVolcengineV2Resource(resourceID string) bool {
	switch strings.TrimSpace(resourceID) {
	case "seed-tts-2.0", "seed-icl-2.0":
		return true
	default:
		return false
	}
}
