package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Stack 对应 config/stacks/*.yaml（标准档模型与预算）。
type Stack struct {
	Name              string               `yaml:"name"`
	Version           string               `yaml:"version"`
	Description       string               `yaml:"description"`
	TargetDurationSec int                  `yaml:"target_duration_sec"`
	CostBudgetCNY     float64              `yaml:"cost_budget_cny"`
	// CostBudgetPer30SecCNY 按成片时长线性缩放总预算（元）；例如 5 表示每 30 秒 5 元。
	CostBudgetPer30SecCNY float64          `yaml:"cost_budget_per_30_sec_cny"`
	LLM               map[string]LLMRef    `yaml:"llm"`
	TTS               map[string]any       `yaml:"tts"`
	Image             map[string]any       `yaml:"image"`
	Video             map[string]any       `yaml:"video"`
	Assemble          map[string]any       `yaml:"assemble"`
	Compose           map[string]any       `yaml:"compose"`
	Publish           map[string]any       `yaml:"publish"`
	CostTargetsCNY    map[string][]float64 `yaml:"cost_targets_cny"`
	UnitPricesCNY     *UnitPricesCNY       `yaml:"unit_prices_cny"`
}

// UnitPricesCNY 用量折算单价（人民币），供 cost.Recorder 使用。
type UnitPricesCNY struct {
	LLMInputPer1KTokens  float64 `yaml:"llm_input_per_1k_tokens"`
	LLMOutputPer1KTokens float64 `yaml:"llm_output_per_1k_tokens"`
	TTSPer1KChars        float64 `yaml:"tts_per_1k_chars"`
	ImagePerShot         float64 `yaml:"image_per_shot"`
	VideoPerSecond       float64 `yaml:"video_per_second"`
}

// LLMRef 某一阶段使用的模型提供方与模型名。
type LLMRef struct {
	Provider       string `yaml:"provider"`
	Model          string `yaml:"model"`
	Stream         bool   `yaml:"stream"`
	ChunkMaxTokens int    `yaml:"chunk_max_tokens"`
}

// LoadStack 加载技术栈配置文件。
func LoadStack(path string) (*Stack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Stack
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
