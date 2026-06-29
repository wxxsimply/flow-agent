package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConcatAudioFiles 将多个音频按顺序拼接为一个文件。
func ConcatAudioFiles(paths []string, out string) error {
	if len(paths) == 0 {
		return fmt.Errorf("concat audio: no inputs")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("concat audio mkdir: %w", err)
	}
	if len(paths) == 1 {
		return copyFile(paths[0], out)
	}
	if !Available() {
		return fmt.Errorf("ffmpeg not available")
	}
	listPath := out + ".list.txt"
	var b strings.Builder
	for _, p := range paths {
		escaped := strings.ReplaceAll(p, "'", "'\\''")
		b.WriteString("file '")
		b.WriteString(escaped)
		b.WriteString("'\n")
	}
	if err := os.WriteFile(listPath, []byte(b.String()), 0o644); err != nil {
		return err
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		out,
	)
	if err := runCmd(cmd); err != nil {
		return concatAudioReencode(paths, out)
	}
	return nil
}

// PadAudioToDuration 将音频尾部补静音至指定时长（旁白短于视频槽位时使用）。
func PadAudioToDuration(path string, durationSec float64) error {
	if durationSec <= 0 {
		return nil
	}
	cur, err := ProbeAudioDurationSec(path)
	if err != nil {
		return fmt.Errorf("pad audio probe: %w", err)
	}
	if cur >= durationSec-0.05 {
		return nil
	}
	if !Available() {
		return fmt.Errorf("ffmpeg not available")
	}
	padSec := durationSec - cur
	tmp := path + ".pad.tmp.mp3"
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", path,
		"-af", fmt.Sprintf("apad=pad_dur=%.3f", padSec),
		"-t", fmt.Sprintf("%.3f", durationSec),
		"-c:a", "libmp3lame",
		tmp,
	)
	if err := runCmd(cmd); err != nil {
		return fmt.Errorf("pad audio: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// TrimAudioToDuration 将音频截断至指定时长（旁白长于视频槽位时使用）。
func TrimAudioToDuration(path string, durationSec float64) error {
	if durationSec <= 0 {
		return nil
	}
	cur, err := ProbeAudioDurationSec(path)
	if err != nil {
		return fmt.Errorf("trim audio probe: %w", err)
	}
	if cur <= durationSec+0.05 {
		return nil
	}
	if !Available() {
		return fmt.Errorf("ffmpeg not available")
	}
	tmp := path + ".trim.tmp.mp3"
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", path,
		"-t", fmt.Sprintf("%.3f", durationSec),
		"-c:a", "libmp3lame",
		tmp,
	)
	if err := runCmd(cmd); err != nil {
		return fmt.Errorf("trim audio: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func concatAudioReencode(paths []string, out string) error {
	args := []string{"-y"}
	for _, p := range paths {
		args = append(args, "-i", p)
	}
	n := len(paths)
	var filter strings.Builder
	for i := 0; i < n; i++ {
		filter.WriteString(fmt.Sprintf("[%d:a]", i))
	}
	filter.WriteString(fmt.Sprintf("concat=n=%d:v=0:a=1[outa]", n))
	filterStr := filter.String()
	args = append(args, "-filter_complex", filterStr, "-map", "[outa]", "-c:a", "libmp3lame", out)
	cmd := exec.Command(ffmpegBin(), args...)
	return runCmd(cmd)
}
