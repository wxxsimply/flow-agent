package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
)

// ProbeStackTTS synthesizes via the same client stack produce uses.
func ProbeStackTTS(app *config.App) (ok bool, detail string) {
	if app == nil || app.Stack == nil {
		return false, "stack not loaded"
	}
	client := selectTTS(app)
	if client == nil {
		return false, "no tts client"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	audio, err := client.Synthesize(ctx, tts.SynthesizeRequest{
		SSML:       "旁白测试",
		Voice:      "zh_male_m191_uranus_bigtts",
		Format:     "mp3",
		ResourceID: "seed-tts-2.0",
	})
	if err != nil {
		return false, err.Error()
	}
	if len(audio) == 0 {
		return false, "empty audio"
	}
	return true, fmt.Sprintf("client=%T bytes=%d", client, len(audio))
}

func FormatStackTTSProbeReport(ok bool, detail string) string {
	var b strings.Builder
	if ok {
		b.WriteString("[OK] " + detail + "\n")
	} else {
		b.WriteString("[FAIL] " + detail + "\n")
	}
	return b.String()
}
