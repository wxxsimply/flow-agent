package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

type providerUserCreds struct {
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	AppID     string `json:"app_id,omitempty"`
	Region    string `json:"region,omitempty"`
}

type providersUserCreds struct {
	Volcengine *providerUserCreds `json:"volcengine,omitempty"`
	DashScope  *providerUserCreds `json:"dashscope,omitempty"`
	Kling      *providerUserCreds `json:"kling,omitempty"`
	Gemini     *providerUserCreds `json:"gemini,omitempty"`
	OpenAI     *providerUserCreds `json:"openai,omitempty"`
	DeepSeek   *providerUserCreds `json:"deepseek,omitempty"`
}

// volcengineUserCreds 向后兼容旧 prefs 格式。
type volcengineUserCreds struct {
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	AppID     string `json:"app_id,omitempty"`
}

type userPrefs struct {
	WorkspaceDir            string                `json:"workspace_dir"`
	DefaultOutputDir        string                `json:"default_output_dir"`
	StackProfile            string                `json:"stack_profile,omitempty"`
	Providers               providersUserCreds    `json:"providers,omitempty"`
	Volcengine              volcengineUserCreds   `json:"volcengine,omitempty"`
	CustomMediaProviders    []customMediaProvider `json:"custom_media_providers,omitempty"`
	ActiveMediaProviderID   string                `json:"active_media_provider_id,omitempty"`
}

func effectiveWorkspaceDir(p *userPrefs) string {
	if p == nil {
		return ""
	}
	if ws := strings.TrimSpace(p.WorkspaceDir); ws != "" {
		return ws
	}
	legacy := strings.TrimSpace(p.DefaultOutputDir)
	if legacy == "" {
		return ""
	}
	if ws, err := runctx.WorkspaceFromPath(legacy); err == nil {
		return ws
	}
	return legacy
}

func legacyVolcengineHasData(v volcengineUserCreds) bool {
	return strings.TrimSpace(v.APIKey) != "" ||
		strings.TrimSpace(v.AccessKey) != "" ||
		strings.TrimSpace(v.SecretKey) != "" ||
		strings.TrimSpace(v.AppID) != "" ||
		strings.TrimSpace(v.BaseURL) != ""
}

func normalizeLoadedPrefs(p *userPrefs) {
	if p == nil {
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
	if strings.TrimSpace(p.StackProfile) == "" {
		p.StackProfile = DefaultStudioStack
	} else {
		p.StackProfile = normalizeStudioStack(p.StackProfile)
	}
	migrateLegacyMediaProviders(p)
}

func providerCredsToMap(c *providerUserCreds) map[string]string {
	if c == nil {
		return map[string]string{}
	}
	return map[string]string{
		"api_key":    strings.TrimSpace(c.APIKey),
		"base_url":   strings.TrimSpace(c.BaseURL),
		"access_key": strings.TrimSpace(c.AccessKey),
		"secret_key": strings.TrimSpace(c.SecretKey),
		"app_id":     strings.TrimSpace(c.AppID),
		"region":     strings.TrimSpace(c.Region),
	}
}

func providersToJSONMap(p providersUserCreds) map[string]any {
	return map[string]any{
		"volcengine": providerCredsToMap(p.Volcengine),
		"dashscope":  providerCredsToMap(p.DashScope),
		"kling":      providerCredsToMap(p.Kling),
		"gemini":     providerCredsToMap(p.Gemini),
		"openai":     providerCredsToMap(p.OpenAI),
		"deepseek":   providerCredsToMap(p.DeepSeek),
	}
}

func volcengineConfigured(p *userPrefs) bool {
	normalizeLoadedPrefs(p)
	v := p.Providers.Volcengine
	if v == nil {
		return false
	}
	return strings.TrimSpace(v.APIKey) != "" && strings.TrimSpace(v.AccessKey) != ""
}

func prefsContainsSecrets(p *userPrefs) bool {
	if p == nil {
		return false
	}
	check := func(c *providerUserCreds) bool {
		if c == nil {
			return false
		}
		return strings.TrimSpace(c.APIKey) != "" ||
			strings.TrimSpace(c.AccessKey) != "" ||
			strings.TrimSpace(c.SecretKey) != ""
	}
	if check(p.Providers.Volcengine) || check(p.Providers.DashScope) ||
		check(p.Providers.Kling) || check(p.Providers.Gemini) ||
		check(p.Providers.OpenAI) || check(p.Providers.DeepSeek) {
		return true
	}
	for _, e := range p.CustomMediaProviders {
		if strings.TrimSpace(e.APIKey) != "" || strings.TrimSpace(e.SecretKey) != "" {
			return true
		}
	}
	return legacyVolcengineHasData(p.Volcengine)
}

func mergeProviderCredsFromBody(dst **providerUserCreds, src *providerUserCreds) {
	if src == nil {
		return
	}
	if !providerBodyHasData(src) {
		return
	}
	if *dst == nil {
		*dst = &providerUserCreds{}
	}
	if v := strings.TrimSpace(src.APIKey); v != "" {
		(*dst).APIKey = v
	}
	if v := strings.TrimSpace(src.BaseURL); v != "" {
		(*dst).BaseURL = v
	}
	if v := strings.TrimSpace(src.AccessKey); v != "" {
		(*dst).AccessKey = v
	}
	if v := strings.TrimSpace(src.SecretKey); v != "" {
		(*dst).SecretKey = v
	}
	if v := strings.TrimSpace(src.AppID); v != "" {
		(*dst).AppID = v
	}
	if v := strings.TrimSpace(src.Region); v != "" {
		(*dst).Region = v
	}
}

func providerBodyHasData(c *providerUserCreds) bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.APIKey) != "" ||
		strings.TrimSpace(c.BaseURL) != "" ||
		strings.TrimSpace(c.AccessKey) != "" ||
		strings.TrimSpace(c.SecretKey) != "" ||
		strings.TrimSpace(c.AppID) != "" ||
		strings.TrimSpace(c.Region) != ""
}

func legacyToProvider(c volcengineUserCreds) *providerUserCreds {
	if !legacyVolcengineHasData(c) {
		return nil
	}
	return &providerUserCreds{
		APIKey:    c.APIKey,
		BaseURL:   c.BaseURL,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
		AppID:     c.AppID,
	}
}

func userPrefsPath(root string) (string, error) {
	app, err := config.Load(root, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(app.DataDir, "user-prefs.json"), nil
}

func loadUserPrefs(root string) (*userPrefs, error) {
	path, err := userPrefsPath(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &userPrefs{StackProfile: DefaultStudioStack}, nil
		}
		return nil, err
	}
	var p userPrefs
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	normalizeLoadedPrefs(&p)
	return &p, nil
}

func saveUserPrefs(root string, p *userPrefs) error {
	path, err := userPrefsPath(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	normalizeLoadedPrefs(p)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if prefsContainsSecrets(p) {
		mode = 0o600
	}
	return os.WriteFile(path, data, mode)
}

func userPrefsResponse(root string, desktopMode bool, p *userPrefs) map[string]any {
	normalizeLoadedPrefs(p)
	stackName := effectiveStackProfile(p, "")
	merged := mergeProvidersFromPrefs(baseProvidersForRuntime(root, desktopMode), p)
	mediaOK := mediaReadyForPrefs(root, stackName, merged, p)
	v := p.Providers.Volcengine
	legacyVolc := providerCredsToMap(v)
	active := resolveActiveMediaProvider(p)
	var required []string
	if active != nil {
		required = requiredHintsForMediaAdapter(active.Adapter)
	} else if stack, err := loadStackByName(root, stackName); err == nil {
		required = requiredProviderHints(stack)
	}
	return map[string]any{
		"workspace_dir":              effectiveWorkspaceDir(p),
		"default_output_dir":         effectiveWorkspaceDir(p),
		"stack_profile":              stackName,
		"providers":                  providersToJSONMap(p.Providers),
		"volcengine":                 legacyVolc,
		"volcengine_configured":      volcengineConfigured(p),
		"media_configured":           mediaOK,
		"custom_media_providers":     customMediaProvidersToJSON(p.CustomMediaProviders),
		"active_media_provider_id":   strings.TrimSpace(p.ActiveMediaProviderID),
		"active_media_provider_label": activeMediaProviderLabel(p),
		"required_providers":         required,
	}
}

func (h *apiHandler) handleUserPrefs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p, err := loadUserPrefs(h.root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, userPrefsResponse(h.root, h.desktopMode, p))
	case http.MethodPut:
		var body struct {
			WorkspaceDir          string                `json:"workspace_dir"`
			DefaultOutputDir      string                `json:"default_output_dir"`
			StackProfile          string                `json:"stack_profile"`
			Providers             providersUserCreds    `json:"providers"`
			Volcengine            volcengineUserCreds     `json:"volcengine"`
			CustomMediaProviders  []customMediaProvider `json:"custom_media_providers"`
			ActiveMediaProviderID string                `json:"active_media_provider_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		existing, _ := loadUserPrefs(h.root)
		if existing == nil {
			existing = &userPrefs{StackProfile: DefaultStudioStack}
		}
		ws := strings.TrimSpace(body.WorkspaceDir)
		if ws == "" {
			ws = strings.TrimSpace(body.DefaultOutputDir)
		}
		if ws == "" {
			ws = effectiveWorkspaceDir(existing)
		}
		if ws != "" {
			if _, err := runctx.ValidateWorkspaceDir(ws); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		toSave := &userPrefs{
			WorkspaceDir:          ws,
			StackProfile:          existing.StackProfile,
			Providers:             existing.Providers,
			Volcengine:            existing.Volcengine,
			CustomMediaProviders:  existing.CustomMediaProviders,
			ActiveMediaProviderID: existing.ActiveMediaProviderID,
		}
		if sp := strings.TrimSpace(body.StackProfile); sp != "" {
			toSave.StackProfile = sp
		}
		if body.CustomMediaProviders != nil {
			toSave.CustomMediaProviders = sanitizeCustomMediaProviders(body.CustomMediaProviders)
		}
		if id := strings.TrimSpace(body.ActiveMediaProviderID); id != "" {
			toSave.ActiveMediaProviderID = id
		}
		mergeProviderCredsFromBody(&toSave.Providers.Volcengine, body.Providers.Volcengine)
		mergeProviderCredsFromBody(&toSave.Providers.DashScope, body.Providers.DashScope)
		mergeProviderCredsFromBody(&toSave.Providers.Kling, body.Providers.Kling)
		mergeProviderCredsFromBody(&toSave.Providers.Gemini, body.Providers.Gemini)
		mergeProviderCredsFromBody(&toSave.Providers.OpenAI, body.Providers.OpenAI)
		mergeProviderCredsFromBody(&toSave.Providers.DeepSeek, body.Providers.DeepSeek)
		if legacy := legacyToProvider(body.Volcengine); legacy != nil {
			if toSave.Providers.Volcengine == nil {
				toSave.Providers.Volcengine = &providerUserCreds{}
			}
			mergeProviderCredsFromBody(&toSave.Providers.Volcengine, legacy)
		}
		if toSave.Providers.Volcengine != nil {
			toSave.Volcengine = volcengineUserCreds{
				APIKey:    toSave.Providers.Volcengine.APIKey,
				BaseURL:   toSave.Providers.Volcengine.BaseURL,
				AccessKey: toSave.Providers.Volcengine.AccessKey,
				SecretKey: toSave.Providers.Volcengine.SecretKey,
				AppID:     toSave.Providers.Volcengine.AppID,
			}
		}
		if err := saveUserPrefs(h.root, toSave); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, userPrefsResponse(h.root, h.desktopMode, toSave))
	default:
		methodNotAllowed(w)
	}
}

func (h *apiHandler) handleMediaAdapters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, map[string]any{"adapters": mediaAdapterCatalog})
}

func (h *apiHandler) handleConfigStacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	stacks, err := listStudioStacks(h.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"stacks": stacks})
}

func (h *apiHandler) handleValidateDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	mode := strings.TrimSpace(strings.ToLower(req.Mode))
	if mode == "" {
		mode = "workspace"
	}
	var abs string
	var err error
	switch mode {
	case "project":
		abs, err = runctx.ValidateOutputDir(req.Path, false)
	default:
		abs, err = runctx.ValidateWorkspaceDir(req.Path)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"path": abs, "valid": "true", "mode": mode})
}

func (h *apiHandler) handleOpenDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.desktopMode {
		http.Error(w, "open directory only available in desktop mode", http.StatusNotFound)
		return
	}
	var req struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	kind := strings.TrimSpace(strings.ToLower(req.Kind))
	if kind == "" {
		kind = "workspace"
	}
	var abs string
	var err error
	switch kind {
	case "project":
		abs, err = runctx.ValidateOutputDir(req.Path, true)
		if err == nil && !fileExists(filepath.Join(abs, "manifest.json")) {
			err = fmt.Errorf("not a project directory")
		}
	default:
		abs, err = runctx.ValidateWorkspaceDir(req.Path)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := openFolder(abs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"path": abs})
}

func (h *apiHandler) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.desktopMode {
		http.Error(w, "folder picker only available in desktop mode", http.StatusNotFound)
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "选择视频工作区目录"
	}
	path, err := pickFolderDialog(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if path == "" {
		writeJSON(w, map[string]string{"cancelled": "true"})
		return
	}
	abs, err := runctx.ValidateWorkspaceDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"path": abs})
}

func persistWorkspacePref(root, workspace string) {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		return
	}
	if abs, err := runctx.ValidateWorkspaceDir(ws); err == nil {
		ws = abs
	} else if abs, err := runctx.WorkspaceFromPath(ws); err == nil {
		ws = abs
	}
	existing, _ := loadUserPrefs(root)
	if existing == nil {
		existing = &userPrefs{StackProfile: DefaultStudioStack}
	}
	existing.WorkspaceDir = ws
	_ = saveUserPrefs(root, existing)
}
