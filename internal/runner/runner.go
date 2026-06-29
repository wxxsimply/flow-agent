// Package runner 编排工作流各阶段的顺序执行、产物与门禁校验。
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/cost"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/stage"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// Options 控制 RunWorkflow 行为。
type Options struct {
	FromStage      string // 非空时从该阶段开始（resume）
	StopAfterStage string // 非空时在该阶段完成后暂停（awaiting_review）
	DryRun         bool
	AutoGate       bool
}

// RunWorkflow 按 YAML 顺序执行各 stage，直至完成或出错。
func RunWorkflow(ctx context.Context, rc *runctx.Context, opts Options) error {
	rc.DryRun = opts.DryRun
	rc.AutoGate = opts.AutoGate
	rc.StopAfterStage = opts.StopAfterStage
	if rc.Def == nil {
		return fmt.Errorf("workflow definition not loaded")
	}

	reg := stage.DefaultRegistry()
	started := opts.FromStage == "" // resume 时先跳过阶段
	logger := slog.Default().With("run_id", rc.RunID, "series", rc.SeriesID, "episode", rc.EpisodeNo)

	for _, st := range rc.Def.Stages {
		if !started {
			if st.ID == opts.FromStage {
				started = true
			} else {
				continue
			}
		}
		if st.ID == "continuity" {
			if err := runContinuityStage(ctx, rc, reg, &st, logger); err != nil {
				return err
			}
			continue
		}
		if shouldSkipStage(rc, st.ID) {
			logger.Info("stage skipped", "stage", st.ID, "reason", "input_mode")
			continue
		}
		if err := runOneStage(ctx, rc, reg, &st, logger); err != nil {
			if rc.Manifest != nil {
				rc.Manifest.LastError = err.Error()
				rc.Manifest.Stage = "failed"
				_ = rc.SaveManifest()
			}
			return err
		}
		if opts.StopAfterStage != "" && st.ID == opts.StopAfterStage {
			rc.Manifest.Stage = "awaiting_review"
			if err := rc.SaveManifest(); err != nil {
				return err
			}
			logger.Info("workflow paused for review", "after_stage", st.ID)
			return nil
		}
	}

	now := time.Now().UTC()
	rc.Manifest.FinishedAt = &now
	rc.Manifest.Stage = "finished"
	if err := rc.SaveManifest(); err != nil {
		return err
	}
	if err := rc.SaveCostLedger(); err != nil {
		return err
	}
	logCostBudget(logger, rc)
	logger.Info("workflow finished", "run_dir", rc.RunDir)
	return nil
}

func logCostBudget(logger *slog.Logger, rc *runctx.Context) {
	if rc.Manifest == nil || rc.Manifest.Cost == nil || rc.App == nil || rc.App.Stack == nil {
		return
	}
	checks := cost.CompareTargets(rc.Manifest.Cost, rc.App.Stack.CostTargetsCNY)
	for _, c := range checks {
		if !c.HasTarget || c.InRange {
			continue
		}
		logger.Warn("cost out of target range",
			"category", c.Category,
			"actual_cny", c.Actual,
			"min_cny", c.Min,
			"max_cny", c.Max)
	}
	if rc.Manifest.Cost.TotalCNY > 0 {
		logger.Info("cost summary",
			"total_cny", rc.Manifest.Cost.TotalCNY,
			"llm_cny", rc.Manifest.Cost.LLMCNY,
			"tts_cny", rc.Manifest.Cost.TTSCNY,
			"image_cny", rc.Manifest.Cost.ImageCNY,
			"video_cny", rc.Manifest.Cost.VideoCNY)
	}
}

// runOneStage 执行单个阶段：Handler → 产物检查 → 门禁检查。
func runOneStage(ctx context.Context, rc *runctx.Context, reg *stage.Registry, st *workflow.StageDefinition, logger *slog.Logger) error {
	for _, dep := range st.DependsOn {
		_ = dep // MVP：依赖顺序由 YAML 排列保证
	}

	rc.Stage = st.ID
	rc.Manifest.Stage = st.ID
	if err := rc.SaveManifest(); err != nil {
		return err
	}

	logger.Info("stage start", "stage", st.ID, "agent", st.Agent)

	stageHooks := workflow.ParseStageHooks(st.Hooks)
	if err := RunHooks(ctx, rc, "before", st.ID, stageHooks); err != nil {
		return err
	}

	handler, ok := reg.Get(st.ID)
	if !ok {
		return fmt.Errorf("no stage handler for %q", st.ID)
	}

	if err := handler.Run(ctx, rc, st); err != nil {
		return fmt.Errorf("stage %q: %w", st.ID, err)
	}
	if err := RunHooks(ctx, rc, "after", st.ID, stageHooks); err != nil {
		return err
	}

	if err := EnsureRequiredArtifacts(rc, st); err != nil {
		return err
	}
	if err := PromptHumanGates(rc, st); err != nil {
		return err
	}
	if err := CheckGates(rc, st); err != nil {
		return err
	}

	rc.SyncCost()
	if err := rc.SaveManifest(); err != nil {
		return err
	}
	if err := rc.SaveCostLedger(); err != nil {
		return err
	}
	logger.Info("stage done", "stage", st.ID, "cost_cny", stageCostTotal(rc))
	return nil
}

func stageCostTotal(rc *runctx.Context) float64 {
	if rc.Manifest == nil || rc.Manifest.Cost == nil {
		return 0
	}
	return rc.Manifest.Cost.TotalCNY
}

func shouldSkipStage(rc *runctx.Context, stageID string) bool {
	if rc.Workflow != "micro-movie" {
		return false
	}
	agent.LoadCreativeOptionsFromRun(rc)
	director := rc.Creative != nil && rc.Creative.IsDirector()
	if director {
		switch stageID {
		case "expand", "script", "storyboard", "character", "prop":
			return true
		}
		return false
	}
	// auto 模式跳过 assemble
	return stageID == "assemble"
}
