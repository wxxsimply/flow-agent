package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// PublishStage 生成 publish-pack（标题、话题、成片路径）。
type PublishStage struct{}

func (PublishStage) ID() string { return "publish" }

func (PublishStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunPublisher(rc)
}
