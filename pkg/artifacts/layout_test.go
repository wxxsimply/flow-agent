package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalWriteRel(t *testing.T) {
	if got := CanonicalWriteRel("artifacts/storyboard.json"); got != "artifacts/story/storyboard.json" {
		t.Fatalf("got %s", got)
	}
	if got := CanonicalWriteRel("artifacts/other.json"); got != "artifacts/other.json" {
		t.Fatalf("got %s", got)
	}
}

func TestResolvePath_fallbackNew(t *testing.T) {
	dir := t.TempDir()
	p := ResolvePath(dir, "artifacts/timeline.json")
	want := filepath.Join(dir, "artifacts/media/timeline.json")
	if p != want {
		t.Fatalf("got %s want %s", p, want)
	}
}

func TestResolvePath_prefersLegacy(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "artifacts", "timeline.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ResolvePath(dir, "artifacts/timeline.json"); got != legacy {
		t.Fatalf("got %s", got)
	}
}

func TestShotVideoRel(t *testing.T) {
	if got := ShotVideoRel("s01"); got != "artifacts/media/shots/s01.mp4" {
		t.Fatalf("got %s", got)
	}
}
