package artifacts

import (
	"os"
	"path/filepath"
	"strings"
)

// 新布局目录常量。
const (
	StoryDir      = "artifacts/story"
	MediaDir      = "artifacts/media"
	ShotsDir      = "artifacts/media/shots"
	CacheDir      = "artifacts/.cache"
	CacheClipsDir = "artifacts/.cache/clips"
)

// canonicalWrites 将旧相对路径映射到新布局写入路径。
var canonicalWrites = map[string]string{
	"artifacts/storyboard.json":     StoryDir + "/storyboard.json",
	"artifacts/narration.ssml":      StoryDir + "/narration.ssml",
	"artifacts/script.json":         StoryDir + "/script.json",
	"artifacts/script.md":           StoryDir + "/script.md",
	"artifacts/story-spine.json":    StoryDir + "/story-spine.json",
	"artifacts/plot-input.md":       StoryDir + "/plot-input.md",
	"artifacts/creative-options.json": StoryDir + "/creative-options.json",
	"artifacts/master.mp4":          MediaDir + "/master.mp4",
	"artifacts/narration.mp3":       MediaDir + "/narration.mp3",
	"artifacts/subtitles.ass":       MediaDir + "/subtitles.ass",
	"artifacts/timeline.json":       MediaDir + "/timeline.json",
	"artifacts/sync-report.json":    MediaDir + "/sync-report.json",
	"artifacts/audio_segments.json": MediaDir + "/audio_segments.json",
	"artifacts/bgm-plan.json":       MediaDir + "/bgm-plan.json",
}

func normalizeRel(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	rel = strings.TrimPrefix(rel, "./")
	return rel
}

// CanonicalWriteRel 返回新布局下的写入相对路径。
func CanonicalWriteRel(rel string) string {
	rel = normalizeRel(rel)
	if c, ok := canonicalWrites[rel]; ok {
		return c
	}
	return rel
}

func readCandidates(rel string) []string {
	rel = normalizeRel(rel)
	if c, ok := canonicalWrites[rel]; ok {
		return []string{c, rel}
	}
	return []string{rel}
}

// ResolvePath 解析产物路径：优先已存在的新/旧路径，否则返回新布局写入路径。
func ResolvePath(runDir, rel string) string {
	rel = normalizeRel(rel)
	for _, c := range readCandidates(rel) {
		p := filepath.Join(runDir, c)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(runDir, CanonicalWriteRel(rel))
}

// FileExists 检查产物是否存在（兼容新旧路径）。
func FileExists(runDir, rel string) bool {
	rel = normalizeRel(rel)
	for _, c := range readCandidates(rel) {
		if _, err := os.Stat(filepath.Join(runDir, c)); err == nil {
			return true
		}
	}
	return false
}

// ShotImageRel 单镜关键帧相对路径。
func ShotImageRel(shotID string) string {
	return filepath.ToSlash(filepath.Join(ShotsDir, shotID+".png"))
}

// ShotVideoRel 单镜视频相对路径。
func ShotVideoRel(shotID string) string {
	return filepath.ToSlash(filepath.Join(ShotsDir, shotID+".mp4"))
}

// ResolveScriptPath 读取剧本文本（script.md 优先，兼容 chapter.md）。
func ResolveScriptPath(runDir string) string {
	candidates := []string{
		StoryDir + "/script.md",
		"artifacts/script.md",
		"artifacts/chapter.md",
	}
	for _, c := range candidates {
		p := filepath.Join(runDir, c)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(runDir, StoryDir, "script.md")
}
