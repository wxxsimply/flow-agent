package runctx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestDeleteRun_removesDirAndRegistry(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	runDir := filepath.Join(dir, "proj-delete")
	if err := os.MkdirAll(filepath.Join(runDir, "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	runID := "run-del-1"
	m := &artifacts.Manifest{
		RunID: runID, Workflow: "micro-movie", Stage: "finished",
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		Gates: map[string]bool{}, Cost: &artifacts.CostLedger{},
		RunDir: runDir,
	}
	if err := m.Save(filepath.Join(runDir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if err := registerRun(store.DataDir, runID, runDir); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRun(runID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatalf("run dir should be removed: %v", err)
	}
	reg, err := loadRegistry(store.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Runs[runID]; ok {
		t.Fatal("registry should not contain deleted run")
	}
}
