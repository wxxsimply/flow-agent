package runctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWorkspaceDir_allowsNonempty(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "existing-project")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	abs, err := ValidateWorkspaceDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if abs != dir {
		t.Fatalf("got %s", abs)
	}
}

func TestValidateWorkspaceDir_rejectsProjectRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateWorkspaceDir(dir); err == nil {
		t.Fatal("expected error for project root")
	}
}

func TestAllocateProjectDir_unique(t *testing.T) {
	ws := t.TempDir()
	dir1, err := AllocateProjectDir(ws, "雨夜天台")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir1, "manifest.json")); !os.IsNotExist(err) {
		// dir created empty, no manifest yet — ok
	}
	dir2, err := AllocateProjectDir(ws, "雨夜天台")
	if err != nil {
		t.Fatal(err)
	}
	if dir1 == dir2 {
		t.Fatalf("expected unique dirs, got %s", dir1)
	}
}

func TestSlugProjectTitle(t *testing.T) {
	if got := slugProjectTitle(""); got != "project" {
		t.Fatalf("empty=%q", got)
	}
	if got := slugProjectTitle("Hello World!"); got != "Hello-World" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkspaceFromPath_projectUsesParent(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "my-movie")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := WorkspaceFromPath(project)
	if err != nil {
		t.Fatal(err)
	}
	if ws != dir {
		t.Fatalf("got workspace %s want %s", ws, dir)
	}
}
