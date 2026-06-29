package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// ExpandStage 模糊剧情 → story-spine.json。
type ExpandStage struct{}

func (ExpandStage) ID() string { return "expand" }

func (ExpandStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunPlotExpander(rc)
}
