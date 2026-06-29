package wmreward

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Ready 检测 V-JEPA WMReward 环境是否可用。
func Ready(projectRoot string) bool {
	repo := strings.TrimSpace(os.Getenv("WMREWARD_REPO"))
	if repo == "" {
		return false
	}
	if _, err := os.Stat(repo); err != nil {
		return false
	}
	script := filepath.Join(projectRoot, "scripts", "wmreward", "compute_vjepa_surprise_wrapper.py")
	if _, err := os.Stat(script); err != nil {
		return false
	}
	cmd := exec.Command("python", "-c", "import sys; sys.path.insert(0, r'"+repo+"'); import compute_wmreward")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

// ScriptPath 返回 stack 默认 V-JEPA 包装脚本相对路径。
func ScriptPath() string {
	return "scripts/wmreward/compute_vjepa_surprise_wrapper.py"
}
