package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// LearnStage 归档本集摘要并生成下集提示（骨架为占位）。
type LearnStage struct{}

func (LearnStage) ID() string { return "learn" }

func (LearnStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	v := vault.ForSeries(rc.App, rc.SeriesID)
	return agent.RunAnalyst(rc, v)
}
