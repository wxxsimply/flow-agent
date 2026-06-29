package agent

import (
	"reflect"
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestMergeCharacterPatchTrustsModelOnly(t *testing.T) {
	modelPatch := map[string]any{
		"林晚": map[string]any{
			"known_secrets": []any{"本集结尾确认顾沉公司账目造假"},
		},
	}
	report := &artifacts.ContinuityReport{
		Issues: []artifacts.ContinuityIssue{
			{
				Severity: "critical",
				Message:  "林晚不该提前知道录音真相",
			},
		},
	}
	got := mergeCharacterPatch(modelPatch, report)
	if !reflect.DeepEqual(got, modelPatch) {
		t.Fatalf("expected merged patch to equal model patch, got: %v", got)
	}
}

func TestMergeCharacterPatchReturnsNilWhenModelEmpty(t *testing.T) {
	report := &artifacts.ContinuityReport{
		Issues: []artifacts.ContinuityIssue{
			{
				Severity: "critical",
				Message:  "林晚 录音 真相 秘密 — 旧版本会硬编码注入",
			},
		},
	}
	got := mergeCharacterPatch(nil, report)
	if got != nil {
		t.Fatalf("expected nil patch, got: %v", got)
	}

	got = mergeCharacterPatch(map[string]any{}, report)
	if got != nil {
		t.Fatalf("expected nil patch on empty map, got: %v", got)
	}
}
