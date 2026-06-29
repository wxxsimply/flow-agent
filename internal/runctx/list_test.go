package runctx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestListRunsForUser(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	runDir := filepath.Join(dir, "run-a")
	if err := os.MkdirAll(filepath.Join(runDir, "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &artifacts.Manifest{
		RunID: "run-a", Workflow: "micro-movie", Stage: "finished",
		UserID: "user-1", Title: "测试项目",
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		Gates: map[string]bool{}, Cost: &artifacts.CostLedger{},
	}
	if err := m.Save(filepath.Join(runDir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	list, err := store.ListRuns("user-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Title != "测试项目" {
		t.Fatalf("list=%+v", list)
	}
	list, _ = store.ListRuns("other", 10)
	if len(list) != 0 {
		t.Fatalf("expected empty for other user, got %d", len(list))
	}
}
