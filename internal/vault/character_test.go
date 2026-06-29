package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestApplyCharacterPatch(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(seriesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	v := ForSeries(&config.App{SeriesDir: dir}, "demo")
	if err := v.ApplyCharacterPatch(map[string]any{"林晚": map[string]any{"mood": "冷静"}}); err != nil {
		t.Fatal(err)
	}
	state, err := v.LoadCharacterState()
	if err != nil {
		t.Fatal(err)
	}
	if state["林晚"] == nil {
		t.Fatal("expected 林晚 in state")
	}
}
