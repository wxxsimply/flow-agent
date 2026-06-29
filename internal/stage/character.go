package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// CharacterStage 为微电影生成角色三视图设定。
type CharacterStage struct{}

func (CharacterStage) ID() string { return "character" }

func (CharacterStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	return agent.RunCharacterDesigner(rc)
}

