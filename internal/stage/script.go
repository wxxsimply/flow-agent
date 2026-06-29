package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// ScriptStage story-spine → script.json。
type ScriptStage struct{}

func (ScriptStage) ID() string { return "script" }

func (ScriptStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunScreenwriter(rc)
}
