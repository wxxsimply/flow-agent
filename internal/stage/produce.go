package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// ProduceStage TTS、出图、可灵、FFmpeg 合成 master.mp4。
type ProduceStage struct{}

func (ProduceStage) ID() string { return "produce" }

func (ProduceStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = def
	return agent.RunProducer(ctx, rc)
}
