package web

import (
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestMergeProvidersFromPrefs_overridesBaseURL(t *testing.T) {
	base := config.Providers{
		Volcengine: config.VolcengineConfig{APIKey: "file-key"},
	}
	prefs := &userPrefs{
		Providers: providersUserCreds{
			Volcengine: &providerUserCreds{
				APIKey:  "pref-key",
				BaseURL: "https://proxy.example.com/v3",
			},
		},
	}
	merged := mergeProvidersFromPrefs(base, prefs)
	if merged.Volcengine.APIKey != "pref-key" {
		t.Fatalf("api_key want pref-key got %q", merged.Volcengine.APIKey)
	}
	if merged.Volcengine.BaseURL != "https://proxy.example.com/v3" {
		t.Fatalf("base_url want proxy got %q", merged.Volcengine.BaseURL)
	}
}

func TestMergeProvidersFromPrefs_legacyVolcengine(t *testing.T) {
	base := config.Providers{}
	prefs := &userPrefs{
		Volcengine: volcengineUserCreds{APIKey: "legacy-ark"},
	}
	merged := mergeProvidersFromPrefs(base, prefs)
	if merged.Volcengine.APIKey != "legacy-ark" {
		t.Fatalf("want legacy key, got %q", merged.Volcengine.APIKey)
	}
	if merged.Volcengine.BaseURL != config.DefaultArkBaseURL {
		t.Fatalf("want default ark url, got %q", merged.Volcengine.BaseURL)
	}
}

func TestMergeProvidersFromPrefs_dashscope(t *testing.T) {
	base := config.Providers{}
	prefs := &userPrefs{
		Providers: providersUserCreds{
			DashScope: &providerUserCreds{
				APIKey:  "sk-test",
				BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
				Region:  "cn-beijing",
			},
		},
	}
	merged := mergeProvidersFromPrefs(base, prefs)
	if merged.DashScope.APIKey != "sk-test" {
		t.Fatalf("dashscope key not merged")
	}
	if merged.DashScope.Region != "cn-beijing" {
		t.Fatalf("region not merged")
	}
}

func TestMediaReadyForStack_seedance(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	p := config.Providers{
		Volcengine: config.VolcengineConfig{APIKey: "ark-test"},
	}
	if !mediaReadyForStack(root, DefaultStudioStack, p) {
		t.Fatal("expected seedance stack ready with ark key")
	}
	p2 := config.Providers{}
	if mediaReadyForStack(root, DefaultStudioStack, p2) {
		t.Fatal("expected not ready without keys")
	}
}

func TestListStudioStacks(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stacks, err := listStudioStacks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) == 0 {
		t.Fatal("expected at least one micro-movie stack")
	}
	found := false
	for _, s := range stacks {
		if s.Name == DefaultStudioStack {
			found = true
			if s.ImageProvider == "" || s.VideoProvider == "" {
				t.Fatalf("stack missing providers: %+v", s)
			}
		}
	}
	if !found {
		t.Fatalf("default stack not in list: %v", stacks)
	}
	_ = filepath.Join(root, "config", "stacks")
}

func TestBaseProvidersForRuntime_desktopIgnoresYAML(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	yamlBase := baseProvidersForRuntime(root, false)
	if yamlBase.Volcengine.APIKey == "" && yamlBase.DashScope.APIKey == "" && yamlBase.DeepSeek.APIKey == "" {
		t.Skip("no providers.local.yaml keys in dev tree")
	}
	desktopBase := baseProvidersForRuntime(root, true)
	if desktopBase.Volcengine.APIKey != "" || desktopBase.DashScope.APIKey != "" {
		t.Fatalf("desktop base should not load yaml keys: volc=%q dash=%q",
			desktopBase.Volcengine.APIKey, desktopBase.DashScope.APIKey)
	}
}

func TestDesktopMediaReady_emptyPrefs(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	base := baseProvidersForRuntime(root, true)
	merged := mergeProvidersFromPrefs(base, &userPrefs{StackProfile: DefaultStudioStack})
	if mediaReadyForStack(root, DefaultStudioStack, merged) {
		t.Fatal("expected not ready with empty desktop prefs")
	}
	merged = mergeProvidersFromPrefs(base, &userPrefs{
		StackProfile: DefaultStudioStack,
		Providers: providersUserCreds{
			Volcengine: &providerUserCreds{APIKey: "ark-test"},
		},
	})
	if !mediaReadyForStack(root, DefaultStudioStack, merged) {
		t.Fatal("expected ready when prefs supply ark key")
	}
}

func TestMediaReadyForPrefs_customOpenAI(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	prefs := &userPrefs{
		StackProfile: DefaultStudioStack,
		CustomMediaProviders: []customMediaProvider{
			{ID: "o1", Adapter: mediaAdapterOpenAI, APIKey: "sk-test", Label: "Sora"},
		},
		ActiveMediaProviderID: "o1",
	}
	base := baseProvidersForRuntime(root, true)
	merged := mergeProvidersFromPrefs(base, prefs)
	if !mediaReadyForPrefs(root, DefaultStudioStack, merged, prefs) {
		t.Fatal("expected ready with active openai provider")
	}
	prefs.CustomMediaProviders[0].APIKey = ""
	if mediaReadyForPrefs(root, DefaultStudioStack, merged, prefs) {
		t.Fatal("expected not ready without api key")
	}
}

func TestEffectiveStackProfile(t *testing.T) {
	prefs := &userPrefs{StackProfile: "micro-movie-sora"}
	if got := effectiveStackProfile(prefs, ""); got != StudioStackFinal {
		t.Fatalf("legacy sora got %q, want %q", got, StudioStackFinal)
	}
	if got := effectiveStackProfile(prefs, "micro-movie-wan-flash"); got != StudioStackFinal {
		t.Fatalf("override got %q", got)
	}
	if got := effectiveStackProfile(&userPrefs{StackProfile: StudioStackSample}, ""); got != StudioStackSample {
		t.Fatalf("sample got %q", got)
	}
}
