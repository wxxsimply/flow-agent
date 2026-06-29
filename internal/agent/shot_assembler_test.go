package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestRunShotAssembler_openingShotDryRun(t *testing.T) {
	dir := t.TempDir()
	artDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	opening := "雨夜，少年独自站在公司天台，霓虹在雨幕中模糊成一片。"
	rc := &runctx.Context{
		RunDir:       dir,
		ArtifactsDir: artDir,
		EpisodeNo:    1,
		Workflow:     "micro-movie",
		DryRun:       true,
		Manifest:     &artifacts.Manifest{Artifacts: []artifacts.ArtifactEntry{}},
		Creative: &artifacts.CreativeOptions{
			InputMode:         "director",
			OpeningShot:       opening,
			TargetDurationSec: 150,
		},
	}
	if err := RunShotAssembler(rc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(artDir, "opening-shot-input.md")); err != nil {
		t.Fatalf("missing opening shot: %v", err)
	}
	brief, err := os.ReadFile(filepath.Join(artDir, "shot-language-brief.md"))
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(string(brief)) < 2500 {
		t.Fatalf("brief too short: %d runes", utf8.RuneCountInString(string(brief)))
	}
	data, err := os.ReadFile(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sb artifacts.Storyboard
	if err := json.Unmarshal(data, &sb); err != nil {
		t.Fatal(err)
	}
	if len(sb.Shots) < 12 {
		t.Fatalf("expected auto-generated shots >=12, got %d", len(sb.Shots))
	}
	if !sb.Shots[0].Expanded {
		t.Fatal("expected expanded shots")
	}
	if !strings.Contains(sb.Shots[0].VisualPrompt, "雨夜") {
		t.Fatalf("shot1 should lock opening: %q", sb.Shots[0].VisualPrompt)
	}
}
