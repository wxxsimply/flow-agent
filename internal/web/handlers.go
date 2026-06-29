package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/auth"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/runner"
	"github.com/flow-agent/flow-agent/internal/wmreward"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

type createMicroMovieRequest struct {
	InputMode         string                    `json:"input_mode"`
	Plot              string                    `json:"plot"`
	Shots             []artifacts.UserShotInput `json:"shots"`
	Style             string                    `json:"style"`
	Orientation       string                    `json:"orientation"`
	Theme             string                    `json:"theme"`
	BGM               string                    `json:"bgm"`
	NarratorVoice     string                    `json:"narrator_voice"`
	TargetDurationSec int                       `json:"target_duration_sec"`
	DryRun            bool                      `json:"dry_run"`
	AutoGate          bool                      `json:"auto_gate"`
	StopAfterStage    string                    `json:"stop_after_stage"`
	StackProfile      string                    `json:"stack_profile"`
	OutputDir         string                    `json:"output_dir"`
}

type voiceOptionJSON struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	SpeedRatio  float64 `json:"speed_ratio"`
}

type createMicroMovieResponse struct {
	RunID  string `json:"run_id"`
	RunDir string `json:"run_dir"`
}

type runStatusResponse struct {
	RunID           string                    `json:"run_id"`
	RunDir          string                    `json:"run_dir,omitempty"`
	Stage           string                    `json:"stage"`
	Finished        bool                      `json:"finished"`
	Failed          bool                      `json:"failed"`
	AwaitingReview  bool                      `json:"awaiting_review"`
	Interrupted     bool                      `json:"interrupted,omitempty"`
	WorkflowActive  bool                      `json:"workflow_active,omitempty"`
	ResumeStage     string                    `json:"resume_stage,omitempty"`
	DryRun          bool                      `json:"dry_run"`
	StackProfile    string                    `json:"stack_profile,omitempty"`
	Artifacts       []artifacts.ArtifactEntry `json:"artifacts,omitempty"`
	MasterVideo     string                    `json:"master_video,omitempty"`
	BriefPath       string                    `json:"brief_path,omitempty"`
	StoryboardPath  string                    `json:"storyboard_path,omitempty"`
	ExpandPath      string                    `json:"expand_path,omitempty"`
	BriefRunesMin   int                       `json:"brief_runes_min,omitempty"`
	Error           string                    `json:"error,omitempty"`
}

type patchArtifactsRequest struct {
	Brief              string                        `json:"brief,omitempty"`
	Storyboard         *artifacts.Storyboard         `json:"storyboard,omitempty"`
	ShotLanguageExpand *artifacts.ShotLanguageExpand `json:"shot_language_expand,omitempty"`
}

type resumeRunRequest struct {
	FromStage string `json:"from_stage"`
	AutoGate  bool   `json:"auto_gate"`
}

type iterateRunRequest struct {
	Plot              string `json:"plot"`
	Style             string `json:"style"`
	Orientation       string `json:"orientation"`
	Theme             string `json:"theme"`
	NarratorVoice     string `json:"narrator_voice"`
	TargetDurationSec int    `json:"target_duration_sec"`
	StackProfile      string `json:"stack_profile"`
	StopAfterStage    string `json:"stop_after_stage"`
	AutoGate          bool   `json:"auto_gate"`
	DryRun              bool   `json:"dry_run"`
}

type configStatusResponse struct {
	Root                  string            `json:"root"`
	StackDefault          string            `json:"stack_default"`
	StackProfile          string            `json:"stack_profile"`
	MediaConfigured       bool              `json:"media_configured"`
	VolcengineConfigured  bool              `json:"volcengine_configured"`
	RequiredProviders     []string          `json:"required_providers,omitempty"`
	AvailableStacks       []stackSummaryJSON `json:"available_stacks,omitempty"`
	ProvidersFile         string            `json:"providers_file"`
	ProvidersExists       bool              `json:"providers_exists"`
	FFmpegAvailable       bool              `json:"ffmpeg_available"`
	WMRewardReady         bool              `json:"wmreward_ready"`
	DesktopMode           bool              `json:"desktop_mode"`
	DesktopWindowControls bool              `json:"desktop_window_controls"`
	ActiveMediaProviderLabel string            `json:"active_media_provider_label,omitempty"`
	AuthEnabled           bool              `json:"auth_enabled"`
}

type apiHandler struct {
	root         string
	desktopMode  bool
	authCfg      auth.Config
	authStore    *auth.Store
	authSvc      *auth.AuthService
	smsSendLimit *auth.RateLimiter
	smsLoginLimit *auth.RateLimiter
}

func (h *apiHandler) handleShotSizes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, artifacts.ListShotSizeOptions())
}

func (h *apiHandler) handleThemes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, config.ListVisualThemes())
}

func (h *apiHandler) handleVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var list []voiceOptionJSON
	for _, v := range config.ListVoicePresets() {
		list = append(list, voiceOptionJSON{
			ID: v.ID, Label: v.Label, Description: v.Description, SpeedRatio: v.SpeedRatio,
		})
	}
	writeJSON(w, list)
}

func (h *apiHandler) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	prefs, _ := loadUserPrefs(h.root)
	stackName := DefaultStudioStack
	if prefs != nil {
		stackName = effectiveStackProfile(prefs, "")
	}
	_, err := h.loadAppForStack(stackName)
	mediaOK := false
	volcConfigured := false
	var required []string
	if err == nil {
		merged := mergeProvidersFromPrefs(baseProvidersForRuntime(h.root, h.desktopMode), prefs)
		mediaOK = mediaReadyForPrefs(h.root, stackName, merged, prefs)
		if prefs != nil {
			volcConfigured = volcengineConfigured(prefs)
		}
		if entry := resolveActiveMediaProvider(prefs); entry != nil {
			required = requiredHintsForMediaAdapter(entry.Adapter)
		} else if stack, sErr := loadStackByName(h.root, stackName); sErr == nil {
			required = requiredProviderHints(stack)
		}
	}
	stacks, _ := listStudioStacks(h.root)
	activeLabel := ""
	if prefs != nil {
		activeLabel = activeMediaProviderLabel(prefs)
	}
	writeJSON(w, configStatusResponse{
		Root:                     h.root,
		StackDefault:             DefaultStudioStack,
		StackProfile:             stackName,
		MediaConfigured:          mediaOK,
		ProvidersFile:            filepath.Join(h.root, "config", "providers.local.yaml"),
		ProvidersExists:          fileExists(filepath.Join(h.root, "config", "providers.local.yaml")),
		FFmpegAvailable:          ffmpegAvailable(),
		WMRewardReady:            wmreward.Ready(h.root),
		DesktopMode:              h.desktopMode,
		DesktopWindowControls:    h.desktopMode,
		AuthEnabled:              h.authCfg.Enabled,
		VolcengineConfigured:     volcConfigured,
		RequiredProviders:        required,
		AvailableStacks:          stacks,
		ActiveMediaProviderLabel: activeLabel,
	})
}

func (h *apiHandler) handleCreateMicroMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req createMicroMovieRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.StackProfile == "" {
		prefs, _ := loadUserPrefs(h.root)
		req.StackProfile = effectiveStackProfile(prefs, "")
	}
	if req.BGM == "" {
		req.BGM = "auto"
	}
	if req.InputMode == "" {
		req.InputMode = "director"
	}
	if req.InputMode == "auto" {
		if req.Plot == "" && !req.DryRun {
			http.Error(w, "auto mode requires plot (enable dry-run to test)", http.StatusBadRequest)
			return
		}
	} else if req.Plot == "" && !req.DryRun {
		http.Error(w, "director mode requires opening shot text (第一镜)", http.StatusBadRequest)
		return
	}

	app, err := h.loadAppForStack(req.StackProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !req.DryRun {
		prefs, _ := loadUserPrefs(h.root)
		if hint := mediaProduceMissingHint(h.root, req.StackProfile, app.Providers, prefs); hint != "" {
			http.Error(w, hint, http.StatusBadRequest)
			return
		}
	}
	def, err := workflow.Load(filepath.Join(h.root, "docs", "workflows"), "micro-movie")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	store := runctx.NewStore(app.RunsDir)
	var rc *runctx.Context
	outputDir := strings.TrimSpace(req.OutputDir)
	title := runTitleFromPlot(req.Plot)
	if outputDir != "" {
		rc, err = store.CreateRunInWorkspace(outputDir, title, "micro-movie", 1, "micro-movie", req.StackProfile, req.DryRun)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ws, wsErr := runctx.WorkspaceFromPath(outputDir); wsErr == nil {
			persistWorkspacePref(h.root, ws)
		}
	} else {
		rc, err = store.CreateRun("micro-movie", 1, "micro-movie", req.StackProfile, req.DryRun)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if userID, err := h.currentUserID(r); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	} else if userID != "" {
		rc.Manifest.UserID = userID
	}
	rc.Manifest.Title = title
	_ = rc.SaveManifest()
	rc.App = app
	rc.Providers = provider.NewBundle(app)
	rc.Def = def
	rc.AutoGate = req.AutoGate
	rc.InitCostRecorder()
	rc.PlotInput = req.Plot
	rc.SeriesDir = filepath.Join(app.SeriesDir, "micro-movie")
	rc.Creative = &artifacts.CreativeOptions{
		InputMode:         req.InputMode,
		AnimationStyle:    req.Style,
		Orientation:       req.Orientation,
		VisualTheme:       req.Theme,
		Plot:              req.Plot,
		Shots:             req.Shots,
		BGMMode:           req.BGM,
		NarratorVoice:     req.NarratorVoice,
		TargetDurationSec: req.TargetDurationSec,
	}
	rc.Creative.Normalize()
	_ = agent.SaveCreativeOptions(rc)
	if req.Plot != "" {
		_ = rc.WriteArtifact("artifacts/plot-input.md", []byte(req.Plot))
	}

	go func() {
		unlock := lockRunWorkflow(rc.RunID)
		defer unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		stopAfter := strings.TrimSpace(req.StopAfterStage)
		rc.StopAfterStage = stopAfter
		if err := runner.RunWorkflow(ctx, rc, runner.Options{
			DryRun:         req.DryRun,
			AutoGate:       req.AutoGate,
			StopAfterStage: stopAfter,
		}); err != nil {
			slog.Error("web run failed", "run_id", rc.RunID, "err", err)
		}
	}()

	writeJSON(w, createMicroMovieResponse{RunID: rc.RunID, RunDir: rc.RunDir})
}

func (h *apiHandler) handleRuns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.Error(w, "missing run id", http.StatusBadRequest)
		return
	}

	parts := strings.Split(path, "/")
	runID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.writeRunStatus(w, r, runID)
		case http.MethodDelete:
			h.deleteRun(w, r, runID)
		default:
			methodNotAllowed(w)
		}
		return
	}

	switch parts[1] {
	case "open-folder":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.openRunFolder(w, r, runID)
	case "artifacts":
		switch r.Method {
		case http.MethodPatch:
			h.patchRunArtifacts(w, r, runID)
		default:
			methodNotAllowed(w)
		}
	case "resume":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.resumeRun(w, r, runID)
	case "iterate":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.iterateRun(w, r, runID)
	case "artifact":
		if r.Method != http.MethodGet || len(parts) < 3 {
			http.Error(w, "usage: /api/runs/{id}/artifact/{relative-path}", http.StatusBadRequest)
			return
		}
		rel := strings.Join(parts[2:], "/")
		h.serveRunArtifact(w, r, runID, rel)
	default:
		http.NotFound(w, r)
	}
}

func (h *apiHandler) deleteRun(w http.ResponseWriter, r *http.Request, runID string) {
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := runctx.NewStore(app.RunsDir)
	userID, err := h.currentUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := store.DeleteRun(runID, userID); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "manifest not found") {
			http.Error(w, msg, http.StatusNotFound)
			return
		}
		if strings.Contains(msg, "forbidden") || strings.Contains(msg, "not owned") {
			http.Error(w, msg, http.StatusForbidden)
			return
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "deleted", "run_id": runID})
}

func (h *apiHandler) writeRunStatus(w http.ResponseWriter, r *http.Request, runID string) {
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := runctx.NewStore(app.RunsDir)
	if err := h.assertRunAccess(r, store, runID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	rc, err := store.LoadRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	m := rc.Manifest
	resp := runStatusResponse{
		RunID:        m.RunID,
		RunDir:       rc.RunDir,
		Stage:        m.Stage,
		Finished:     m.Stage == "finished",
		AwaitingReview: m.Stage == "awaiting_review",
		DryRun:       m.DryRun,
		StackProfile: m.Stack,
		Artifacts:    m.Artifacts,
	}
	if m.Stage == "awaiting_review" || artifactExists(rc.RunDir, "artifacts/shot-language-brief.md") {
		resp.BriefPath = "/api/runs/" + runID + "/artifact/artifacts/shot-language-brief.md"
	}
	if artifactExists(rc.RunDir, "artifacts/shot-language-expand.json") {
		resp.ExpandPath = "/api/runs/" + runID + "/artifact/artifacts/shot-language-expand.json"
	}
	if artifactExists(rc.RunDir, "artifacts/storyboard.json") {
		resp.StoryboardPath = "/api/runs/" + runID + "/artifact/artifacts/storyboard.json"
	}
	stackName := m.Stack
	if stackName == "" {
		stackName = "micro-movie-cap5"
	}
	if stackApp, err := config.Load(h.root, stackName); err == nil {
		resp.BriefRunesMin = stackApp.Stack.AssembleConfig().BriefRunesMin
	}
	if resp.BriefRunesMin <= 0 {
		resp.BriefRunesMin = 2000
	}
	if artifactExists(rc.RunDir, "artifacts/master.mp4") {
		resp.MasterVideo = "/api/runs/" + runID + "/artifact/artifacts/master.mp4"
	}
	if m.Stage == "failed" {
		resp.Failed = true
	}
	if m.LastError != "" {
		resp.Error = m.LastError
	}
	active := IsRunWorkflowActive(runID)
	resp.WorkflowActive = active
	if isRunStaleStage(m.Stage) && !active {
		resp.Interrupted = true
		resp.ResumeStage = m.Stage
	}
	if m.Stage == "failed" && !active {
		resp.ResumeStage = inferResumeStage(rc)
	}
	writeJSON(w, resp)
}

func (h *apiHandler) openRunFolder(w http.ResponseWriter, r *http.Request, runID string) {
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := runctx.NewStore(app.RunsDir)
	if err := h.assertRunAccess(r, store, runID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	rc, err := store.LoadRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	runDir := rc.RunDir
	if err := OpenRunDir(runDir); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"run_dir": runDir})
}

func (h *apiHandler) serveRunArtifact(w http.ResponseWriter, r *http.Request, runID, rel string) {
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := runctx.NewStore(app.RunsDir)
	if err := h.assertRunAccess(r, store, runID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	rc, err := store.LoadRun(runID)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	abs, err := resolveArtifactAbsSafe(rc.RunDir, rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if !fileExists(abs) {
		candidate := resolveArtifactAbs(rc.RunDir, rel)
		if candidate == "" || !fileExists(candidate) {
			http.NotFound(w, nil)
			return
		}
		abs, err = assertAbsUnderRun(rc.RunDir, candidate)
		if err != nil {
			http.NotFound(w, nil)
			return
		}
	}
	http.ServeFile(w, r, abs)
}

func (h *apiHandler) patchRunArtifacts(w http.ResponseWriter, r *http.Request, runID string) {
	rc, err := h.loadRun(r, runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if rc.Manifest.Stage != "awaiting_review" && rc.Manifest.Stage != "assemble" {
		http.Error(w, "run is not in review state", http.StatusConflict)
		return
	}
	stackName := rc.Stack
	if stackName == "" {
		stackName = "micro-movie-wan-flash"
	}
	app, err := config.Load(h.root, stackName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rc.App = app

	var req patchArtifactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	var exp *artifacts.ShotLanguageExpand
	if req.ShotLanguageExpand != nil {
		exp = req.ShotLanguageExpand
	}
	briefBody := strings.TrimSpace(req.Brief)
	if exp != nil && briefBody != "" {
		exp.ShotLanguageBrief = agent.ExtractBriefBody(briefBody)
	} else if briefBody != "" && exp == nil {
		briefBody = agent.ExtractBriefBody(briefBody)
	}

	if exp == nil && req.Storyboard == nil && briefBody == "" {
		http.Error(w, "nothing to save", http.StatusBadRequest)
		return
	}
	var sb *artifacts.Storyboard
	if req.ShotLanguageExpand == nil {
		sb = req.Storyboard
	}
	if err := agent.SyncReviewArtifacts(rc, briefBody, exp, sb); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "saved"})
}

func (h *apiHandler) resumeRun(w http.ResponseWriter, r *http.Request, runID string) {
	rc, err := h.loadRun(r, runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	stage := rc.Manifest.Stage
	active := IsRunWorkflowActive(runID)
	canResume := stage == "awaiting_review" ||
		stage == "failed" ||
		(isRunStaleStage(stage) && !active)
	if !canResume {
		http.Error(w, "run cannot be resumed", http.StatusConflict)
		return
	}
	if active {
		http.Error(w, "run workflow already in progress", http.StatusConflict)
		return
	}
	unlock, ok := tryLockRunWorkflow(runID)
	if !ok {
		http.Error(w, "run workflow already in progress", http.StatusConflict)
		return
	}
	unlock()

	var req resumeRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	from := strings.TrimSpace(req.FromStage)
	if from == "" {
		switch stage {
		case "awaiting_review":
			from = "produce"
		case "failed":
			from = inferResumeStage(rc)
		default:
			from = stage
		}
	}
	rc.Manifest.LastError = ""
	stackName := rc.Stack
	if stackName == "" {
		stackName = DefaultStudioStack
	}
	app, err := h.loadAppForStack(stackName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	def, err := workflow.Load(filepath.Join(h.root, "docs", "workflows"), "micro-movie")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rc.App = app
	rc.InitCostRecorder()
	rc.Providers = provider.NewBundle(app)
	rc.Def = def
	rc.AutoGate = req.AutoGate
	agent.LoadCreativeOptionsFromRun(rc)
	rc.SetGate("brief_confirmed", true)
	if err := agent.PrepareResumeFromStage(rc, from, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = rc.SaveManifest()

	go func() {
		unlock := lockRunWorkflow(runID)
		defer unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		if err := runner.RunWorkflow(ctx, rc, runner.Options{
			FromStage: from,
			AutoGate:  req.AutoGate,
		}); err != nil {
			slog.Error("web resume failed", "run_id", runID, "err", err)
		}
	}()

	writeJSON(w, map[string]string{"status": "resumed", "from_stage": from})
}

func (h *apiHandler) iterateRun(w http.ResponseWriter, r *http.Request, runID string) {
	rc, err := h.loadRun(r, runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	stage := rc.Manifest.Stage
	if stage != "finished" && stage != "failed" {
		http.Error(w, "run must be finished or failed to iterate", http.StatusConflict)
		return
	}
	unlock, ok := tryLockRunWorkflow(runID)
	if !ok {
		http.Error(w, "run workflow already in progress", http.StatusConflict)
		return
	}
	unlock()

	var req iterateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	plot := strings.TrimSpace(req.Plot)
	if plot == "" && !req.DryRun {
		http.Error(w, "plot is required", http.StatusBadRequest)
		return
	}
	stackName := strings.TrimSpace(req.StackProfile)
	if stackName == "" {
		stackName = rc.Stack
	}
	if stackName == "" {
		stackName = DefaultStudioStack
	}
	app, err := h.loadAppForStack(stackName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	def, err := workflow.Load(filepath.Join(h.root, "docs", "workflows"), "micro-movie")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc.App = app
	rc.Providers = provider.NewBundle(app)
	rc.Def = def
	rc.Stack = stackName
	rc.AutoGate = req.AutoGate
	rc.InitCostRecorder()
	rc.PlotInput = plot
	if rc.Creative == nil {
		rc.Creative = &artifacts.CreativeOptions{}
	}
	if plot != "" {
		rc.Creative.Plot = plot
	}
	if req.Style != "" {
		rc.Creative.AnimationStyle = req.Style
	}
	if req.Orientation != "" {
		rc.Creative.Orientation = req.Orientation
	}
	if req.Theme != "" {
		rc.Creative.VisualTheme = req.Theme
	}
	if req.NarratorVoice != "" {
		rc.Creative.NarratorVoice = req.NarratorVoice
	}
	if req.TargetDurationSec > 0 {
		rc.Creative.TargetDurationSec = req.TargetDurationSec
	}
	rc.Creative.Normalize()
	_ = agent.SaveCreativeOptions(rc)
	if plot != "" {
		_ = rc.WriteArtifact("artifacts/plot-input.md", []byte(plot))
		rc.Manifest.Title = runTitleFromPlot(plot)
	}

	for k := range rc.Manifest.Gates {
		rc.Manifest.Gates[k] = false
	}
	rc.Manifest.LastError = ""
	rc.Manifest.Stage = "created"
	rc.Manifest.FinishedAt = nil

	from := "assemble"
	if err := agent.PrepareResumeFromStage(rc, from, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = rc.SaveManifest()

	stopAfter := strings.TrimSpace(req.StopAfterStage)
	if stopAfter == "" {
		stopAfter = "assemble"
	}

	go func() {
		unlock := lockRunWorkflow(runID)
		defer unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		rc.StopAfterStage = stopAfter
		if err := runner.RunWorkflow(ctx, rc, runner.Options{
			FromStage:      from,
			DryRun:         req.DryRun,
			AutoGate:       req.AutoGate,
			StopAfterStage: stopAfter,
		}); err != nil {
			slog.Error("web iterate failed", "run_id", runID, "err", err)
		}
	}()

	writeJSON(w, map[string]string{"status": "iterating", "from_stage": from})
}

func (h *apiHandler) loadRun(r *http.Request, runID string) (*runctx.Context, error) {
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		return nil, err
	}
	store := runctx.NewStore(app.RunsDir)
	if err := h.assertRunAccess(r, store, runID); err != nil {
		return nil, err
	}
	return store.LoadRun(runID)
}

func (h *apiHandler) assertRunAccess(r *http.Request, store *runctx.Store, runID string) error {
	userID, err := h.currentUserID(r)
	if err != nil {
		return err
	}
	return store.AssertRunOwner(runID, userID)
}

func runTitleFromPlot(plot string) string {
	plot = strings.TrimSpace(plot)
	if plot == "" {
		return "未命名项目"
	}
	r := []rune(plot)
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return plot
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func openFolder(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func ffmpegAvailable() bool {
	if p := os.Getenv("FFMPEG_PATH"); p != "" && fileExists(p) {
		return true
	}
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
