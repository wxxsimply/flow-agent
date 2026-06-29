package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/stage"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// runContinuityStage 执行 continuity 阶段，critical 时回退重写场景（受 max_continuity_retries 限制）。
func runContinuityStage(ctx context.Context, rc *runctx.Context, reg *stage.Registry, st *workflow.StageDefinition, logger *slog.Logger) error {
	maxRetries := maxContinuityRetries(rc)
	v := vault.ForSeries(rc.App, rc.SeriesID)

	handler, ok := reg.Get(st.ID)
	if !ok {
		return fmt.Errorf("no stage handler for %q", st.ID)
	}

	rc.Stage = st.ID
	rc.Manifest.Stage = st.ID
	if err := rc.SaveManifest(); err != nil {
		return err
	}

	stageHooks := workflow.ParseStageHooks(st.Hooks)
	if err := RunHooks(ctx, rc, "before", st.ID, stageHooks); err != nil {
		return err
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		logger.Info("stage start", "stage", st.ID, "agent", st.Agent, "continuity_attempt", attempt)

		if err := handler.Run(ctx, rc, st); err != nil {
			return fmt.Errorf("stage %q: %w", st.ID, err)
		}
		if err := EnsureRequiredArtifacts(rc, st); err != nil {
			return err
		}

		report, err := artifacts.LoadContinuityReport(rc.ArtifactPath("artifacts/continuity-report.json"))
		if err != nil {
			return err
		}

		if err := agent.ApplyContinuityPatch(rc, v); err != nil {
			return fmt.Errorf("apply character patch: %w", err)
		}

		if report.CriticalCount == 0 {
			if err := RunHooks(ctx, rc, "after", st.ID, stageHooks); err != nil {
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
			logger.Info("stage done", "stage", st.ID)
			return nil
		}

		if attempt >= maxRetries {
			report.NormalizeSeverity()
			if report.CriticalCount > 0 && rc.AutoGate {
				logger.Warn("continuity auto-gate: softening remaining criticals to warnings", "count", report.CriticalCount)
				report.SoftenRemainingCriticals()
				_ = report.Save(rc.ArtifactPath("artifacts/continuity-report.json"))
				rc.SetGate("continuity_passed", true)
				if err := CheckGates(rc, st); err != nil {
					return err
				}
				if err := rc.SaveManifest(); err != nil {
					return err
				}
				logger.Info("stage done", "stage", st.ID, "auto_gate_soften", true)
				return nil
			}
			return fmt.Errorf("continuity: %d critical issue(s) after %d rewrite(s)", report.CriticalCount, maxRetries)
		}

		plan, err := artifacts.LoadHookPlan(rc.ArtifactPath("artifacts/hook-plan.json"))
		if err != nil {
			return err
		}
		logger.Info("continuity retry", "attempt", attempt+1, "critical", report.CriticalCount, "scenes", report.SceneIDsForRewrite(sceneIDSet(plan)))
		var rewriteErr error
		// 第 2 次起整章重写，避免局部修改前后不一致
		if attempt >= 1 {
			logger.Info("continuity full rewrite", "attempt", attempt+1)
			rewriteErr = agent.FullRewriteAfterContinuity(rc, report, plan)
		} else {
			rewriteErr = agent.RewriteScenesAfterContinuity(rc, report, plan)
		}
		if rewriteErr != nil {
			return fmt.Errorf("rewrite scenes: %w", rewriteErr)
		}
		_ = os.Remove(rc.ArtifactPath("artifacts/continuity-report.json"))
		_ = os.Remove(rc.ArtifactPath("artifacts/character-state.patch.json"))
	}
	return nil
}

func maxContinuityRetries(rc *runctx.Context) int {
	if rc.Def == nil || rc.Def.Budget == nil {
		return 3
	}
	switch v := rc.Def.Budget["max_continuity_retries"].(type) {
	case int:
		if v < 1 {
			return 3
		}
		return v
	case float64:
		if int(v) < 1 {
			return 3
		}
		return int(v)
	default:
		return 3
	}
}

func sceneIDSet(plan *artifacts.HookPlan) map[int]bool {
	m := map[int]bool{}
	for _, s := range plan.Scenes {
		m[s.ID] = true
	}
	return m
}
