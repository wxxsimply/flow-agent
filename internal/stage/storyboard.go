package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// StoryboardStage 生成可拍摄分镜与 SSML 旁白。
type StoryboardStage struct{}

func (StoryboardStage) ID() string { return "storyboard" }

func (StoryboardStage) Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error {
	_ = ctx
	_ = def
	v := vault.ForSeries(rc.App, rc.SeriesID)
	return agent.RunStoryboard(rc, v)
}
