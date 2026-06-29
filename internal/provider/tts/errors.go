package tts

import (
	"os"
	"strings"
)

// UseSilentOnBillingFailure 默认 false：TTS 失败时阻断 produce。
// 设 FLOWAGENT_TTS_ALLOW_SILENT=1 可用静音轨继续出视频。
func UseSilentOnBillingFailure() bool {
	return strings.TrimSpace(os.Getenv("FLOWAGENT_TTS_ALLOW_SILENT")) == "1"
}

// AllowSilentOnFailure 与 UseSilentOnBillingFailure 相同（兼容旧名）。
func AllowSilentOnFailure() bool {
	return UseSilentOnBillingFailure()
}

// IsNonRetryableError reports billing, permission, or configuration errors.
func IsNonRetryableError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return IsVolcengineResourceGrantError(err) ||
		strings.Contains(s, "arrearage") ||
		strings.Contains(s, "not configured") ||
		strings.Contains(s, "access denied, please make sure your account is in good standing")
}

// UserHint returns a short remediation hint for common TTS failures.
func UserHint(err error) string {
	if err == nil {
		return ""
	}
	if IsVolcengineResourceGrantError(err) {
		return "请在火山控制台 https://console.volcengine.com/speech/service 开通「语音合成2.0」并确认应用 AppID/Token 已绑定；运行 flowagent config test-volcengine-tts 诊断"
	}
	if strings.Contains(strings.ToLower(err.Error()), "arrearage") {
		return "百炼 TTS 欠费；当前 stack 旁白仅走火山，请配置 volcengine.app_id + access_key"
	}
	return ""
}
