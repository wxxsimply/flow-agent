package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func writeJSON(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCharacterStateMigratesLegacyPath(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(seriesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(seriesDir, "character-state.json")
	writeJSON(t, legacyPath, map[string]any{
		"林晚": map[string]any{"mood": "冷静"},
	})

	v := ForSeries(&config.App{SeriesDir: dir}, "demo")
	state, err := v.LoadCharacterState()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if state["林晚"] == nil {
		t.Fatal("expected migrated 林晚 entry")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy path should be gone, got err=%v", err)
	}
	newPath := filepath.Join(seriesDir, "vault", "character-state.json")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new path should exist: %v", err)
	}
}

func TestEnsureCharacterStateCleansLegacyPollution(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(seriesDir, "vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(seriesDir, "vault", "character-state.json")
	writeJSON(t, statePath, map[string]any{
		"林晚": map[string]any{
			"role":          "女主",
			"traits":        "冷静、隐忍",
			"known_secrets": []any{legacyInferredSecretMarker, "其它真实秘密"},
			"notes":         "听完录音后情绪克制，可有短暂心理留白",
		},
	})

	v := ForSeries(&config.App{SeriesDir: dir}, "demo")
	if err := v.EnsureCharacterStateFromBible(); err != nil {
		t.Fatal(err)
	}
	state, err := v.LoadCharacterState()
	if err != nil {
		t.Fatal(err)
	}
	lin, ok := state["林晚"].(map[string]any)
	if !ok {
		t.Fatal("林晚 missing or wrong type")
	}
	secrets, _ := lin["known_secrets"].([]any)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret remaining, got %d: %v", len(secrets), secrets)
	}
	if s, _ := secrets[0].(string); strings.Contains(s, legacyInferredSecretMarker) {
		t.Fatalf("polluted secret still present: %s", s)
	}
	if _, ok := lin["notes"]; ok {
		t.Fatal("legacy notes should be removed")
	}
}

func TestEnsureCharacterStateNoopWhenClean(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(seriesDir, "vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	clean := map[string]any{
		"林晚": map[string]any{
			"role":          "女主",
			"traits":        "冷静、隐忍",
			"known_secrets": []any{"手中已收集证据"},
			"notes":         "实时进展记录",
		},
	}
	statePath := filepath.Join(seriesDir, "vault", "character-state.json")
	writeJSON(t, statePath, clean)
	before, _ := os.ReadFile(statePath)

	v := ForSeries(&config.App{SeriesDir: dir}, "demo")
	if err := v.EnsureCharacterStateFromBible(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(statePath)
	if string(before) != string(after) {
		t.Fatalf("clean state should not be modified\nbefore: %s\nafter: %s", before, after)
	}
}

func TestEnsureCharacterStateSeedsFromBible(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(seriesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bible := `series_id: demo
characters:
  - name: 林晚
    role: 女主
    traits: 冷静
  - name: 顾沉
    role: 男主
    traits: 腹黑
`
	if err := os.WriteFile(filepath.Join(seriesDir, "series-bible.yaml"), []byte(bible), 0o644); err != nil {
		t.Fatal(err)
	}
	v := ForSeries(&config.App{SeriesDir: dir}, "demo")
	if err := v.EnsureCharacterStateFromBible(); err != nil {
		t.Fatal(err)
	}
	state, err := v.LoadCharacterState()
	if err != nil {
		t.Fatal(err)
	}
	if state["林晚"] == nil || state["顾沉"] == nil {
		t.Fatalf("expected seeded characters, got: %v", state)
	}
}
