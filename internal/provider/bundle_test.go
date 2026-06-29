package provider

import (
	"os"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
)

func TestSelectTTSVolcengineStackNoDashFallback(t *testing.T) {
	t.Setenv("FLOWAGENT_TTS_FALLBACK", "")
	app := &config.App{
		Providers: config.Providers{
			DashScope: config.DashScopeConfig{APIKey: "sk-test"},
			Volcengine: config.VolcengineConfig{
				AppID:     "123",
				AccessKey: "token-not-ark",
			},
		},
		Stack: &config.Stack{
			TTS: map[string]any{"provider": "volcengine"},
		},
	}
	c := selectTTS(app)
	if _, ok := c.(*tts.Volcengine); !ok {
		t.Fatalf("expected *tts.Volcengine, got %T", c)
	}
}

func TestSelectTTSVolcengineStackExplicitFallbackEnv(t *testing.T) {
	t.Setenv("FLOWAGENT_TTS_FALLBACK", "dashscope")
	t.Cleanup(func() { os.Unsetenv("FLOWAGENT_TTS_FALLBACK") })
	app := &config.App{
		Providers: config.Providers{
			DashScope: config.DashScopeConfig{APIKey: "sk-test"},
			Volcengine: config.VolcengineConfig{
				AppID:     "123",
				AccessKey: "token-not-ark",
			},
		},
		Stack: &config.Stack{
			TTS: map[string]any{"provider": "volcengine"},
		},
	}
	c := selectTTS(app)
	if _, ok := c.(*tts.ResourceGrantFallback); !ok {
		t.Fatalf("expected *tts.ResourceGrantFallback, got %T", c)
	}
}

func TestSelectTTSVolcengineStackMissingCreds(t *testing.T) {
	app := &config.App{
		Providers: config.Providers{
			DashScope: config.DashScopeConfig{APIKey: "sk-test"},
			Volcengine: config.VolcengineConfig{
				AccessKey: "ark-should-not-use",
			},
		},
		Stack: &config.Stack{
			TTS: map[string]any{"provider": "volcengine"},
		},
	}
	c := selectTTS(app)
	if _, ok := c.(tts.Noop); !ok {
		t.Fatalf("expected noop when volcengine tts not configured, got %T", c)
	}
}

func TestSelectTTSExplicitFallbackEnv(t *testing.T) {
	os.Setenv("FLOWAGENT_TTS_FALLBACK", "dashscope")
	t.Cleanup(func() { os.Unsetenv("FLOWAGENT_TTS_FALLBACK") })
	app := &config.App{
		Providers: config.Providers{
			DashScope: config.DashScopeConfig{APIKey: "sk-test"},
			Volcengine: config.VolcengineConfig{
				AppID:     "123",
				AccessKey: "token",
			},
		},
	}
	c := selectTTS(app)
	if _, ok := c.(*tts.Fallback); !ok {
		t.Fatalf("expected Fallback when FLOWAGENT_TTS_FALLBACK=dashscope, got %T", c)
	}
}
