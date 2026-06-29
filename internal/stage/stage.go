// Package stage 将工作流阶段 id 映射到具体 Handler，并委托 internal/agent 执行业务。
package stage

import (
	"context"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// Handler 单个流水线阶段的执行器。
type Handler interface {
	ID() string
	Run(ctx context.Context, rc *runctx.Context, def *workflow.StageDefinition) error
}

// Registry 阶段 id → Handler 注册表。
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// Register 注册阶段处理器。
func (r *Registry) Register(h Handler) {
	r.handlers[h.ID()] = h
}

// Get 按阶段 id 查找 Handler。
func (r *Registry) Get(id string) (Handler, bool) {
	h, ok := r.handlers[id]
	return h, ok
}

// DefaultRegistry 返回内容生产阶段注册表（含微电影 expand/script）。
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(&AssembleStage{})
	r.Register(&ExpandStage{})
	r.Register(&ScriptStage{})
	r.Register(&CharacterStage{})
	r.Register(&PropStage{})
	r.Register(&PlanStage{})
	r.Register(&WriteStage{})
	r.Register(&ContinuityStage{})
	r.Register(&StoryboardStage{})
	r.Register(&ProduceStage{})
	r.Register(&ComplyStage{})
	r.Register(&PublishStage{})
	r.Register(&LearnStage{})
	return r
}
