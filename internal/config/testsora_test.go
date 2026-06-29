package config

import "testing"

func TestSoraStackReadiness(t *testing.T) {
	p := Providers{
		DeepSeek:  DeepSeekConfig{APIKey: "sk-d"},
		DashScope: DashScopeConfig{APIKey: "sk-b"},
		OpenAI:    OpenAIConfig{APIKey: "sk-o"},
	}
	if missing := SoraStackReadiness(p); len(missing) != 0 {
		t.Fatalf("expected ready, missing %v", missing)
	}
	p2 := Providers{DeepSeek: DeepSeekConfig{APIKey: "sk-d"}, DashScope: DashScopeConfig{APIKey: "sk-b"}}
	if missing := SoraStackReadiness(p2); len(missing) == 0 {
		t.Fatal("expected missing openai")
	}
}
