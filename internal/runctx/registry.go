package runctx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type runRegistry struct {
	Runs map[string]string `json:"runs"` // run_id -> absolute run_dir
}

var registryMu sync.Mutex

func registryPath(dataDir string) string {
	return filepath.Join(dataDir, "run-registry.json")
}

func loadRegistry(dataDir string) (*runRegistry, error) {
	path := registryPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &runRegistry{Runs: map[string]string{}}, nil
		}
		return nil, err
	}
	var reg runRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg.Runs == nil {
		reg.Runs = map[string]string{}
	}
	return &reg, nil
}

func saveRegistry(dataDir string, reg *runRegistry) error {
	if reg.Runs == nil {
		reg.Runs = map[string]string{}
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	path := registryPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func registerRun(dataDir, runID, runDir string) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return err
	}
	reg.Runs[runID] = runDir
	return saveRegistry(dataDir, reg)
}

func lookupRunDir(dataDir, runID string) (string, bool) {
	registryMu.Lock()
	defer registryMu.Unlock()
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return "", false
	}
	dir, ok := reg.Runs[runID]
	return dir, ok
}

func listRegisteredRuns(dataDir string) (map[string]string, error) {
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(reg.Runs))
	for id, dir := range reg.Runs {
		out[id] = dir
	}
	return out, nil
}

func unregisterRun(dataDir, runID string) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return err
	}
	delete(reg.Runs, runID)
	return saveRegistry(dataDir, reg)
}
