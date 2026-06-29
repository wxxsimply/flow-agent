package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// StoryboardPolicyForRun 返回当前运行应使用的分镜校验策略。
func StoryboardPolicyForRun(rc *runctx.Context) artifacts.StoryboardPolicy {
	LoadCreativeOptionsFromRun(rc)
	if rc.Creative != nil && rc.Creative.IsDirector() {
		pol := artifacts.DirectorPolicy()
		if IsBudgetCapStack(rc) {
			pol.RelaxDurationTarget = false
			name := rc.Stack
			if name == "" && rc.App != nil && rc.App.Stack != nil {
				name = rc.App.Stack.Name
			}
			mm := artifacts.MicroMoviePolicy(name)
			pol.MinShots = mm.MinShots
			pol.MaxShots = mm.MaxShots
			pol.MinAIVideo = mm.MinAIVideo
			pol.MaxAIVideo = mm.MaxAIVideo
			pol.DurationToleranceSec = mm.DurationToleranceSec
		}
		return pol
	}
	if rc.Workflow == "micro-movie" {
		name := rc.Stack
		if name == "" && rc.App != nil && rc.App.Stack != nil {
			name = rc.App.Stack.Name
		}
		return artifacts.MicroMoviePolicy(name)
	}
	if rc.App != nil && rc.App.Stack != nil {
		return rc.App.Stack.StoryboardPolicy()
	}
	return artifacts.KenBurnsShortDramaPolicy()
}

// IsDirectorRun 是否逐镜导演模式。
func IsDirectorRun(rc *runctx.Context) bool {
	LoadCreativeOptionsFromRun(rc)
	return rc.Creative != nil && rc.Creative.IsDirector()
}

func isMicroMovieStack(rc *runctx.Context) bool {
	if rc.Workflow == "micro-movie" {
		return true
	}
	return strings.HasPrefix(rc.Stack, "micro-movie")
}
