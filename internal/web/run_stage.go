package web

import (
	"github.com/flow-agent/flow-agent/internal/runctx"
)

func isRunTerminalStage(stage string) bool {
	switch stage {
	case "finished", "failed", "awaiting_review":
		return true
	default:
		return false
	}
}

func isRunStaleStage(stage string) bool {
	return stage != "" && !isRunTerminalStage(stage)
}

func inferResumeStage(rc *runctx.Context) string {
	if rc == nil || rc.RunDir == "" {
		return "assemble"
	}
	runDir := rc.RunDir
	if artifactExists(runDir, "artifacts/storyboard.json") {
		return "produce"
	}
	if artifactExists(runDir, "artifacts/shot-language-brief.md") {
		return "assemble"
	}
	return "assemble"
}
