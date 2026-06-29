// Package config 负责项目根目录发现、应用路径与 Provider/Stack 配置加载。
package config

import (
	"os"
	"path/filepath"
)

// FindRoot 查找项目根目录：环境变量 → 可执行文件旁 → 向上查找 go.mod / config/stacks。
func FindRoot() (string, error) {
	if v := os.Getenv("FLOWAGENT_ROOT"); v != "" {
		if isProjectRoot(v) {
			return filepath.Clean(v), nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		for _, candidate := range []string{
			dir,
			filepath.Join(dir, ".."),
			filepath.Join(dir, "..", ".."),
		} {
			if isProjectRoot(candidate) {
				return filepath.Clean(candidate), nil
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isProjectRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func isProjectRoot(dir string) bool {
	dir = filepath.Clean(dir)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "config", "stacks")); err == nil {
		return true
	}
	return false
}
