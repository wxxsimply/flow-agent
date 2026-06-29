package config

import "testing"

func TestArkAPIKey(t *testing.T) {
	p := Providers{Volcengine: VolcengineConfig{APIKey: "ark-test"}}
	if got := p.ArkAPIKey(); got != "ark-test" {
		t.Fatalf("got %q", got)
	}
	p2 := Providers{Volcengine: VolcengineConfig{AccessKey: "ark-fallback"}}
	if got := p2.ArkAPIKey(); got != "ark-fallback" {
		t.Fatalf("fallback %q", got)
	}
}

func TestMediaProduceConfiguredVolcengineStack(t *testing.T) {
	stack := &Stack{
		Image: map[string]any{"provider": "volcengine"},
		Video: map[string]any{"provider": "volcengine", "enabled": true},
	}
	p := Providers{Volcengine: VolcengineConfig{APIKey: "ark-x"}}
	if !p.MediaProduceConfigured(stack) {
		t.Fatal("expected ok")
	}
	p2 := Providers{}
	if p2.MediaProduceConfigured(stack) {
		t.Fatal("expected missing ark key")
	}
}

func TestMediaProduceConfiguredSoraStack(t *testing.T) {
	stack := &Stack{
		Image: map[string]any{"provider": "bailian"},
		Video: map[string]any{"provider": "openai", "enabled": true},
	}
	p := Providers{DashScope: DashScopeConfig{APIKey: "sk-x"}, OpenAI: OpenAIConfig{APIKey: "sk-oai"}}
	if !p.MediaProduceConfigured(stack) {
		t.Fatal("expected sora stack ok")
	}
	p2 := Providers{DashScope: DashScopeConfig{APIKey: "sk-x"}}
	if p2.MediaProduceConfigured(stack) {
		t.Fatal("expected missing openai")
	}
}
