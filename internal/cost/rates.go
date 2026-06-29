package cost

import "github.com/flow-agent/flow-agent/internal/config"

// Rates 将用量折算为人民币（元）的单价。
type Rates struct {
	LLMInputPer1KTokens  float64
	LLMOutputPer1KTokens float64
	TTSPer1KChars        float64
	ImagePerShot         float64
	VideoPerSecond       float64
}

// DefaultRates 标准档参考单价（与 config/stacks/standard-tier.yaml cost_targets 中值对齐）。
func DefaultRates() Rates {
	return Rates{
		LLMInputPer1KTokens:  0.0014,
		LLMOutputPer1KTokens: 0.0028,
		TTSPer1KChars:        0.12,
		ImagePerShot:         0.38,
		VideoPerSecond:       0.25,
	}
}

// RatesFromStack 从 stack 的 unit_prices_cny 读取；缺省用 DefaultRates。
func RatesFromStack(s *config.Stack) Rates {
	r := DefaultRates()
	if s == nil || s.UnitPricesCNY == nil {
		return r
	}
	u := s.UnitPricesCNY
	if v := u.LLMInputPer1KTokens; v > 0 {
		r.LLMInputPer1KTokens = v
	}
	if v := u.LLMOutputPer1KTokens; v > 0 {
		r.LLMOutputPer1KTokens = v
	}
	if v := u.TTSPer1KChars; v > 0 {
		r.TTSPer1KChars = v
	}
	if v := u.ImagePerShot; v > 0 {
		r.ImagePerShot = v
	}
	if v := u.VideoPerSecond; v > 0 {
		r.VideoPerSecond = v
	}
	return r
}

// LLMCNY 按 token 折算 LLM 成本。
func (r Rates) LLMCNY(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)/1000*r.LLMInputPer1KTokens +
		float64(completionTokens)/1000*r.LLMOutputPer1KTokens
}

// TTSCNY 按字符折算 TTS 成本。
func (r Rates) TTSCNY(chars int) float64 {
	return float64(chars) / 1000 * r.TTSPer1KChars
}

// ImageCNY 按张数折算出图成本。
func (r Rates) ImageCNY(count int) float64 {
	return float64(count) * r.ImagePerShot
}

// VideoCNY 按秒数折算视频生成成本。
func (r Rates) VideoCNY(seconds float64) float64 {
	return seconds * r.VideoPerSecond
}
