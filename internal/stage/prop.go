package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// PropStage 为微电影生成物体三视图设定。
type PropStage struct{}

func (PropStage) ID() string { return "prop" }

func (PropStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunPropDesignerFromArtifacts(rc)
}
