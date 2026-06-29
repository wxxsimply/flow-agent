package runctx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
	"github.com/google/uuid"
)

// Store 管理 runs/ 目录下的运行实例及自定义输出目录注册表。
type Store struct {
	RunsDir string // 通常为 <root>/runs
	DataDir string // 持久化数据根，存放 run-registry.json
}

// NewStore 创建 RunStore。DataDir 默认为 RunsDir 的父目录。
func NewStore(runsDir string) *Store {
	dataDir := filepath.Dir(runsDir)
	return &Store{RunsDir: runsDir, DataDir: dataDir}
}

// ResolveRunDir 定位 run 目录：注册表优先，兼容 runs/<id>/。
func (s *Store) ResolveRunDir(runID string) (string, error) {
	if dir, ok := lookupRunDir(s.DataDir, runID); ok && fileExists(filepath.Join(dir, "manifest.json")) {
		return dir, nil
	}
	legacy := filepath.Join(s.RunsDir, runID)
	if fileExists(filepath.Join(legacy, "manifest.json")) {
		return legacy, nil
	}
	return "", fmt.Errorf("load run %q: manifest not found", runID)
}

// CreateRun 新建一次运行：创建目录并写入初始 manifest（默认 runs/<uuid>/）。
func (s *Store) CreateRun(seriesID string, episode int, wfName, stack string, dryRun bool) (*Context, error) {
	runID := uuid.New().String()
	runDir := filepath.Join(s.RunsDir, runID)
	return s.createRunAt(runDir, runID, seriesID, episode, wfName, stack, dryRun, false)
}

// CreateRunInWorkspace 在工作区下自动创建子项目目录并初始化 run。
func (s *Store) CreateRunInWorkspace(workspace, title, seriesID string, episode int, wfName, stack string, dryRun bool) (*Context, error) {
	projectDir, err := AllocateProjectDir(workspace, title)
	if err != nil {
		return nil, err
	}
	runID := uuid.New().String()
	return s.createRunAt(projectDir, runID, seriesID, episode, wfName, stack, dryRun, true)
}

// CreateRunAt 在用户指定的目录创建 run（该目录即为 run 根目录）。
func (s *Store) CreateRunAt(outputDir, seriesID string, episode int, wfName, stack string, dryRun bool) (*Context, error) {
	abs, err := ValidateOutputDir(outputDir, false)
	if err != nil {
		return nil, err
	}
	runID := uuid.New().String()
	return s.createRunAt(abs, runID, seriesID, episode, wfName, stack, dryRun, true)
}

func (s *Store) createRunAt(runDir, runID, seriesID string, episode int, wfName, stack string, dryRun, register bool) (*Context, error) {
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	m := &artifacts.Manifest{
		RunID:     runID,
		TraceID:   uuid.New().String(),
		SeriesID:  seriesID,
		EpisodeNo: episode,
		Workflow:  wfName,
		Stack:     stack,
		Stage:     "created",
		Gates:     map[string]bool{},
		Cost:      &artifacts.CostLedger{},
		StartedAt: now,
		UpdatedAt: now,
		DryRun:    dryRun,
		RunDir:    runDir,
	}

	rc := &Context{
		RunID:        runID,
		TraceID:      m.TraceID,
		Workflow:     wfName,
		SeriesID:     seriesID,
		EpisodeNo:    episode,
		Stack:        stack,
		DryRun:       dryRun,
		Manifest:     m,
		RunDir:       runDir,
		ArtifactsDir: artDir,
	}
	if err := rc.SaveManifest(); err != nil {
		return nil, err
	}
	if register {
		if err := registerRun(s.DataDir, runID, runDir); err != nil {
			return nil, fmt.Errorf("register run: %w", err)
		}
	}
	return rc, nil
}

// LoadRun 从磁盘恢复已有运行上下文。
func (s *Store) LoadRun(runID string) (*Context, error) {
	runDir, err := s.ResolveRunDir(runID)
	if err != nil {
		return nil, err
	}
	m, err := artifacts.LoadManifest(filepath.Join(runDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("load run %q: %w", runID, err)
	}
	if m.RunID != runID {
		return nil, fmt.Errorf("load run %q: manifest run_id mismatch", runID)
	}
	return &Context{
		RunID:        m.RunID,
		TraceID:      m.TraceID,
		Workflow:     m.Workflow,
		SeriesID:     m.SeriesID,
		EpisodeNo:    m.EpisodeNo,
		Stack:        m.Stack,
		DryRun:       m.DryRun,
		Manifest:     m,
		RunDir:       runDir,
		ArtifactsDir: filepath.Join(runDir, "artifacts"),
		Stage:        m.Stage,
	}, nil
}

// DeleteRun 删除 run 注册表项及本地项目目录（不可恢复）。
func (s *Store) DeleteRun(runID, userID string) error {
	if err := s.AssertRunOwner(runID, userID); err != nil {
		return err
	}
	runDir, err := s.ResolveRunDir(runID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(runDir); err != nil {
		return fmt.Errorf("remove run dir: %w", err)
	}
	_ = unregisterRun(s.DataDir, runID)
	// 兼容未注册但落在 runs/<id>/ 的 legacy 目录
	legacy := filepath.Join(s.RunsDir, runID)
	if legacy != runDir {
		_ = os.RemoveAll(legacy)
	}
	return nil
}
