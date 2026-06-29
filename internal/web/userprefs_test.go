package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserPrefs_roundTrip(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "providers.local.yaml.example"), []byte("deepseek: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &userPrefs{
		WorkspaceDir: filepath.Join(root, "workspace"),
		StackProfile: "micro-movie-seedance",
		Providers: providersUserCreds{
			Volcengine: &providerUserCreds{
				APIKey:  "ark-test",
				BaseURL: "https://custom.example/v3",
			},
		},
	}
	if err := saveUserPrefs(root, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadUserPrefs(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.StackProfile != "micro-movie-seedance" {
		t.Fatalf("stack_profile got %q", loaded.StackProfile)
	}
	if loaded.Providers.Volcengine == nil || loaded.Providers.Volcengine.APIKey != "ark-test" {
		t.Fatal("volcengine key not persisted")
	}
	if loaded.Providers.Volcengine.BaseURL != "https://custom.example/v3" {
		t.Fatalf("base_url got %q", loaded.Providers.Volcengine.BaseURL)
	}
}

func TestUserPrefs_customMediaRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "providers.local.yaml.example"), []byte("deepseek: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &userPrefs{
		WorkspaceDir: filepath.Join(root, "workspace"),
		StackProfile: DefaultStudioStack,
		CustomMediaProviders: []customMediaProvider{
			{
				ID:      "mp-1",
				Label:   "Sora Proxy",
				Adapter: mediaAdapterOpenAI,
				APIKey:  "sk-test",
				BaseURL: "https://proxy.example/v1",
			},
		},
		ActiveMediaProviderID: "mp-1",
	}
	if err := saveUserPrefs(root, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadUserPrefs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CustomMediaProviders) != 1 {
		t.Fatalf("custom providers count = %d", len(loaded.CustomMediaProviders))
	}
	if loaded.CustomMediaProviders[0].APIKey != "sk-test" {
		t.Fatal("api key not persisted")
	}
	if loaded.ActiveMediaProviderID != "mp-1" {
		t.Fatalf("active id got %q", loaded.ActiveMediaProviderID)
	}
}

func TestMigrateLegacyMediaProviders_skipsWhenCustomExists(t *testing.T) {
	p := &userPrefs{
		CustomMediaProviders: []customMediaProvider{
			{ID: "existing", Adapter: mediaAdapterOpenAI, APIKey: "k"},
		},
		Providers: providersUserCreds{
			Volcengine: &providerUserCreds{APIKey: "ark-should-not-migrate"},
		},
	}
	migrateLegacyMediaProviders(p)
	if len(p.CustomMediaProviders) != 1 {
		t.Fatalf("want 1 provider, got %d", len(p.CustomMediaProviders))
	}
}

func TestNormalizeLoadedPrefs_legacyVolcengine(t *testing.T) {
	p := &userPrefs{
		Volcengine: volcengineUserCreds{APIKey: "legacy"},
	}
	normalizeLoadedPrefs(p)
	if p.Providers.Volcengine == nil || p.Providers.Volcengine.APIKey != "legacy" {
		t.Fatal("legacy volcengine not migrated")
	}
	if p.StackProfile != DefaultStudioStack {
		t.Fatalf("default stack got %q", p.StackProfile)
	}
}
