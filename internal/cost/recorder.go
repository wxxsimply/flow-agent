package cost

import (
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// Totals 累计原始用量（未折算）。
type Totals struct {
	LLMPromptTokens     int
	LLMCompletionTokens int
	TTSCharacters       int
	ImageCount          int
	VideoSeconds        float64
	VideoAPICalls       int
}

// Recorder 单次 run 的成本累计器。
type Recorder struct {
	rates  Rates
	totals Totals
}

// NewRecorder 创建记账器。
func NewRecorder(rates Rates) *Recorder {
	return &Recorder{rates: rates}
}

// AddLLM 累加 LLM token 用量。
func (rec *Recorder) AddLLM(u llm.TokenUsage) {
	rec.totals.LLMPromptTokens += u.PromptTokens
	rec.totals.LLMCompletionTokens += u.CompletionTokens
}

// AddTTS 累加 TTS 字符数。
func (rec *Recorder) AddTTS(chars int) {
	if chars > 0 {
		rec.totals.TTSCharacters += chars
	}
}

// AddImage 累加成功生成的图片张数。
func (rec *Recorder) AddImage(n int) {
	if n > 0 {
		rec.totals.ImageCount += n
	}
}

// AddVideo 累加 AI 视频秒数。
func (rec *Recorder) AddVideo(seconds float64) {
	if seconds > 0 {
		rec.totals.VideoSeconds += seconds
	}
}

// AddVideoAPICall 累加视频 API 调用次数（含 BoN 候选与重试）。
func (rec *Recorder) AddVideoAPICall(n int) {
	if n > 0 {
		rec.totals.VideoAPICalls += n
	}
}

// TotalsSnapshot 返回当前累计用量副本。
func (rec *Recorder) TotalsSnapshot() Totals {
	return rec.totals
}

// RestoreFrom 从 ledger 恢复累计用量（resume 续跑时用）。
func (rec *Recorder) RestoreFrom(ledger *artifacts.CostLedger) {
	if ledger == nil {
		return
	}
	rec.totals = Totals{
		LLMPromptTokens:     ledger.LLMPromptTokens,
		LLMCompletionTokens: ledger.LLMCompletionTokens,
		TTSCharacters:       ledger.TTSCharacters,
		ImageCount:          ledger.ImageCount,
		VideoSeconds:        ledger.VideoSeconds,
		VideoAPICalls:       ledger.VideoAPICalls,
	}
}

// SyncTo 按当前累计用量重算 manifest 各分项（覆盖，非增量叠加）。
func (rec *Recorder) SyncTo(ledger *artifacts.CostLedger) {
	if ledger == nil {
		return
	}
	t := rec.totals
	ledger.LLMPromptTokens = t.LLMPromptTokens
	ledger.LLMCompletionTokens = t.LLMCompletionTokens
	ledger.TTSCharacters = t.TTSCharacters
	ledger.ImageCount = t.ImageCount
	ledger.VideoSeconds = t.VideoSeconds
	ledger.VideoAPICalls = t.VideoAPICalls
	ledger.LLMCNY = rec.rates.LLMCNY(t.LLMPromptTokens, t.LLMCompletionTokens)
	ledger.TTSCNY = rec.rates.TTSCNY(t.TTSCharacters)
	ledger.ImageCNY = rec.rates.ImageCNY(t.ImageCount)
	ledger.VideoCNY = rec.rates.VideoCNY(t.VideoSeconds)
	ledger.Recalc()
}
