package web

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
)

func mergeStringField(dst *string, src string) {
	if v := strings.TrimSpace(src); v != "" {
		*dst = v
	}
}

func mergeProviderUserCredsVolcengine(dst *config.VolcengineConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
	mergeStringField(&dst.AccessKey, src.AccessKey)
	mergeStringField(&dst.SecretKey, src.SecretKey)
	mergeStringField(&dst.AppID, src.AppID)
}

func mergeProviderUserCredsDashScope(dst *config.DashScopeConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
	mergeStringField(&dst.Region, src.Region)
}

func mergeProviderUserCredsKling(dst *config.KlingConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.AccessKey, src.AccessKey)
	mergeStringField(&dst.SecretKey, src.SecretKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
}

func mergeProviderUserCredsGemini(dst *config.GeminiConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
}

func mergeProviderUserCredsOpenAI(dst *config.OpenAIConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
}

func mergeProviderUserCredsDeepSeek(dst *config.DeepSeekConfig, src *providerUserCreds) {
	if src == nil {
		return
	}
	mergeStringField(&dst.APIKey, src.APIKey)
	mergeStringField(&dst.BaseURL, src.BaseURL)
}

// mergeProvidersFromPrefs 将 user-prefs 中的密钥覆盖到 providers.local 加载结果。
func mergeProvidersFromPrefs(base config.Providers, prefs *userPrefs) config.Providers {
	if prefs == nil {
		if base.Volcengine.BaseURL == "" {
			base.Volcengine.BaseURL = config.DefaultArkBaseURL
		}
		return base
	}
	normalizeLoadedPrefs(prefs)
	mergeProviderUserCredsVolcengine(&base.Volcengine, prefs.Providers.Volcengine)
	mergeProviderUserCredsDashScope(&base.DashScope, prefs.Providers.DashScope)
	mergeProviderUserCredsKling(&base.Kling, prefs.Providers.Kling)
	mergeProviderUserCredsGemini(&base.Gemini, prefs.Providers.Gemini)
	mergeProviderUserCredsOpenAI(&base.OpenAI, prefs.Providers.OpenAI)
	mergeProviderUserCredsDeepSeek(&base.DeepSeek, prefs.Providers.DeepSeek)
	if base.Volcengine.BaseURL == "" {
		base.Volcengine.BaseURL = config.DefaultArkBaseURL
	}
	return base
}

func effectiveStackProfile(prefs *userPrefs, override string) string {
	if s := strings.TrimSpace(override); s != "" {
		return normalizeStudioStack(s)
	}
	if prefs != nil {
		if s := strings.TrimSpace(prefs.StackProfile); s != "" {
			return normalizeStudioStack(s)
		}
	}
	return DefaultStudioStack
}

func baseProvidersForRuntime(root string, desktopMode bool) config.Providers {
	if desktopMode {
		p := config.Providers{}
		config.ApplyProviderEnv(&p)
		return p
	}
	app, err := config.Load(root, "")
	if err != nil {
		return config.Providers{}
	}
	return app.Providers
}

func (h *apiHandler) loadAppForStack(stackName string) (*config.App, error) {
	prefs, err := loadUserPrefs(h.root)
	if err != nil {
		return nil, err
	}
	stackName = effectiveStackProfile(prefs, stackName)
	app, err := config.Load(h.root, stackName)
	if err != nil {
		return nil, err
	}
	app.Providers = mergeProvidersFromPrefs(baseProvidersForRuntime(h.root, h.desktopMode), prefs)
	if entry := resolveActiveMediaProvider(prefs); entry != nil {
		applyCustomMediaProvider(app, entry)
	}
	return app, nil
}

func loadStackByName(root, stackName string) (*config.Stack, error) {
	if strings.TrimSpace(stackName) == "" {
		stackName = DefaultStudioStack
	}
	return config.LoadStack(filepath.Join(root, "config", "stacks", stackName+".yaml"))
}

func mediaReadyForPrefs(root, stackName string, p config.Providers, prefs *userPrefs) bool {
	stack, err := loadStackByName(root, stackName)
	if err != nil {
		return p.VolcengineArkConfigured() || strings.TrimSpace(p.DashScope.APIKey) != ""
	}
	if entry := resolveActiveMediaProvider(prefs); entry != nil {
		if !mediaProviderConfigured(entry) {
			return false
		}
		app := &config.App{Providers: p, Stack: stack}
		applyCustomMediaProvider(app, entry)
		return app.Providers.MediaProduceConfigured(app.Stack)
	}
	return p.MediaProduceConfigured(stack)
}

func mediaReadyForStack(root, stackName string, p config.Providers) bool {
	return mediaReadyForPrefs(root, stackName, p, nil)
}

func requiredProviderHints(stack *config.Stack) []string {
	if stack == nil {
		return []string{"volcengine.api_key 或 dashscope.api_key"}
	}
	var hints []string
	img := strings.ToLower(strings.TrimSpace(stack.ImageConfig().Provider))
	vid := strings.ToLower(strings.TrimSpace(stack.VideoConfig().Provider))
	tts := strings.ToLower(strings.TrimSpace(stack.TTSConfig().Provider))

	addVolc := func(label string) {
		hints = append(hints, label+": volcengine.api_key（方舟）")
	}
	addDash := func(label string) {
		hints = append(hints, label+": dashscope.api_key")
	}
	addOpenAI := func(label string) {
		hints = append(hints, label+": openai.api_key")
	}
	addGemini := func(label string) {
		hints = append(hints, label+": gemini.api_key")
	}
	addKling := func(label string) {
		hints = append(hints, label+": kling.access_key + secret_key")
	}

	switch {
	case img == "volcengine" || img == "seedream" || img == "jimeng" || img == "ark":
		addVolc("出图")
	case img == "bailian" || img == "dashscope" || img == "wan":
		addDash("出图")
	}

	switch {
	case vid == "volcengine" || vid == "seedance" || vid == "jimeng" || vid == "ark":
		addVolc("视频")
	case vid == "openai" || vid == "sora":
		addOpenAI("视频")
		if img != "volcengine" && img != "seedream" {
			addDash("出图")
		}
	case vid == "gemini" || vid == "veo" || vid == "google":
		addGemini("视频")
		if img != "volcengine" && img != "seedream" {
			addVolc("出图")
		}
	case vid == "kling":
		addKling("视频")
	case vid == "bailian" || vid == "dashscope" || vid == "wan":
		addDash("视频")
	default:
		addDash("视频")
	}

	switch tts {
	case "volcengine":
		hints = append(hints, "旁白 TTS: volcengine.app_id + access_key")
	case "dashscope", "bailian", "qwen":
		hints = append(hints, "旁白 TTS: dashscope.api_key")
	}
	return hints
}

func mediaProduceMissingHint(root, stackName string, p config.Providers, prefs *userPrefs) string {
	stack, err := loadStackByName(root, stackName)
	if err != nil {
		return "无法加载 stack 配置"
	}
	if entry := resolveActiveMediaProvider(prefs); entry != nil {
		app := &config.App{Providers: p, Stack: stack}
		applyCustomMediaProvider(app, entry)
		if app.Providers.MediaProduceConfigured(app.Stack) {
			return ""
		}
		hints := requiredHintsForMediaAdapter(entry.Adapter)
		return fmt.Sprintf("当前媒体供应商「%s」需要在设置中配置：%s", entry.Label, strings.Join(hints, "；"))
	}
	if p.MediaProduceConfigured(stack) {
		return ""
	}
	hints := requiredProviderHints(stack)
	if len(hints) == 0 {
		return fmt.Sprintf("方案 %s 缺少媒体 API 密钥，请在设置中配置", stackName)
	}
	return fmt.Sprintf("方案 %s 需要在设置中配置：%s", stack.Name, strings.Join(hints, "；"))
}
