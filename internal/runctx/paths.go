package runctx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ValidateOutputDir 校验用户指定的输出目录：绝对路径、可创建、可写、非系统敏感路径。
// allowExistingManifest 为 true 时允许目录内已有 manifest（iterate 复用）。
func ValidateOutputDir(dir string, allowExistingManifest bool) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("output directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)
	if err := rejectSensitivePath(abs); err != nil {
		return "", err
	}
	manifestPath := filepath.Join(abs, "manifest.json")
	if info, err := os.Stat(abs); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", fmt.Errorf("cannot create directory: %w", err)
		}
	} else if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", abs)
	} else {
		entries, err := os.ReadDir(abs)
		if err != nil {
			return "", err
		}
		hasManifest := fileExists(manifestPath)
		if len(entries) > 0 {
			if hasManifest {
				if !allowExistingManifest {
					return "", fmt.Errorf("directory already contains a project; use continue editing instead")
				}
			} else {
				return "", fmt.Errorf("directory is not empty and has no manifest")
			}
		}
	}
	if err := checkWritable(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// ValidateWorkspaceDir 校验工作区目录：可创建、可写、允许非空，但根目录不得已是项目（含 manifest）。
func ValidateWorkspaceDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("workspace directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)
	if err := rejectSensitivePath(abs); err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", fmt.Errorf("cannot create directory: %w", err)
		}
	} else if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", abs)
	} else if fileExists(filepath.Join(abs, "manifest.json")) {
		return "", fmt.Errorf("path is a project directory; select the parent workspace folder instead")
	}
	if err := checkWritable(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// AllocateProjectDir 在工作区下分配唯一子项目目录。
func AllocateProjectDir(workspace, title string) (string, error) {
	ws, err := ValidateWorkspaceDir(workspace)
	if err != nil {
		return "", err
	}
	slug := slugProjectTitle(title)
	ts := time.Now().Format("20060102-1504")
	base := fmt.Sprintf("%s-%s", slug, ts)
	candidate := filepath.Join(ws, base)
	for i := 0; i < 100; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		candidate = filepath.Join(ws, name)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			if err := os.MkdirAll(candidate, 0o755); err != nil {
				return "", err
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not allocate unique project directory under %s", ws)
}

var slugSanitize = regexp.MustCompile(`[^\p{L}\p{N}\-_]+`)

func slugProjectTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "project"
	}
	title = slugSanitize.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-_")
	runes := []rune(title)
	if len(runes) > 24 {
		title = string(runes[:24])
	}
	title = strings.Trim(title, "-_")
	if title == "" {
		return "project"
	}
	return title
}

// WorkspaceFromPath 若 path 本身是项目目录则返回其父工作区，否则返回规范化工作区路径。
func WorkspaceFromPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if fileExists(filepath.Join(abs, "manifest.json")) {
		return filepath.Dir(abs), nil
	}
	return ValidateWorkspaceDir(abs)
}

func rejectSensitivePath(abs string) error {
	lower := strings.ToLower(filepath.ToSlash(abs))
	blocked := []string{
		"/windows", "/system32", "/program files", "/program files (x86)",
		"/etc", "/usr/bin", "/bin", "/sbin", "/var",
	}
	for _, b := range blocked {
		if strings.Contains(lower, b) {
			return fmt.Errorf("cannot use system directory: %s", abs)
		}
	}
	if runtime.GOOS == "windows" && len(abs) >= 2 && abs[1] == ':' {
		vol := strings.ToLower(abs[:1])
		if vol == "c" {
			rest := strings.ToLower(filepath.ToSlash(abs[2:]))
			for _, prefix := range []string{"/windows", "/program files", "/program files (x86)", "/users/default"} {
				if strings.HasPrefix(rest, prefix) {
					return fmt.Errorf("cannot use system directory: %s", abs)
				}
			}
		}
	}
	return nil
}

func checkWritable(dir string) error {
	test := filepath.Join(dir, ".flowagent-write-test")
	if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	}
	_ = os.Remove(test)
	return nil
}
