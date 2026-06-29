package llm

import (
	"strings"
	"testing"
)

func TestParseSSEStream(t *testing.T) {
	body := strings.NewReader(`data: {"choices":[{"delta":{"content":"你"}}]}

data: {"choices":[{"delta":{"content":"好"}}]}

data: [DONE]

`)
	var got strings.Builder
	err := parseSSEStream(body, func(s string) error {
		got.WriteString(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "你好" {
		t.Fatalf("got %q", got.String())
	}
}

func TestExtractDeltaContent(t *testing.T) {
	s, err := extractDeltaContent(`{"choices":[{"delta":{"content":"x"}}]}`)
	if err != nil || s != "x" {
		t.Fatalf("delta: %q err=%v", s, err)
	}
}

func TestBuildChatPayload_V4JSONDisablesThinking(t *testing.T) {
	p := buildChatPayload(CompletionRequest{
		Model:    "deepseek-v4-flash",
		User:     "hi",
		JSONMode: true,
	}, false)
	if p.Thinking == nil || p.Thinking.Type != "disabled" {
		t.Fatalf("expected thinking disabled, got %+v", p.Thinking)
	}
	if p.Model != "deepseek-v4-flash" {
		t.Fatalf("model=%q", p.Model)
	}
}

func TestBuildChatPayload_DefaultModel(t *testing.T) {
	p := buildChatPayload(CompletionRequest{User: "hi"}, false)
	if p.Model != "deepseek-v4-flash" {
		t.Fatalf("default model=%q", p.Model)
	}
}
