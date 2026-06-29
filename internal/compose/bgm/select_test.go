package bgm

import (
	"os"
	"path/filepath"
	"testing"
)

func projectBGMDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "assets", "bgm")
		if st, err := os.Stat(filepath.Join(candidate, "neutral.mp3")); err == nil && !st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("assets/bgm not found from test cwd")
	return ""
}

func TestSelectPath_library(t *testing.T) {
	lib := projectBGMDir(t)

	p, mood, ok := SelectPath(lib, "悲伤", "")
	if !ok {
		t.Fatal("expected sad.mp3 in assets/bgm")
	}
	if mood != MoodSad {
		t.Fatalf("mood=%s want sad", mood)
	}
	if filepath.Base(p) != "sad.mp3" {
		t.Fatalf("path=%s want sad.mp3", p)
	}

	p, mood, ok = SelectPath(lib, "", "悬疑")
	if !ok || mood != MoodNeutral {
		t.Fatalf("suspense fallback: ok=%v mood=%s path=%s", ok, mood, p)
	}
}
