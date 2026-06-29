package stage

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// PlanStage 规划本集目标、钩子与 brief。
type PlanStage struct{}

func (PlanStage) ID() string { return "plan" }

func (PlanStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	v := vault.ForSeries(rc.App, rc.SeriesID)
	if err := agent.RunPlanner(rc, v); err != nil {
		return err
	}

	if rc.AutoGate {
		rc.SetGate("outline_confirmed", true)
		return nil
	}

	briefPath := rc.ArtifactPath("artifacts/episode-brief.md")
	fmt.Fprintf(os.Stdout, "\n--- Plan 阶段完成 ---\n")
	fmt.Fprintf(os.Stdout, "episode brief: %s\n", briefPath)

	if plan, err := artifacts.LoadHookPlan(rc.ArtifactPath("artifacts/hook-plan.json")); err == nil {
		fmt.Fprintf(os.Stdout, "hook: %s | scenes: %d\n", plan.HookLine, len(plan.Scenes))
	}

	fmt.Fprint(os.Stdout, "确认大纲并继续? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "y" && line != "Y" {
		return fmt.Errorf("human gate outline_confirmed: not approved")
	}
	rc.SetGate("outline_confirmed", true)
	return nil
}
