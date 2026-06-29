package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// IsRealVideoFile 判断是否为可解码的视频（非文本占位）。
func IsRealVideoFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 16 {
		return false
	}
	if string(data[:4]) == "FLOW" || string(data[:8]) == "[dry-run" {
		return false
	}
	// MP4 ftyp
	return data[4] == 'f' && data[5] == 't' && data[6] == 'y' && data[7] == 'p'
}

// ExtractCoverFrame 从视频截取封面帧为 JPEG。
func ExtractCoverFrame(videoPath, outPath string, atSec float64) error {
	if !Available() {
		return fmt.Errorf("ffmpeg not available")
	}
	if !IsRealVideoFile(videoPath) {
		return fmt.Errorf("not a real video file: %s", videoPath)
	}
	if atSec < 0 {
		atSec = 1
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-ss", fmt.Sprintf("%.2f", atSec),
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		outPath,
	)
	if err := runCmd(cmd); err != nil {
		return fmt.Errorf("extract cover: %w", err)
	}
	return nil
}
