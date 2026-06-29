package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// AssembleStage 用户逐镜输入 → storyboard.json（director 模式）。
type AssembleStage struct{}

func (AssembleStage) ID() string { return "assemble" }

func (AssembleStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunShotAssembler(rc)
}
