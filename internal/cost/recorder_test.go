package cost

import (
	"testing"

	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestRecorderSyncTo(t *testing.T) {
	rec := NewRecorder(DefaultRates())
	rec.AddLLM(llm.TokenUsage{PromptTokens: 1000, CompletionTokens: 2000})
	rec.AddTTS(5000)
	rec.AddImage(10)
	rec.AddVideo(30)

	ledger := &artifacts.CostLedger{}
	rec.SyncTo(ledger)

	if ledger.LLMCNY <= 0 || ledger.TTSCNY <= 0 || ledger.ImageCNY <= 0 || ledger.VideoCNY <= 0 {
		t.Fatalf("expected non-zero costs: %+v", ledger)
	}
	if ledger.TotalCNY != ledger.LLMCNY+ledger.TTSCNY+ledger.ImageCNY+ledger.VideoCNY+ledger.OtherCNY {
		t.Fatalf("total mismatch: %+v", ledger)
	}
}
