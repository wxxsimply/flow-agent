package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunContinuity 校验 chapter 与 bible/伏笔一致性，输出 continuity-report.json。
func RunContinuity(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.DryRun {
		return runContinuityDryRun(rc)
	}
	return runContinuityLive(rc, v)
}

func runContinuityDryRun(rc *runctx.Context) error {
	report := &artifacts.ContinuityReport{
		EpisodeNo:     rc.EpisodeNo,
		CriticalCount: 0,
		WarningCount:  0,
		Issues:        []artifacts.ContinuityIssue{},
		Passed:        true,
	}
	return saveContinuityArtifacts(rc, report, nil)
}

func runContinuityLive(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.Providers == nil {
		return fmt.Errorf("dashscope api_key required for continuity (or use --dry-run)")
	}
	if rc.App == nil || strings.TrimSpace(rc.App.Providers.DashScope.APIKey) == "" {
		return fmt.Errorf("dashscope api_key required for continuity (or use --dry-run)")
	}

	_ = v.EnsureCharacterStateFromBible()

	bible, err := v.LoadBible()
	if err != nil {
		return err
	}
	chapter, err := os.ReadFile(rc.ArtifactPath("artifacts/chapter.md"))
	if err != nil {
		return fmt.Errorf("read chapter: %w", err)
	}
	charState, err := v.LoadCharacterState()
	if err != nil {
		return err
	}
	charJSON, _ := json.MarshalIndent(charState, "", "  ")

	_ = v.SyncIndex()
	hits, _ := v.SearchFTS("伏笔", 5)
	var hitText strings.Builder
	for _, h := range hits {
		hitText.WriteString(fmt.Sprintf("- [%s ep%d] %s\n", h.Kind, h.EpisodeNo, h.Snippet))
	}

	ref := rc.App.LLMRef("continuity")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	res, err := completeJSONWithRetry(ctx, rc.Providers.LLMForStage(rc.App, "continuity"), llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.ContinuitySystem,
		User:        prompts.ContinuityUser(rc.EpisodeNo, bible, string(charJSON), hitText.String(), string(chapter)),
		MaxTokens:   4096,
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("continuity llm: %w", err)
	}
	rc.RecordLLM(res.Usage)

	report, patch, err := parseContinuityJSON(res.Text, rc.EpisodeNo)
	if err != nil {
		return err
	}
	report.NormalizeSeverity()
	patch = mergeCharacterPatch(patch, report)
	return saveContinuityArtifacts(rc, report, patch)
}

func parseContinuityJSON(raw string, episodeNo int) (*artifacts.ContinuityReport, map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 && idx < len(raw)-1 {
		raw = raw[:idx+1]
	}
	var payload struct {
		artifacts.ContinuityReport
		CharacterStatePatch map[string]any `json:"character_state_patch"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		snip := raw
		if len(snip) > 500 {
			snip = snip[:500] + "..."
		}
		return nil, nil, fmt.Errorf("parse continuity json: %w\nsnippet: %s", err, snip)
	}
	report := &payload.ContinuityReport
	if err := report.Validate(episodeNo); err != nil {
		return nil, nil, err
	}
	patch := payload.CharacterStatePatch
	if patch == nil {
		patch = report.CharacterPatch
	}
	return report, patch, nil
}

func saveContinuityArtifacts(rc *runctx.Context, report *artifacts.ContinuityReport, patch map[string]any) error {
	reportPath := rc.ArtifactPath("artifacts/continuity-report.json")
	if err := report.Save(reportPath); err != nil {
		return err
	}
	rc.RecordArtifact("continuity-report.json", "artifacts/continuity-report.json", true)
	rc.SetGate("continuity_passed", report.Passed)

	if len(patch) > 0 {
		patchBytes, err := json.MarshalIndent(patch, "", "  ")
		if err != nil {
			return err
		}
		if err := rc.WriteArtifact("artifacts/character-state.patch.json", patchBytes); err != nil {
			return err
		}
		rc.RecordArtifact("character-state.patch.json", "artifacts/character-state.patch.json", false)
	}
	return nil
}

// ApplyContinuityPatch 将 continuity 产出的 patch 写入系列 character-state.json。
func ApplyContinuityPatch(rc *runctx.Context, v *vault.SeriesVault) error {
	patchPath := rc.ArtifactPath("artifacts/character-state.patch.json")
	if !rc.ArtifactExists("artifacts/character-state.patch.json") {
		return nil
	}
	return v.ApplyCharacterPatchFile(patchPath)
}
