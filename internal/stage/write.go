package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// WriteStage 流式（骨架为分场景）生成本章正文。
type WriteStage struct{}

func (WriteStage) ID() string { return "write" }

func (WriteStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunWriter(rc)
}
