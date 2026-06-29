package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ExtractVideoLastFrame 截取视频末帧为 PNG，用于镜间角色衔接。
func ExtractVideoLastFrame(videoPath, outPath string) error {
	if !Available() {
		return fmt.Errorf("ffmpeg not available")
	}
	if !IsRealVideoFile(videoPath) {
		return fmt.Errorf("not a real video file: %s", videoPath)
	}
	dur, err := ProbeVideoDurationSec(videoPath)
	if err != nil || dur <= 0 {
		return fmt.Errorf("probe duration: %w", err)
	}
	at := dur - 0.08
	if at < 0 {
		at = 0
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-sseof", "-0.12",
		"-i", videoPath,
		"-vframes", "1",
		"-f", "image2",
		outPath,
	)
	if err := runCmd(cmd); err != nil {
		cmd = exec.Command(ffmpegBin(),
			"-y",
			"-ss", fmt.Sprintf("%.3f", at),
			"-i", videoPath,
			"-vframes", "1",
			"-f", "image2",
			outPath,
		)
		if err2 := runCmd(cmd); err2 != nil {
			return fmt.Errorf("extract last frame: %w", err)
		}
	}
	return nil
}
