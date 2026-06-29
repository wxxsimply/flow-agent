package auth

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 云端认证与短信配置（环境变量）。
type Config struct {
	Enabled   bool
	JWTSecret string
	SMS       SMSConfig
	OTP       OTPConfig
}

// SMSConfig 短信服务商配置。
type SMSConfig struct {
	Provider string // dev | aliyun
	// 阿里云 SMS（provider=aliyun 时使用）
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	// dev 模式固定验证码
	DevCode string
}

// OTPConfig 验证码策略。
type OTPConfig struct {
	TTL       time.Duration
	ResendGap time.Duration
	CodeLen   int
}

// LoadConfigFromEnv 读取环境变量；未设置 JWT_SECRET 时 auth 仍可 dev 启动。
func LoadConfigFromEnv() Config {
	enabled := envBool("FLOWAGENT_AUTH_ENABLED", false)
	cfg := Config{
		Enabled:   enabled,
		JWTSecret: strings.TrimSpace(os.Getenv("FLOWAGENT_JWT_SECRET")),
		SMS: SMSConfig{
			Provider:        strings.TrimSpace(envDefault("FLOWAGENT_SMS_PROVIDER", "dev")),
			AccessKeyID:     strings.TrimSpace(os.Getenv("FLOWAGENT_SMS_ACCESS_KEY_ID")),
			AccessKeySecret: strings.TrimSpace(os.Getenv("FLOWAGENT_SMS_ACCESS_KEY_SECRET")),
			SignName:        strings.TrimSpace(os.Getenv("FLOWAGENT_SMS_SIGN_NAME")),
			TemplateCode:    strings.TrimSpace(os.Getenv("FLOWAGENT_SMS_TEMPLATE_CODE")),
			DevCode:         envDefault("FLOWAGENT_SMS_DEV_CODE", "123456"),
		},
		OTP: OTPConfig{
			TTL:       time.Duration(envInt("FLOWAGENT_OTP_TTL_SEC", 300)) * time.Second,
			ResendGap: time.Duration(envInt("FLOWAGENT_OTP_RESEND_SEC", 60)) * time.Second,
			CodeLen:   envInt("FLOWAGENT_OTP_CODE_LEN", 6),
		},
	}
	if cfg.JWTSecret == "" && cfg.Enabled {
		cfg.JWTSecret = "flowagent-dev-secret-change-me"
	}
	return cfg
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// DataDir 持久化数据根目录（runs/series/app.db）。
func DataDir(root string) string {
	if d := strings.TrimSpace(os.Getenv("FLOWAGENT_DATA_DIR")); d != "" {
		return d
	}
	return root
}
