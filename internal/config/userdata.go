package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// UserDataDir 返回本机用户持久化目录（Desktop Settings / runs / registry）。
func UserDataDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "FlowAgent"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "FlowAgent"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if xdg == "" {
			xdg = filepath.Join(home, ".config")
		}
		return filepath.Join(xdg, "flow-agent"), nil
	}
}

// EnsureUserDataDir 创建用户数据目录。
func EnsureUserDataDir() error {
	dir, err := UserDataDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// UsesIsolatedUserConfig Desktop 等场景：FLOWAGENT_DATA_DIR 与项目根分离时，密钥仅来自 user-prefs。
func UsesIsolatedUserConfig() bool {
	dataDir := strings.TrimSpace(os.Getenv("FLOWAGENT_DATA_DIR"))
	if dataDir == "" {
		return false
	}
	root, err := FindRoot()
	if err != nil {
		return true
	}
	return filepath.Clean(dataDir) != filepath.Clean(root)
}

// ProviderConfigHint 根据运行模式返回配置指引文案。
func ProviderConfigHint() string {
	if UsesIsolatedUserConfig() {
		return "configure in Studio Settings (环境设置)"
	}
	return "edit config/providers.local.yaml"
}

// ProviderConfigHintZh 中文配置指引。
func ProviderConfigHintZh() string {
	if UsesIsolatedUserConfig() {
		return "请在环境设置中填写"
	}
	return "请编辑 config/providers.local.yaml"
}

// MigrateLegacyUserPrefs 若用户目录尚无 prefs，从项目根复制旧 user-prefs.json。
func MigrateLegacyUserPrefs(repoRoot string) error {
	userDir, err := UserDataDir()
	if err != nil {
		return err
	}
	newPath := filepath.Join(userDir, "user-prefs.json")
	if _, err := os.Stat(newPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	oldPath := filepath.Join(repoRoot, "user-prefs.json")
	data, err := os.ReadFile(oldPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := EnsureUserDataDir(); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if err := os.WriteFile(newPath, data, mode); err != nil {
		return fmt.Errorf("migrate user-prefs: %w", err)
	}
	return nil
}
