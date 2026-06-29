package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Providers 各 AI 服务商的连接配置。
type Providers struct {
	DeepSeek   DeepSeekConfig   `yaml:"deepseek"`
	DashScope  DashScopeConfig  `yaml:"dashscope"`
	Volcengine VolcengineConfig `yaml:"volcengine"`
	Kling      KlingConfig      `yaml:"kling"`
	Gemini     GeminiConfig     `yaml:"gemini"`
	OpenAI     OpenAIConfig     `yaml:"openai"`
}

// OpenAIConfig OpenAI API（Sora Videos API 等）。
type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"` // 默认 https://api.openai.com/v1
}

// GeminiConfig Google Gemini API（Veo 图生视频等）。
type GeminiConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"` // 默认 https://generativelanguage.googleapis.com/v1beta
}

// DeepSeekConfig DeepSeek / OpenAI 兼容接口。
type DeepSeekConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

// DashScopeConfig 阿里云百炼（原灵积 DashScope）— Key 在 bailian.console.aliyun.com 申请。
type DashScopeConfig struct {
	APIKey  string `yaml:"api_key"`
	Region  string `yaml:"region"`  // cn-beijing | intl | us | hk，须与控制台 Key 地域一致
	BaseURL string `yaml:"base_url"` // 留空则按 region 自动选择官方端点
}

// VolcengineConfig 火山引擎（豆包语音 + 方舟 Seedream/Seedance）。
type VolcengineConfig struct {
	AccessKey string `yaml:"access_key"` // OpenSpeech TTS Access Token（非 ark-）
	SecretKey string `yaml:"secret_key"`
	AppID     string `yaml:"app_id"`
	APIKey    string `yaml:"api_key"`  // 方舟 API Key（出图/视频），通常 ark- 开头
	BaseURL   string `yaml:"base_url"` // 默认 https://ark.cn-beijing.volces.com/api/v3
}

// KlingConfig 可灵图生视频（官方 JWT：access_key + secret_key）。
type KlingConfig struct {
	APIKey    string `yaml:"api_key"`    // 遗留字段，不再用于 JWT
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	BaseURL   string `yaml:"base_url"`
}

// LoadProviders 从 YAML 读取密钥，并由环境变量覆盖。
func LoadProviders(path string) (Providers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Providers{}, err
	}
	var p Providers
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Providers{}, err
	}
	applyProviderEnv(&p)
	return p, nil
}

// ApplyProviderEnv 从环境变量填充 Providers（Desktop 空 base 或 YAML 加载后均可调用）。
func ApplyProviderEnv(p *Providers) {
	applyProviderEnv(p)
}

// applyProviderEnv 环境变量优先于文件（便于 CI / 本地不把密钥写入磁盘）。
func applyProviderEnv(p *Providers) {
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		p.DeepSeek.APIKey = v
	}
	if v := os.Getenv("DASHSCOPE_API_KEY"); v != "" {
		p.DashScope.APIKey = v
	}
	if v := os.Getenv("DASHSCOPE_REGION"); v != "" {
		p.DashScope.Region = v
	}
	if v := os.Getenv("DASHSCOPE_BASE_URL"); v != "" {
		p.DashScope.BaseURL = v
	}
	if v := os.Getenv("VOLCENGINE_ACCESS_KEY"); v != "" {
		p.Volcengine.AccessKey = v
	}
	if v := os.Getenv("VOLCENGINE_SECRET_KEY"); v != "" {
		p.Volcengine.SecretKey = v
	}
	if v := os.Getenv("VOLCENGINE_API_KEY"); v != "" {
		p.Volcengine.APIKey = v
	}
	if v := os.Getenv("VOLCENGINE_ARK_BASE_URL"); v != "" {
		p.Volcengine.BaseURL = v
	}
	if v := os.Getenv("KLING_API_KEY"); v != "" {
		p.Kling.APIKey = v
	}
	if v := os.Getenv("KLING_ACCESS_KEY"); v != "" {
		p.Kling.AccessKey = v
	}
	if v := os.Getenv("KLING_SECRET_KEY"); v != "" {
		p.Kling.SecretKey = v
	}
	if v := os.Getenv("KLING_BASE_URL"); v != "" {
		p.Kling.BaseURL = v
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		p.Gemini.APIKey = v
	}
	if v := os.Getenv("GEMINI_BASE_URL"); v != "" {
		p.Gemini.BaseURL = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		p.OpenAI.APIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		p.OpenAI.BaseURL = v
	}
}

// OpenAIEnabled 是否可调用 OpenAI Videos API（Sora）。
func (p Providers) OpenAIEnabled() bool {
	return strings.TrimSpace(p.OpenAI.APIKey) != ""
}

// GeminiEnabled 是否可调用 Gemini Veo API。
func (p Providers) GeminiEnabled() bool {
	return strings.TrimSpace(p.Gemini.APIKey) != ""
}

// KlingEnabled 是否可调用可灵 API。
func (p Providers) KlingEnabled() bool {
	return strings.TrimSpace(p.Kling.AccessKey) != "" && strings.TrimSpace(p.Kling.SecretKey) != ""
}
