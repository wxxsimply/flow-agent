package ffmpeg

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ImageMatchesSize 检查图片是否为精确 WxH（Sora input_reference 须与 size 一致）。
func ImageMatchesSize(path string, w, h int) bool {
	gotW, gotH, err := imageDimensions(path)
	if err != nil {
		return false
	}
	return gotW == w && gotH == h
}

// ScaleImageToSize 缩放并中心裁剪到精确 WxH。
func ScaleImageToSize(src, dst string, w, h int) error {
	bin := ffmpegBin()
	if bin == "" {
		return fmt.Errorf("ffmpeg not found")
	}
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h)
	cmd := exec.Command(bin,
		"-y",
		"-i", src,
		"-vf", filter,
		"-frames:v", "1",
		dst,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func imageDimensions(path string) (w, h int, err error) {
	bin := ffprobeBin()
	if bin == "" {
		return 0, 0, fmt.Errorf("ffprobe not found")
	}
	cmd := exec.Command(bin,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output: %q", out)
	}
	w, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	h, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}
