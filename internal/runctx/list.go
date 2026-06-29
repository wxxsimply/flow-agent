package runctx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunSummary 历史列表摘要。
type RunSummary struct {
	RunID          string    `json:"run_id"`
	Title          string    `json:"title"`
	Stage          string    `json:"stage"`
	Stack          string    `json:"stack_profile,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Finished       bool      `json:"finished"`
	Failed         bool      `json:"failed"`
	AwaitingReview bool      `json:"awaiting_review"`
	HasMaster      bool      `json:"has_master_video"`
	UserID         string    `json:"user_id,omitempty"`
	RunDir         string    `json:"run_dir,omitempty"`
}

// ListRuns 扫描注册表与 runs 目录，按 started_at 倒序返回摘要。
func (s *Store) ListRuns(userID string, limit int) ([]RunSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	seen := map[string]bool{}
	var out []RunSummary

	addSummary := func(runDir string) {
		m, err := artifacts.LoadManifest(filepath.Join(runDir, "manifest.json"))
		if err != nil {
			return
		}
		if seen[m.RunID] {
			return
		}
		if userID != "" && m.UserID != userID {
			return
		}
		seen[m.RunID] = true
		sum := manifestToSummary(m)
		sum.HasMaster = artifacts.FileExists(runDir, "artifacts/master.mp4")
		sum.RunDir = runDir
		out = append(out, sum)
	}

	if registered, err := listRegisteredRuns(s.DataDir); err == nil {
		for _, runDir := range registered {
			addSummary(runDir)
		}
	}

	entries, err := os.ReadDir(s.RunsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		addSummary(filepath.Join(s.RunsDir, e.Name()))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func manifestToSummary(m *artifacts.Manifest) RunSummary {
	title := strings.TrimSpace(m.Title)
	if title == "" {
		title = "未命名项目"
	}
	return RunSummary{
		RunID:          m.RunID,
		Title:          title,
		Stage:          m.Stage,
		Stack:          m.Stack,
		StartedAt:      m.StartedAt,
		UpdatedAt:      m.UpdatedAt,
		Finished:       m.Stage == "finished",
		Failed:         m.Stage == "failed",
		AwaitingReview: m.Stage == "awaiting_review",
		UserID:         m.UserID,
		RunDir:         m.RunDir,
	}
}

// AssertRunOwner 校验 run 归属；userID 为空（auth 关闭）时不校验。
func (s *Store) AssertRunOwner(runID, userID string) error {
	if userID == "" {
		return nil
	}
	rc, err := s.LoadRun(runID)
	if err != nil {
		return err
	}
	if rc.Manifest.UserID == "" {
		return fmt.Errorf("run not owned by user")
	}
	if rc.Manifest.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
