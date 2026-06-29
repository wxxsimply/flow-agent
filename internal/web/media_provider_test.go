package web

import (
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestApplyCustomMediaProvider_openai(t *testing.T) {
	stack := &config.Stack{
		Image: map[string]any{"provider": "volcengine"},
		Video: map[string]any{"provider": "volcengine"},
	}
	app := &config.App{
		Providers: config.Providers{},
		Stack:     stack,
	}
	entry := &customMediaProvider{
		ID:      "test-openai",
		Label:   "Sora Proxy",
		Adapter: mediaAdapterOpenAI,
		APIKey:  "sk-test",
		BaseURL: "https://proxy.example/v1",
	}
	applyCustomMediaProvider(app, entry)
	if app.Stack.ImageConfig().Provider != "openai" {
		t.Fatalf("image provider got %q", app.Stack.ImageConfig().Provider)
	}
	if app.Stack.VideoConfig().Provider != "openai" {
		t.Fatalf("video provider got %q", app.Stack.VideoConfig().Provider)
	}
	if app.Providers.OpenAI.APIKey != "sk-test" {
		t.Fatalf("openai key not set")
	}
	if app.Providers.OpenAI.BaseURL != "https://proxy.example/v1" {
		t.Fatalf("openai base_url got %q", app.Providers.OpenAI.BaseURL)
	}
}

func TestApplyCustomMediaProvider_volcengine(t *testing.T) {
	stack := &config.Stack{Image: map[string]any{}, Video: map[string]any{}}
	app := &config.App{Providers: config.Providers{}, Stack: stack}
	entry := &customMediaProvider{
		Adapter: mediaAdapterVolcengine,
		APIKey:  "ark-test",
		BaseURL: "https://ark.example/v3",
	}
	applyCustomMediaProvider(app, entry)
	if app.Stack.ImageConfig().Provider != "volcengine" {
		t.Fatalf("image provider got %q", app.Stack.ImageConfig().Provider)
	}
	if app.Providers.Volcengine.APIKey != "ark-test" {
		t.Fatal("volcengine key not set")
	}
}

func TestApplyCustomMediaProvider_kling(t *testing.T) {
	stack := &config.Stack{Image: map[string]any{}, Video: map[string]any{}}
	app := &config.App{Providers: config.Providers{}, Stack: stack}
	entry := &customMediaProvider{
		Adapter:   mediaAdapterKling,
		APIKey:    "ak-test",
		SecretKey: "sk-test",
		BaseURL:   "https://kling.example",
	}
	applyCustomMediaProvider(app, entry)
	if app.Stack.VideoConfig().Provider != "kling" {
		t.Fatalf("video provider got %q", app.Stack.VideoConfig().Provider)
	}
	if app.Providers.Kling.AccessKey != "ak-test" || app.Providers.Kling.SecretKey != "sk-test" {
		t.Fatal("kling keys not set")
	}
}

func TestApplyCustomMediaProvider_modelOverride(t *testing.T) {
	stack := &config.Stack{Image: map[string]any{}, Video: map[string]any{}}
	app := &config.App{Providers: config.Providers{}, Stack: stack}
	entry := &customMediaProvider{
		Adapter:    mediaAdapterOpenAI,
		APIKey:     "sk-test",
		ImageModel: "dall-e-2",
		VideoModel: "sora-2",
	}
	applyCustomMediaProvider(app, entry)
	if app.Stack.Image["model"] != "dall-e-2" {
		t.Fatalf("image model got %v", app.Stack.Image["model"])
	}
	if app.Stack.Video["model"] != "sora-2" {
		t.Fatalf("video model got %v", app.Stack.Video["model"])
	}
}

func TestMigrateLegacyMediaProviders_volcengine(t *testing.T) {
	p := &userPrefs{
		Providers: providersUserCreds{
			Volcengine: &providerUserCreds{APIKey: "ark-legacy"},
		},
	}
	migrateLegacyMediaProviders(p)
	if len(p.CustomMediaProviders) != 1 {
		t.Fatalf("want 1 provider, got %d", len(p.CustomMediaProviders))
	}
	if p.CustomMediaProviders[0].Adapter != mediaAdapterVolcengine {
		t.Fatalf("adapter got %q", p.CustomMediaProviders[0].Adapter)
	}
	if p.ActiveMediaProviderID != p.CustomMediaProviders[0].ID {
		t.Fatal("active id not set")
	}
}

func TestMediaProviderConfigured_kling(t *testing.T) {
	if mediaProviderConfigured(&customMediaProvider{Adapter: mediaAdapterKling, APIKey: "ak"}) {
		t.Fatal("kling should need secret")
	}
	if !mediaProviderConfigured(&customMediaProvider{Adapter: mediaAdapterKling, APIKey: "ak", SecretKey: "sk"}) {
		t.Fatal("kling should be configured with ak+sk")
	}
}

func TestResolveActiveMediaProvider(t *testing.T) {
	p := &userPrefs{
		CustomMediaProviders: []customMediaProvider{
			{ID: "a", Label: "First", Adapter: mediaAdapterOpenAI, APIKey: "k1"},
			{ID: "b", Label: "Second", Adapter: mediaAdapterVolcengine, APIKey: "k2"},
		},
		ActiveMediaProviderID: "b",
	}
	active := resolveActiveMediaProvider(p)
	if active == nil || active.ID != "b" {
		t.Fatalf("active got %+v", active)
	}
}

func TestSanitizeCustomMediaProviders(t *testing.T) {
	list := sanitizeCustomMediaProviders([]customMediaProvider{
		{Label: "", Adapter: mediaAdapterOpenAI, APIKey: "k"},
		{ID: "dup", Adapter: mediaAdapterOpenAI},
		{ID: "dup", Adapter: mediaAdapterVolcengine},
	})
	if len(list) != 2 {
		t.Fatalf("want 2 entries, got %d", len(list))
	}
	if list[0].ID == "" {
		t.Fatal("id should be generated")
	}
	if list[0].Label == "" {
		t.Fatal("label should be filled from catalog")
	}
}
