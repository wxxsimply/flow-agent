package agent

import (
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// mergeCharacterPatch 只信任 LLM 显式返回的 patch；
// 不做任何剧情推断，避免 agent 替剧情下断言（如预写未揭示的真相）。
func mergeCharacterPatch(modelPatch map[string]any, _ *artifacts.ContinuityReport) map[string]any {
	if len(modelPatch) == 0 {
		return nil
	}
	out := make(map[string]any, len(modelPatch))
	for k, v := range modelPatch {
		out[k] = v
	}
	return out
}
