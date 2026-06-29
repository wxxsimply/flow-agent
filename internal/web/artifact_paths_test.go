package web

import "testing"

func TestResolveArtifactAbsSafe_rejectsTraversal(t *testing.T) {
	runDir := t.TempDir()
	_, err := resolveArtifactAbsSafe(runDir, "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestResolveArtifactAbsSafe_allowsNormalPath(t *testing.T) {
	runDir := t.TempDir()
	abs, err := resolveArtifactAbsSafe(runDir, "artifacts/storyboard.json")
	if err != nil {
		t.Fatal(err)
	}
	if abs == "" {
		t.Fatal("expected path")
	}
}

func TestCORSConfig_allowOrigin(t *testing.T) {
	cfg := loadCORSConfig(false)
	if got := cfg.allowOrigin("http://127.0.0.1:8080"); got != "http://127.0.0.1:8080" {
		t.Fatalf("local origin: %q", got)
	}
	if got := cfg.allowOrigin("https://evil.example"); got != "" {
		t.Fatalf("unexpected allow: %q", got)
	}
}
