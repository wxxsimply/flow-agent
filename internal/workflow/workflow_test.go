package workflow

import (
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

// TestLoadNovelShortDouyin 确保默认工作流可被解析且阶段数完整。
func TestLoadNovelShortDouyin(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root:", err)
	}
	def, err := Load(filepath.Join(root, "docs", "workflows"), "novel-short-douyin")
	if err != nil {
		t.Fatal(err)
	}
	if def.Name != "novel-short-douyin" {
		t.Fatalf("name=%q", def.Name)
	}
	if len(def.Stages) < 8 { // plan → learn 共 8 阶段
		t.Fatalf("stages=%d", len(def.Stages))
	}
	plan := def.StageByID("plan")
	if plan == nil {
		t.Fatal("plan stage missing")
	}
	hooks := ParseStageHooks(plan.Hooks)
	if len(hooks) == 0 {
		t.Fatal("expected plan hooks")
	}
	before := HooksForPhase(hooks, "before", "plan")
	if len(before) == 0 || len(before[0].Actions) < 2 {
		t.Fatalf("before_plan hooks: %+v", before)
	}
}
