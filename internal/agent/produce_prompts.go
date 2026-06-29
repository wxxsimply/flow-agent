package agent

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/flow-agent/flow-agent/internal/runctx"
)

// ProducePromptEntry 单镜 produce 阶段实际使用的 prompt（调试与耦合验收）。
type ProducePromptEntry struct {
	ShotID               string   `json:"shot_id"`
	KeyframePrompts      []string `json:"keyframe_prompts,omitempty"`
	SegmentMotionPrompts []string `json:"segment_motion_prompts,omitempty"`
	SingleMotionPrompt   string   `json:"single_motion_prompt,omitempty"`
	BoNEnabled           bool     `json:"bon_enabled"`
}

type producePromptStore struct {
	mu      sync.Mutex
	entries []ProducePromptEntry
}

var globalProducePrompts producePromptStore

func resetProducePrompts() {
	globalProducePrompts.mu.Lock()
	globalProducePrompts.entries = nil
	globalProducePrompts.mu.Unlock()
}

func appendProducePromptEntry(e ProducePromptEntry) {
	globalProducePrompts.mu.Lock()
	globalProducePrompts.entries = append(globalProducePrompts.entries, e)
	globalProducePrompts.mu.Unlock()
}

func flushProducePrompts(rc *runctx.Context) error {
	globalProducePrompts.mu.Lock()
	data, err := json.MarshalIndent(globalProducePrompts.entries, "", "  ")
	globalProducePrompts.mu.Unlock()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		data = []byte("[]")
	}
	if err := rc.WriteArtifact("artifacts/produce-prompts.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("produce-prompts.json", "artifacts/produce-prompts.json", false)
	return nil
}

// loadProducePromptsIfExists 用于测试或续跑（当前 produce 开始时 reset）。
func loadProducePromptsIfExists(path string) ([]ProducePromptEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []ProducePromptEntry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
