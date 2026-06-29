package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// ContinuityStage 校验人设、伏笔与 bible 一致性。
type ContinuityStage struct{}

func (ContinuityStage) ID() string { return "continuity" }

func (ContinuityStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	v := vault.ForSeries(rc.App, rc.SeriesID)
	return agent.RunContinuity(rc, v)
}
