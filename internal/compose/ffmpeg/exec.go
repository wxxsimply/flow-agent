package ffmpeg

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// runCmd 执行 ffmpeg/ffprobe；不把子进程 stderr 接到 os.Stderr（避免 PowerShell 将 banner 标为 NativeCommandError）。
func runCmd(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			if len(msg) > 800 {
				msg = msg[len(msg)-800:]
			}
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
