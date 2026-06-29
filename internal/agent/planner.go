package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunPlanner 生成本集 brief 与 hook-plan（Plan 阶段产物）。
func RunPlanner(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.DryRun {
		return runPlannerDryRun(rc, v)
	}
	return runPlannerLive(rc, v)
}

func runPlannerDryRun(rc *runctx.Context, v *vault.SeriesVault) error {
	bible, _ := v.LoadBible()
	_, pubMetrics, nextHints := plannerInjectContext(v, rc.EpisodeNo)
	hook := fmt.Sprintf("第 %d 集钩子：主角遭遇反转，留下悬念。", rc.EpisodeNo)
	hook = "[dry-run] " + hook

	brief := fmt.Sprintf(`# Episode %d Brief

## Goal
推进主线冲突，结尾留 cliffhanger。

## Length
目标时长约 %d 秒旁白（约 750–1000 字）。

## Series
%s
`, rc.EpisodeNo, rc.TargetDurationSec(), truncate(bible, 500))
	if pubMetrics != "" {
		brief += "\n## 上集发布指标\n" + pubMetrics + "\n"
	}
	if nextHints != "" {
		brief += "\n## 下集策划提示\n" + truncate(nextHints, 600) + "\n"
	}

	hookPlan := fmt.Sprintf(`{
  "episode_no": %d,
  "hook_type": "cliffhanger",
  "hook_line": %q,
  "scene_count": 4,
  "scenes": [
    {"id": 1, "title": "雨夜来电", "goal": "接到改变命运的电话"},
    {"id": 2, "title": "回忆", "goal": "三年前误会真相渐明"},
    {"id": 3, "title": "天台", "goal": "他在等她，手里握着未寄出的信"},
    {"id": 4, "title": "离去", "goal": "她转身离去，复仇下集开始"}
  ]
}`, rc.EpisodeNo, hook)

	return writePlannerArtifacts(rc, []byte(brief), []byte(hookPlan))
}

func runPlannerLive(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.Providers == nil {
		return fmt.Errorf("deepseek api_key required (or use --dry-run)")
	}
	if rc.App == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return fmt.Errorf("deepseek api_key required (or use --dry-run)")
	}

	_ = v.EnsureCharacterStateFromBible()

	bible, err := v.LoadBible()
	if err != nil {
		return err
	}
	prev, pubMetrics, nextHints := plannerInjectContext(v, rc.EpisodeNo)
	targetSec := rc.TargetDurationSec()

	ref := rc.App.LLMRef("planner")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	userPrompt := prompts.PlannerUser(rc.SeriesID, rc.EpisodeNo, targetSec, bible, prev, pubMetrics, nextHints)
	req := llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.PlannerSystem,
		User:        userPrompt,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
	client := rc.Providers.LLMForStage(rc.App, "planner")

	var out *artifacts.PlannerOutput
	var lastParse error
	for attempt := 0; attempt < maxLLMAttempts; attempt++ {
		if attempt > 0 {
			slog.Warn("planner parse retry", "attempt", attempt+1, "err", lastParse)
		}
		res, err := completeJSONWithRetry(ctx, client, req)
		if err != nil {
			return fmt.Errorf("planner llm: %w", err)
		}
		rc.RecordLLM(res.Usage)
		out, lastParse = parsePlannerJSON(res.Text)
		if lastParse == nil {
			break
		}
		if attempt == maxLLMAttempts-1 {
			return lastParse
		}
	}
	if out == nil {
		return fmt.Errorf("planner: no output")
	}
	out.HookPlan.EpisodeNo = rc.EpisodeNo
	if err := out.HookPlan.Validate(rc.EpisodeNo); err != nil {
		return err
	}

	hookBytes, err := json.MarshalIndent(out.HookPlan, "", "  ")
	if err != nil {
		return err
	}
	return writePlannerArtifacts(rc, []byte(out.BriefMD), hookBytes)
}

func parsePlannerJSON(raw string) (*artifacts.PlannerOutput, error) {
	jsonStr := ExtractTopLevelJSON(raw)
	var out artifacts.PlannerOutput
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		snip := jsonStr
		if len(snip) > 500 {
			snip = snip[:500] + "..."
		}
		return nil, fmt.Errorf("parse planner json: %w\nmodel output snippet: %s", err, snip)
	}
	if strings.TrimSpace(out.BriefMD) == "" {
		return nil, fmt.Errorf("planner json: brief_md is empty")
	}
	return &out, nil
}

func writePlannerArtifacts(rc *runctx.Context, brief, hookPlan []byte) error {
	if err := rc.WriteArtifact("artifacts/episode-brief.md", brief); err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/hook-plan.json", hookPlan); err != nil {
		return err
	}
	rc.RecordArtifact("episode-brief.md", "artifacts/episode-brief.md", true)
	rc.RecordArtifact("hook-plan.json", "artifacts/hook-plan.json", true)
	return nil
}

func modelOrDefault(ref config.LLMRef, def string) string {
	if ref.Model != "" {
		return ref.Model
	}
	return def
}

// truncate 截断字符串，避免 brief 过长。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
