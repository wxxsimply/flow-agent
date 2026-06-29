package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

type billingFailTTS struct{}

func (billingFailTTS) Synthesize(context.Context, tts.SynthesizeRequest) ([]byte, error) {
	return nil, fmt.Errorf(`volcengine tts v3: code=403: resource not granted`)
}

func TestSynthesizeShotTextVolcengine(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip(err)
	}
	app, err := config.Load(root, "micro-movie-seedance")
	if err != nil {
		t.Fatal(err)
	}
	if !app.Providers.VolcengineTTSConfigured() {
		t.Skip("volcengine TTS not configured (app_id + access_key in providers.local.yaml)")
	}

	dir := t.TempDir()
	rc := &runctx.Context{
		RunDir:    dir,
		App:       app,
		Providers: provider.NewBundle(app),
		Creative: &artifacts.CreativeOptions{
			NarratorVoice: "documentary_male",
		},
	}
	out := filepath.Join(dir, "s01.mp3")
	err = synthesizeShotText(context.Background(), rc, app.Stack.TTSConfig(), "测试火山语音旁白", out)
	if err != nil {
		if tts.IsNonRetryableError(err) {
			t.Skipf("tts billing/permission unavailable: %v", err)
		}
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil || st.Size() < 1000 {
		t.Fatalf("expected mp3 output, stat err=%v size=%d", err, st.Size())
	}
}

func TestVerifyTTSReadyFailsByDefaultOnBillingFailure(t *testing.T) {
	t.Setenv("FLOWAGENT_TTS_ALLOW_SILENT", "")
	app := &config.App{
		Providers: config.Providers{
			Volcengine: config.VolcengineConfig{AppID: "1", AccessKey: "token"},
		},
		Stack: &config.Stack{
			TTS: map[string]any{
				"provider": "volcengine",
				"format":   "mp3",
				"product":  "doubao-speech-2.0-emotion",
			},
		},
	}
	rc := &runctx.Context{
		App: app,
		Providers: &provider.Bundle{
			TTS: billingFailTTS{},
		},
		Creative: &artifacts.CreativeOptions{NarratorVoice: "epic_male"},
	}
	err := verifyTTSReady(context.Background(), rc, app.Stack.TTSConfig())
	if err == nil {
		t.Fatal("expected error when TTS fails and FLOWAGENT_TTS_ALLOW_SILENT is unset")
	}
	if !strings.Contains(err.Error(), "tts unavailable") {
		t.Fatalf("expected tts unavailable error, got %v", err)
	}
}

func TestVerifyTTSReadySilentWhenAllowSilentEnv(t *testing.T) {
	t.Setenv("FLOWAGENT_TTS_ALLOW_SILENT", "1")
	app := &config.App{
		Providers: config.Providers{
			Volcengine: config.VolcengineConfig{AppID: "1", AccessKey: "token"},
		},
		Stack: &config.Stack{
			TTS: map[string]any{
				"provider": "volcengine",
				"format":   "mp3",
				"product":  "doubao-speech-2.0-emotion",
			},
		},
	}
	rc := &runctx.Context{
		App: app,
		Providers: &provider.Bundle{
			TTS: billingFailTTS{},
		},
		Creative: &artifacts.CreativeOptions{NarratorVoice: "epic_male"},
	}
	err := verifyTTSReady(context.Background(), rc, app.Stack.TTSConfig())
	if err != nil {
		t.Fatalf("expected silent fallback (nil) with FLOWAGENT_TTS_ALLOW_SILENT=1, got %v", err)
	}
}

func TestSynthesizeShotTextVolcengineMissingConfig(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip(err)
	}
	app, err := config.Load(root, "micro-movie-cap5")
	if err != nil {
		t.Fatal(err)
	}
	if app.Providers.VolcengineTTSConfigured() {
		t.Skip("volcengine TTS already configured")
	}
	if strings.TrimSpace(app.Stack.TTSConfig().Provider) != "volcengine" {
		t.Skip("stack tts provider not volcengine")
	}

	dir := t.TempDir()
	rc := &runctx.Context{
		RunDir:    dir,
		App:       app,
		Providers: provider.NewBundle(app),
	}
	out := filepath.Join(dir, "s01.mp3")
	err = synthesizeShotText(context.Background(), rc, app.Stack.TTSConfig(), "测试", out)
	if err == nil || !strings.Contains(err.Error(), "tts not configured") {
		t.Fatalf("expected tts config error, got %v", err)
	}
}
