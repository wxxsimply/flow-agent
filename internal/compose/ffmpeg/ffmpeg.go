// Package ffmpeg 调用本地 FFmpeg 合成竖屏 master.mp4。
package ffmpeg

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func slogWarnCrossfade(err error) {
	slog.Warn("clip crossfade failed, using hard concat", "err", err)
}

// ComposeOptions 合成参数。
type ComposeOptions struct {
	RunDir            string
	DryRun            bool
	DurationSec       int
	Timeline          *artifacts.Timeline
	SubtitleASS       string // 全局 ASS 路径（K3）
	UseGlobalASS      bool   // 为 true 时不按镜 drawtext
	BGMPath           string // 可选 BGM（K4）
	BGMVolume         float64
	ClipCrossfadeSec  float64
	ClipEdgeFadeSec   float64 // 单镜首尾淡入淡出
	VideoNativeOnly   bool    // 仅 mp4 镜头；缺视频则失败，禁止 Ken Burns
}

// Available 检查本机是否可用 ffmpeg。
func Available() bool {
	return ffmpegBin() != ""
}

// BinPath 返回解析到的 ffmpeg 路径（调试用）。
func BinPath() string {
	return ffmpegBin()
}

// Compose 在 RunDir/artifacts/ 下生成 master.mp4。
func Compose(opts ComposeOptions) error {
	out := filepath.Join(opts.RunDir, artifacts.CanonicalWriteRel("artifacts/master.mp4"))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	assets := filepath.Join(opts.RunDir, artifacts.ShotsDir)
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return err
	}

	if opts.DryRun || !Available() {
		if !Available() && !opts.DryRun {
			fmt.Fprintln(os.Stderr, "ffmpeg: not found — set FFMPEG_PATH or install ffmpeg (winget install Gyan.FFmpeg); writing placeholder master.mp4")
		}
		return writePlaceholder(out, opts.DryRun)
	}

	if opts.Timeline != nil && len(opts.Timeline.Shots) > 0 {
		return composeFromTimeline(opts, out)
	}

	dur := opts.DurationSec
	if dur <= 0 {
		dur = 10
	}
	return runBlackClip(out, dur)
}

func composeFromTimeline(opts ComposeOptions, out string) error {
	runDir := opts.RunDir
	tl := opts.Timeline
	tmpDir := filepath.Join(runDir, artifacts.CacheClipsDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	fps := tl.FPS
	if fps <= 0 {
		fps = 24
	}
	resSpec := tl.Resolution
	if strings.TrimSpace(resSpec) == "" {
		resSpec = "1080x1920"
	}

	var clipPaths []string
	for i, shot := range tl.Shots {
		clipPath := filepath.Join(tmpDir, fmt.Sprintf("clip-%03d.mp4", i+1))
		dur := shot.DurationSec
		if dur <= 0 {
			dur = 3
		}

		videoPath := filepath.Join(runDir, shot.VideoPath)
		imagePath := filepath.Join(runDir, shot.ImagePath)

		subtitle := shot.Subtitle
		if opts.UseGlobalASS {
			subtitle = ""
		}

		var err error
		if fileExists(videoPath) {
			err = buildVideoClipWithOptionalTail(videoPath, imagePath, clipPath, dur, shot.AudioDurationSec, shot.VideoDurationSec, tl.FPS, 0, tl.Resolution)
		} else if opts.VideoNativeOnly {
			err = fmt.Errorf("missing mp4 for motion-only compose (video_path=%s)", shot.VideoPath)
		} else if fileExists(imagePath) {
			err = kenBurnsClip(imagePath, clipPath, dur, tl.FPS, subtitle, i, tl.Resolution)
		} else {
			err = runBlackClip(clipPath, int(dur+0.5))
		}
		if err != nil {
			return fmt.Errorf("shot %s: %w", shot.ID, err)
		}
		if opts.ClipEdgeFadeSec > 0 {
			softPath := clipPath + ".soft.mp4"
			if err := softenClipEdges(clipPath, softPath, opts.ClipEdgeFadeSec); err == nil {
				clipPath = softPath
			}
		}
		normPath := filepath.Join(tmpDir, fmt.Sprintf("norm-%03d.mp4", i+1))
		if err := normalizeClip(clipPath, normPath, fps, resSpec); err != nil {
			return fmt.Errorf("shot %s normalize: %w", shot.ID, err)
		}
		clipPaths = append(clipPaths, normPath)
	}

	concatPath := filepath.Join(tmpDir, "concat.mp4")
	if opts.ClipCrossfadeSec > 0 && len(clipPaths) > 1 {
		if err := concatClipsCrossfade(clipPaths, concatPath, opts.ClipCrossfadeSec); err != nil {
			slogWarnCrossfade(err)
			if err := concatClipsReencode(clipPaths, concatPath, fps, resSpec); err != nil {
				return err
			}
		}
	} else if err := concatClipsReencode(clipPaths, concatPath, fps, resSpec); err != nil {
		return err
	}

	scaledPath := filepath.Join(tmpDir, "scaled.mp4")
	if err := scaleToResolution(concatPath, scaledPath, resSpec); err != nil {
		return err
	}

	audioPath := filepath.Join(runDir, artifacts.CanonicalWriteRel("artifacts/narration.mp3"))
	muxed := filepath.Join(tmpDir, "muxed.mp4")
	if fileExists(audioPath) {
		if err := muxVideoAudio(scaledPath, audioPath, muxed); err != nil {
			return err
		}
	} else {
		slog.Warn("narration.mp3 missing; composing video without narration track")
		muxed = scaledPath
	}

	current := muxed
	assPath := opts.SubtitleASS
	if assPath == "" && tl.SubtitleFile != "" {
		assPath = filepath.Join(runDir, tl.SubtitleFile)
	}
	if assPath != "" && fileExists(assPath) {
		withSubs := filepath.Join(tmpDir, "with_subs.mp4")
		if err := burnASS(current, assPath, withSubs); err != nil {
			return err
		}
		current = withSubs
	}

	if opts.BGMPath != "" && fileExists(opts.BGMPath) {
		withBGM := filepath.Join(tmpDir, "with_bgm.mp4")
		vol := opts.BGMVolume
		if vol <= 0 {
			vol = 0.25
		}
		if err := mixBGM(current, opts.BGMPath, withBGM, vol); err != nil {
			return err
		}
		current = withBGM
	}

	if current == out {
		return nil
	}
	return copyFile(current, out)
}

func parseResolution(resolution string) (w, h int) {
	w, h = 1080, 1920
	if strings.Contains(resolution, "1920x1080") {
		w, h = 1920, 1080
	}
	return w, h
}

// normalizeClip 统一单镜片段的 fps、分辨率与像素格式，避免镜间硬拼卡顿。
func normalizeClip(in, out string, fps int, resolution string) error {
	if fps <= 0 {
		fps = 24
	}
	w, h := parseResolution(resolution)
	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,fps=%d,format=yuv420p",
		w, h, w, h, fps,
	)
	cmd := exec.Command(ffmpegBin(),
		"-y", "-i", in,
		"-vf", filter,
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an",
		out,
	)
	return runCmd(cmd)
}

// concatClipsReencode 规格已统一的片段拼接（重编码，避免 stream copy 帧率不一致）。
func concatClipsReencode(clips []string, out string, fps int, resolution string) error {
	if len(clips) == 0 {
		return fmt.Errorf("no clips to concat")
	}
	if len(clips) == 1 {
		return copyFile(clips[0], out)
	}
	merged := out + ".pre.mp4"
	if err := concatClips(clips, merged); err != nil {
		return err
	}
	defer os.Remove(merged)
	return normalizeClip(merged, out, fps, resolution)
}

func scaleToResolution(in, out, resolution string) error {
	w, h := parseResolution(resolution)
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h)
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", in,
		"-vf", filter,
		"-c:v", "libx264",
		"-c:a", "copy",
		out,
	)
	return runCmd(cmd)
}

// muxVideoAudio 混流旁白，不以 -shortest 截断（K1：时长由时间轴对齐保证）。
func muxVideoAudio(videoPath, audioPath, out string) error {
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-map", "0:v:0",
		"-map", "1:a:0",
		out,
	)
	return runCmd(cmd)
}

func mixBGM(videoPath, bgmPath, out string, bgmVol float64) error {
	dur, err := ProbeAudioDurationSec(videoPath)
	if err != nil || dur <= 0 {
		dur = 180
	}
	filter := fmt.Sprintf(
		"[1:a]aloop=loop=-1:size=2e+09,atrim=0:%.2f,volume=%.3f[bgm];[0:a][bgm]amix=inputs=2:duration=first:dropout_transition=2[outa]",
		dur, bgmVol,
	)
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", videoPath,
		"-i", bgmPath,
		"-filter_complex", filter,
		"-map", "0:v:0",
		"-map", "[outa]",
		"-c:v", "copy",
		"-c:a", "aac",
		out,
	)
	return runCmd(cmd)
}

func burnASS(videoPath, assPath, out string) error {
	rel, err := filepath.Rel(filepath.Dir(out), assPath)
	if err != nil {
		rel = assPath
	}
	rel = escapeFilterPath(filepath.ToSlash(rel))
	filter := fmt.Sprintf("subtitles='%s'", rel)
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", videoPath,
		"-vf", filter,
		"-c:v", "libx264",
		"-c:a", "copy",
		out,
	)
	cmd.Dir = filepath.Dir(out)
	return runCmd(cmd)
}

// KenBurnsClip 由静图生成 Ken Burns mp4（无音轨）。
func KenBurnsClip(imagePath, out string, durSec float64, fps int, subtitle string, shotIndex int) error {
	return KenBurnsClipSized(imagePath, out, durSec, fps, subtitle, shotIndex, "1080x1920")
}

func KenBurnsClipSized(imagePath, out string, durSec float64, fps int, subtitle string, shotIndex int, resolution string) error {
	if strings.TrimSpace(resolution) == "" {
		resolution = "1080x1920"
	}
	return kenBurnsClip(imagePath, out, durSec, fps, subtitle, shotIndex, resolution)
}

func kenBurnsClip(imagePath, out string, durSec float64, fps int, subtitle string, shotIndex int, resolution string) error {
	if fps <= 0 {
		fps = 24
	}
	frames := int(durSec*float64(fps)) + 1
	if frames < 1 {
		frames = 1
	}
	filter := kenBurnsFilter(shotIndex, frames, fps, resolution)
	clipDir := filepath.Dir(out)
	imageInput := imagePath
	if sub := strings.TrimSpace(subtitle); sub != "" {
		textName := strings.TrimSuffix(filepath.Base(out), ".mp4") + ".sub.txt"
		textFile := filepath.Join(clipDir, textName)
		if err := os.WriteFile(textFile, []byte(sub), 0o644); err != nil {
			return fmt.Errorf("subtitle file: %w", err)
		}
		font := drawtextFontArg()
		// textfile 用相对路径，避免 D\: 盘符在滤镜里被误解析
		filter += fmt.Sprintf(",drawtext=%stextfile='%s':fontcolor=white:fontsize=42:borderw=2:bordercolor=black@0.6:x=(w-text_w)/2:y=h-160",
			font, escapeFilterPath(textName))
	}
	if rel, err := filepath.Rel(clipDir, imagePath); err == nil {
		imageInput = filepath.ToSlash(rel)
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-loop", "1",
		"-i", imageInput,
		"-vf", filter,
		"-t", fmt.Sprintf("%.2f", durSec),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-an",
		out,
	)
	cmd.Dir = clipDir
	return runCmd(cmd)
}

func trimVideoClip(in, out string, durSec float64) error {
	return trimVideoClipToDuration(in, out, durSec, false)
}

// trimVideoClipToDuration 裁剪视频；extend=false 时不将短视频静帧拉长到 durSec。
func trimVideoClipToDuration(in, out string, durSec float64, extend bool) error {
	probed, err := ProbeVideoDurationSec(in)
	if err != nil || probed <= 0 {
		probed = durSec
	}
	effective := durSec
	if !extend && probed < effective {
		effective = probed
	}
	if effective > probed {
		effective = probed
	}
	if effective < 0.1 {
		effective = probed
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", in,
		"-t", fmt.Sprintf("%.3f", effective),
		"-c:v", "libx264",
		"-c:a", "aac",
		out,
	)
	return runCmd(cmd)
}

// buildVideoClipWithOptionalTail 合成单镜：主视频 + 不足时长时用关键帧 Ken Burns 补尾（避免静帧定格）。
func buildVideoClipWithOptionalTail(videoPath, imagePath, out string, targetDur, audioDur, probedVideo float64, fps int, shotIndex int, resolution string) error {
	if targetDur <= 0 {
		targetDur = 3
	}
	if probedVideo <= 0 {
		var err error
		probedVideo, err = ProbeVideoDurationSec(videoPath)
		if err != nil || probedVideo <= 0 {
			probedVideo = targetDur
		}
	}
	gap := targetDur - probedVideo
	if gap < 0.2 {
		return trimVideoClipToDuration(videoPath, out, targetDur, false)
	}
	mainPath := out + ".main.mp4"
	if err := trimVideoClipToDuration(videoPath, mainPath, probedVideo, false); err != nil {
		return err
	}
	tailPath := out + ".tail.mp4"
	still := imagePath
	if !fileExists(still) {
		still = videoPath
	}
	if err := kenBurnsClip(still, tailPath, gap, fps, "", shotIndex, resolution); err != nil {
		_ = os.Remove(mainPath)
		return trimVideoClipToDuration(videoPath, out, targetDur, false)
	}
	var err error
	if gap >= 0.5 {
		fadeSec := 0.25
		if gap < fadeSec*2 {
			fadeSec = gap * 0.4
		}
		err = concatClipsCrossfade([]string{mainPath, tailPath}, out, fadeSec)
	} else {
		err = concatClips([]string{mainPath, tailPath}, out)
	}
	_ = os.Remove(mainPath)
	_ = os.Remove(tailPath)
	if err != nil {
		return trimVideoClipToDuration(videoPath, out, minFloat(targetDur, probedVideo), false)
	}
	_ = audioDur // reserved for future per-shot audio mux
	return nil
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// TrimVideoToDuration 裁剪/限制视频时长（供多关键帧拼接后对齐旁白）；不静帧拉长。
func TrimVideoToDuration(in, out string, durSec float64) error {
	return trimVideoClipToDuration(in, out, durSec, false)
}

// ConcatVideoClips 硬拼接多个 mp4 片段。
func ConcatVideoClips(clips []string, out string) error {
	return concatClips(clips, out)
}

func concatClips(clips []string, out string) error {
	listPath := out + ".txt"
	var b strings.Builder
	for _, c := range clips {
		escaped := strings.ReplaceAll(c, "'", "'\\''")
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
	return runCmd(cmd)
}

func muxAudio(videoPath, audioPath, out string) error {
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-shortest",
		out,
	)
	return runCmd(cmd)
}

// RunBlackClip 生成纯色占位 mp4（dry-run / 测试用）。
func RunBlackClip(out string, durationSec int) error {
	return runBlackClip(out, durationSec)
}

func runBlackClip(out string, durationSec int) error {
	if durationSec <= 0 {
		durationSec = 5
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=black:s=1080x1920:d=%d", durationSec),
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=stereo",
		"-shortest",
		"-c:v", "libx264",
		"-c:a", "aac",
		out,
	)
	if err := runCmd(cmd); err != nil {
		return fmt.Errorf("ffmpeg: %w (install ffmpeg for real output)", err)
	}
	return nil
}

func writePlaceholder(out string, dryRun bool) error {
	msg := []byte("FLOWAGENT_PLACEHOLDER_MP4\nReplace with ffmpeg output when assets are ready.\n")
	if dryRun {
		msg = append([]byte("[dry-run]\n"), msg...)
	}
	return os.WriteFile(out, msg, 0o644)
}

// drawtextFontArg 返回 drawtext 的 fontfile= 前缀（路径加引号，避免 Windows 盘符被当成选项分隔符）。
func drawtextFontArg() string {
	candidates := []string{
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\msyhbd.ttc`,
		`C:\Windows\Fonts\simhei.ttf`,
	}
	for _, p := range candidates {
		if fileExists(p) {
			return "fontfile='" + escapeFilterPath(p) + "':"
		}
	}
	return ""
}

// escapeFilterPath 转义 ffmpeg -vf 滤镜中的路径（仅转义盘符冒号，勿把 C\: 里的反斜杠删掉）。
func escapeFilterPath(p string) string {
	p = filepath.ToSlash(p)
	if len(p) >= 2 && p[1] == ':' {
		p = p[:1] + `\:` + p[2:]
	}
	p = strings.ReplaceAll(p, `'`, `\'`)
	return p
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// GenerateSilentMP3 用 ffmpeg 生成指定时长的静音 mp3（占位/测试）。
func GenerateSilentMP3(out string, durationSec float64) error {
	if !Available() {
		return os.WriteFile(out, []byte("FLOWAGENT_PLACEHOLDER_MP3\n"), 0o644)
	}
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=stereo",
		"-t", fmt.Sprintf("%.2f", durationSec),
		"-c:a", "libmp3lame",
		out,
	)
	return runCmd(cmd)
}

// GeneratePlaceholderPNG 生成竖屏占位图（纯色，避免 lavfi drawtext 引号解析失败）。
func GeneratePlaceholderPNG(out, label string) error {
	_ = label
	if !Available() {
		return os.WriteFile(out, []byte("FLOWAGENT_PLACEHOLDER_PNG\n"), 0o644)
	}
	// color 滤镜用 d= 表示时长；勿在 -i 里嵌 drawtext（shot id 含引号会炸滤镜链）
	filter := "color=c=0x1a1a2e:s=1080x1920:d=0.04"
	cmd := exec.Command(ffmpegBin(),
		"-y",
		"-f", "lavfi",
		"-i", filter,
		"-frames:v", "1",
		out,
	)
	return runCmd(cmd)
}

func kenBurnsFilter(shotIndex, frames, fps int, resolution string) string {
	w, h := 1080, 1920
	if strings.Contains(resolution, "1920x1080") {
		w, h = 1920, 1080
	}
	base := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,", w, h, w, h)
	size := fmt.Sprintf("%dx%d", w, h)
	switch shotIndex % 4 {
	case 1:
		return base + fmt.Sprintf("zoompan=z='if(lte(zoom,1.0),1.28,max(1.001,zoom-0.0016))':d=%d:s=%s:fps=%d", frames, size, fps)
	case 2:
		return base + fmt.Sprintf("zoompan=z='1.22':x='(iw-iw/zoom)*on/%d':y='ih/4':d=%d:s=%s:fps=%d", frames, frames, size, fps)
	case 3:
		return base + fmt.Sprintf("zoompan=z='1.22':x='(iw-iw/zoom)*(1-on/%d)':y='ih/4':d=%d:s=%s:fps=%d", frames, frames, size, fps)
	default:
		return base + fmt.Sprintf("zoompan=z='min(zoom+0.0018,1.32)':d=%d:s=%s:fps=%d", frames, size, fps)
	}
}

func softenClipEdges(in, out string, fadeSec float64) error {
	dur, err := ProbeVideoDurationSec(in)
	if err != nil || dur <= fadeSec*2.5 {
		return copyFile(in, out)
	}
	startOut := dur - fadeSec
	if startOut < 0 {
		startOut = 0
	}
	filter := fmt.Sprintf("fade=t=in:st=0:d=%.3f,fade=t=out:st=%.3f:d=%.3f", fadeSec, startOut, fadeSec)
	cmd := exec.Command(ffmpegBin(),
		"-y", "-i", in,
		"-vf", filter,
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an",
		out,
	)
	return runCmd(cmd)
}

func concatClipsCrossfade(clips []string, out string, fadeSec float64) error {
	if len(clips) == 1 {
		return copyFile(clips[0], out)
	}
	current := clips[0]
	tmpBase := out + ".xf"
	for i := 1; i < len(clips); i++ {
		nextOut := fmt.Sprintf("%s-%03d.mp4", tmpBase, i)
		d0, err := ProbeVideoDurationSec(current)
		if err != nil {
			return err
		}
		offset := d0 - fadeSec
		if offset < 0 {
			offset = 0
		}
		filter := fmt.Sprintf("[0:v][1:v]xfade=transition=fade:duration=%.3f:offset=%.3f[v]", fadeSec, offset)
		cmd := exec.Command(ffmpegBin(),
			"-y", "-i", current, "-i", clips[i],
			"-filter_complex", filter,
			"-map", "[v]", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-an",
			nextOut,
		)
		if err := runCmd(cmd); err != nil {
			return err
		}
		current = nextOut
	}
	return copyFile(current, out)
}

// ProbeVideoDurationSec 读取视频时长（秒）。
func ProbeVideoDurationSec(path string) (float64, error) {
	if !Available() {
		return 0, fmt.Errorf("ffmpeg not available")
	}
	cmd := exec.Command(ffprobeBin(),
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return 0, fmt.Errorf("%w: %s", err, msg)
		}
		return 0, err
	}
	var dur float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur)
	return dur, err
}

// ProbeAudioDurationSec 读取音频时长（秒）。
func ProbeAudioDurationSec(path string) (float64, error) {
	if !Available() {
		return 0, fmt.Errorf("ffmpeg not available")
	}
	cmd := exec.Command(ffprobeBin(),
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return 0, fmt.Errorf("%w: %s", err, msg)
		}
		return 0, err
	}
	var dur float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur)
	return dur, err
}

// ProbeVideoResolution 读取视频宽高（像素）。
func ProbeVideoResolution(path string) (width, height int, err error) {
	if !Available() {
		return 0, 0, fmt.Errorf("ffmpeg not available")
	}
	cmd := exec.Command(ffprobeBin(),
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		path,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return 0, 0, fmt.Errorf("%w: %s", err, msg)
		}
		return 0, 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe resolution: %q", strings.TrimSpace(string(out)))
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &width); err != nil {
		return 0, 0, err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &height); err != nil {
		return 0, 0, err
	}
	return width, height, nil
}
