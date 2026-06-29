package web

import (
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/google/uuid"
)

const (
	mediaAdapterOpenAI     = "openai"
	mediaAdapterVolcengine = "volcengine"
	mediaAdapterKling      = "kling"
	mediaAdapterGemini     = "gemini"
	mediaAdapterDashscope  = "dashscope"
)

type customMediaProvider struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Adapter    string `json:"adapter"`
	APIKey     string `json:"api_key,omitempty"`
	SecretKey  string `json:"secret_key,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
	ImageModel string `json:"image_model,omitempty"`
	VideoModel string `json:"video_model,omitempty"`
}

type mediaAdapterInfo struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	DefaultBaseURL  string `json:"default_base_url,omitempty"`
	NeedsSecretKey  bool   `json:"needs_secret_key"`
}

var mediaAdapterCatalog = []mediaAdapterInfo{
	{ID: mediaAdapterOpenAI, Label: "OpenAI / Sora", DefaultBaseURL: "https://api.openai.com/v1"},
	{ID: mediaAdapterVolcengine, Label: "火山 Seedance", DefaultBaseURL: config.DefaultArkBaseURL},
	{ID: mediaAdapterKling, Label: "可灵 Kling", DefaultBaseURL: "https://api-beijing.klingai.com", NeedsSecretKey: true},
	{ID: mediaAdapterGemini, Label: "Google Gemini / Veo", DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta"},
	{ID: mediaAdapterDashscope, Label: "百炼 / 万相", DefaultBaseURL: ""},
}

func normalizeMediaAdapter(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case mediaAdapterOpenAI, "sora":
		return mediaAdapterOpenAI
	case mediaAdapterVolcengine, "seedance", "seedream", "ark", "volc":
		return mediaAdapterVolcengine
	case mediaAdapterKling:
		return mediaAdapterKling
	case mediaAdapterGemini, "veo", "google":
		return mediaAdapterGemini
	case mediaAdapterDashscope, "bailian", "wan":
		return mediaAdapterDashscope
	default:
		return mediaAdapterVolcengine
	}
}

func migrateLegacyMediaProviders(p *userPrefs) {
	if p == nil {
		return
	}
	if len(p.CustomMediaProviders) > 0 {
		return
	}
	if p.Providers.Volcengine == nil && legacyVolcengineHasData(p.Volcengine) {
		p.Providers.Volcengine = &providerUserCreds{
			APIKey:    p.Volcengine.APIKey,
			BaseURL:   p.Volcengine.BaseURL,
			AccessKey: p.Volcengine.AccessKey,
			SecretKey: p.Volcengine.SecretKey,
			AppID:     p.Volcengine.AppID,
		}
	}
	tryAdd := func(label, adapter string, creds *providerUserCreds) {
		if creds == nil {
			return
		}
		entry := customMediaProvider{
			ID:      uuid.NewString(),
			Label:   label,
			Adapter: adapter,
		}
		switch adapter {
		case mediaAdapterKling:
			entry.APIKey = strings.TrimSpace(creds.AccessKey)
			if entry.APIKey == "" {
				entry.APIKey = strings.TrimSpace(creds.APIKey)
			}
			entry.SecretKey = strings.TrimSpace(creds.SecretKey)
			entry.BaseURL = strings.TrimSpace(creds.BaseURL)
			if entry.APIKey == "" || entry.SecretKey == "" {
				return
			}
		default:
			entry.APIKey = strings.TrimSpace(creds.APIKey)
			entry.BaseURL = strings.TrimSpace(creds.BaseURL)
			if entry.APIKey == "" {
				return
			}
		}
		p.CustomMediaProviders = append(p.CustomMediaProviders, entry)
		if strings.TrimSpace(p.ActiveMediaProviderID) == "" {
			p.ActiveMediaProviderID = entry.ID
		}
	}
	tryAdd("火山 Seedance", mediaAdapterVolcengine, p.Providers.Volcengine)
	tryAdd("OpenAI Sora", mediaAdapterOpenAI, p.Providers.OpenAI)
	tryAdd("可灵 Kling", mediaAdapterKling, p.Providers.Kling)
	tryAdd("Gemini Veo", mediaAdapterGemini, p.Providers.Gemini)
	tryAdd("百炼万相", mediaAdapterDashscope, p.Providers.DashScope)
}

func resolveActiveMediaProvider(p *userPrefs) *customMediaProvider {
	if p == nil {
		return nil
	}
	migrateLegacyMediaProviders(p)
	activeID := strings.TrimSpace(p.ActiveMediaProviderID)
	if activeID == "" && len(p.CustomMediaProviders) > 0 {
		activeID = p.CustomMediaProviders[0].ID
	}
	for i := range p.CustomMediaProviders {
		if p.CustomMediaProviders[i].ID == activeID {
			cp := p.CustomMediaProviders[i]
			cp.Adapter = normalizeMediaAdapter(cp.Adapter)
			return &cp
		}
	}
	return nil
}

func customMediaProvidersToJSON(list []customMediaProvider) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, e := range list {
		out = append(out, map[string]any{
			"id":          e.ID,
			"label":       e.Label,
			"adapter":     normalizeMediaAdapter(e.Adapter),
			"api_key":     strings.TrimSpace(e.APIKey),
			"secret_key":  strings.TrimSpace(e.SecretKey),
			"base_url":    strings.TrimSpace(e.BaseURL),
			"image_model": strings.TrimSpace(e.ImageModel),
			"video_model": strings.TrimSpace(e.VideoModel),
		})
	}
	return out
}

func mediaProviderConfigured(entry *customMediaProvider) bool {
	if entry == nil {
		return false
	}
	switch normalizeMediaAdapter(entry.Adapter) {
	case mediaAdapterKling:
		return strings.TrimSpace(entry.APIKey) != "" && strings.TrimSpace(entry.SecretKey) != ""
	default:
		return strings.TrimSpace(entry.APIKey) != ""
	}
}

func requiredHintsForMediaAdapter(adapter string) []string {
	switch normalizeMediaAdapter(adapter) {
	case mediaAdapterOpenAI:
		return []string{"出图/视频: openai.api_key + base_url（可选）"}
	case mediaAdapterVolcengine:
		return []string{"出图/视频: volcengine.api_key（方舟）+ base_url（可选）"}
	case mediaAdapterKling:
		return []string{"视频: kling Access Key + Secret Key + base_url（可选）"}
	case mediaAdapterGemini:
		return []string{"出图/视频: gemini.api_key + base_url（可选）"}
	case mediaAdapterDashscope:
		return []string{"出图/视频: dashscope.api_key + base_url/region（可选）"}
	default:
		return []string{"媒体供应商: api_key"}
	}
}

func applyCustomMediaProvider(app *config.App, entry *customMediaProvider) {
	if app == nil || entry == nil || app.Stack == nil {
		return
	}
	entry.Adapter = normalizeMediaAdapter(entry.Adapter)
	if app.Stack.Image == nil {
		app.Stack.Image = map[string]any{}
	}
	if app.Stack.Video == nil {
		app.Stack.Video = map[string]any{}
	}
	if m := strings.TrimSpace(entry.ImageModel); m != "" {
		app.Stack.Image["model"] = m
	}
	if m := strings.TrimSpace(entry.VideoModel); m != "" {
		app.Stack.Video["model"] = m
	}

	switch entry.Adapter {
	case mediaAdapterOpenAI:
		app.Providers.OpenAI.APIKey = strings.TrimSpace(entry.APIKey)
		app.Providers.OpenAI.BaseURL = strings.TrimSpace(entry.BaseURL)
		app.Stack.Image["provider"] = "openai"
		app.Stack.Video["provider"] = "openai"
	case mediaAdapterVolcengine:
		app.Providers.Volcengine.APIKey = strings.TrimSpace(entry.APIKey)
		if u := strings.TrimSpace(entry.BaseURL); u != "" {
			app.Providers.Volcengine.BaseURL = u
		} else if app.Providers.Volcengine.BaseURL == "" {
			app.Providers.Volcengine.BaseURL = config.DefaultArkBaseURL
		}
		app.Stack.Image["provider"] = "volcengine"
		app.Stack.Video["provider"] = "volcengine"
	case mediaAdapterKling:
		app.Providers.Kling.AccessKey = strings.TrimSpace(entry.APIKey)
		app.Providers.Kling.SecretKey = strings.TrimSpace(entry.SecretKey)
		app.Providers.Kling.BaseURL = strings.TrimSpace(entry.BaseURL)
		app.Stack.Image["provider"] = "kling"
		app.Stack.Video["provider"] = "kling"
	case mediaAdapterGemini:
		app.Providers.Gemini.APIKey = strings.TrimSpace(entry.APIKey)
		app.Providers.Gemini.BaseURL = strings.TrimSpace(entry.BaseURL)
		app.Stack.Image["provider"] = "gemini"
		app.Stack.Video["provider"] = "gemini"
	case mediaAdapterDashscope:
		app.Providers.DashScope.APIKey = strings.TrimSpace(entry.APIKey)
		app.Providers.DashScope.BaseURL = strings.TrimSpace(entry.BaseURL)
		app.Stack.Image["provider"] = "dashscope"
		app.Stack.Video["provider"] = "dashscope"
	}
}

func activeMediaProviderLabel(p *userPrefs) string {
	if e := resolveActiveMediaProvider(p); e != nil && strings.TrimSpace(e.Label) != "" {
		return strings.TrimSpace(e.Label)
	}
	return ""
}

func sanitizeCustomMediaProviders(list []customMediaProvider) []customMediaProvider {
	out := make([]customMediaProvider, 0, len(list))
	seen := map[string]struct{}{}
	for _, e := range list {
		e.Adapter = normalizeMediaAdapter(e.Adapter)
		if strings.TrimSpace(e.Label) == "" {
			for _, info := range mediaAdapterCatalog {
				if info.ID == e.Adapter {
					e.Label = info.Label
					break
				}
			}
		}
		if strings.TrimSpace(e.ID) == "" {
			e.ID = uuid.NewString()
		}
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		out = append(out, e)
	}
	return out
}
