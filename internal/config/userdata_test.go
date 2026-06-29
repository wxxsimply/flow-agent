package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUserDataDir_nonEmpty(t *testing.T) {
	dir, err := UserDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(dir) == "" {
		t.Fatal("UserDataDir returned empty path")
	}
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(strings.ToLower(dir), "flowagent") {
			t.Fatalf("unexpected windows path: %q", dir)
		}
	case "darwin":
		if !strings.Contains(dir, "Application Support") {
			t.Fatalf("unexpected darwin path: %q", dir)
		}
	default:
		if !strings.Contains(dir, "flow-agent") {
			t.Fatalf("unexpected unix path: %q", dir)
		}
	}
	_ = filepath.Base(dir)
}

func TestEnsureUserDataDir_creates(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("APPDATA override test for windows")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	if err := EnsureUserDataDir(); err != nil {
		t.Fatal(err)
	}
	dir, err := UserDataDir()
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		t.Fatalf("dir not created: %v", err)
	}
}

func TestUsesIsolatedUserConfig(t *testing.T) {
	root, err := FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	t.Setenv("FLOWAGENT_DATA_DIR", "")
	if UsesIsolatedUserConfig() {
		t.Fatal("expected false when env unset")
	}
	t.Setenv("FLOWAGENT_DATA_DIR", root)
	if UsesIsolatedUserConfig() {
		t.Fatal("expected false when data dir equals root")
	}
	t.Setenv("FLOWAGENT_DATA_DIR", filepath.Join(root, "isolated-data"))
	if !UsesIsolatedUserConfig() {
		t.Fatal("expected true when data dir differs from root")
	}
}

func TestProviderConfigHint_isolated(t *testing.T) {
	root, err := FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	t.Setenv("FLOWAGENT_DATA_DIR", filepath.Join(root, "isolated-data"))
	if !strings.Contains(ProviderConfigHintZh(), "环境设置") {
		t.Fatalf("want settings hint, got %q", ProviderConfigHintZh())
	}
	t.Setenv("FLOWAGENT_DATA_DIR", root)
	if !strings.Contains(ProviderConfigHintZh(), "providers.local.yaml") {
		t.Fatalf("want yaml hint, got %q", ProviderConfigHintZh())
	}
}
