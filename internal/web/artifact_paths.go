package web

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func artifactExists(runDir, rel string) bool {
	return artifacts.FileExists(runDir, rel)
}

func resolveArtifactAbs(runDir, rel string) string {
	return artifacts.ResolvePath(runDir, rel)
}

// resolveArtifactAbsSafe 解析 run 内产物绝对路径，拒绝目录穿越。
func resolveArtifactAbsSafe(runDir, rel string) (string, error) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = filepath.Clean(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("invalid path")
	}
	runRoot, err := filepath.Abs(filepath.Clean(runDir))
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Join(runRoot, rel))
	if err != nil {
		return "", err
	}
	return assertAbsUnderRun(runRoot, abs)
}

// assertAbsUnderRun 确认绝对路径仍在 run 根目录内。
func assertAbsUnderRun(runDir, abs string) (string, error) {
	runRoot, err := filepath.Abs(filepath.Clean(runDir))
	if err != nil {
		return "", err
	}
	abs, err = filepath.Abs(filepath.Clean(abs))
	if err != nil {
		return "", err
	}
	relToRoot, err := filepath.Rel(runRoot, abs)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside run dir")
	}
	return abs, nil
}
