package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// ComplyStage 敏感词与平台合规检查。
type ComplyStage struct{}

func (ComplyStage) ID() string { return "comply" }

func (ComplyStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunCompliance(rc)
}
