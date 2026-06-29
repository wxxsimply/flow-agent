package agent

import "testing"

func TestExtractTopLevelJSON(t *testing.T) {
	raw := "说明文字\n```json\n{\"a\":1,\"b\":{\"c\":2}}\n```\n尾部"
	got := ExtractTopLevelJSON(raw)
	want := `{"a":1,"b":{"c":2}}`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
