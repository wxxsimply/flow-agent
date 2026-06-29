package ffmpeg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ffmpegBin 返回 ffmpeg 可执行文件路径；空表示未找到。
func ffmpegBin() string {
	return resolveTool("ffmpeg", "FFMPEG_PATH")
}

// ffprobeBin 返回 ffprobe 可执行文件路径。
func ffprobeBin() string {
	return resolveTool("ffprobe", "FFPROBE_PATH")
}

func resolveTool(name, envKey string) string {
	if p := strings.TrimSpace(os.Getenv(envKey)); p != "" {
		if fileExists(p) {
			return p
		}
	}
	if p := findProjectTool(name); p != "" {
		return p
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Windows：winget Gyan.FFmpeg 常通过 Links 暴露，部分 IDE 终端 PATH 未刷新
	if filepath.Ext(name) == "" {
		if p := findWindowsTool(name + ".exe"); p != "" {
			return p
		}
	}
	return ""
}

// findProjectTool 查找项目根目录下 ffmpeg/bin/<tool>（与 go.mod 同级）。
func findProjectTool(name string) string {
	root := findProjectRoot()
	if root == "" {
		return ""
	}
	exe := name
	if filepath.Ext(exe) == "" {
		exe += ".exe"
	}
	p := filepath.Join(root, "ffmpeg", "bin", exe)
	if fileExists(p) {
		return p
	}
	return ""
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findWindowsTool(exe string) string {
	candidates := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WinGet", "Links", exe),
		filepath.Join(os.Getenv("ProgramFiles"), "ffmpeg", "bin", exe),
		filepath.Join(os.Getenv("ProgramFiles"), "Gyan", "FFmpeg", "bin", exe),
		filepath.Join(`C:\`, "ffmpeg", "bin", exe),
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	// WinGet Packages 目录：Gyan.FFmpeg_*\ffmpeg-*-full_build\bin\ffmpeg.exe
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		packages := filepath.Join(local, "Microsoft", "WinGet", "Packages")
		if entries, err := os.ReadDir(packages); err == nil {
			for _, e := range entries {
				if !e.IsDir() || !strings.Contains(strings.ToLower(e.Name()), "ffmpeg") {
					continue
				}
				root := filepath.Join(packages, e.Name())
				var found string
				_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
					if err != nil || found != "" {
						return nil
					}
					if !d.IsDir() && strings.EqualFold(filepath.Base(path), exe) {
						found = path
						return filepath.SkipAll
					}
					return nil
				})
				if found != "" {
					return found
				}
			}
		}
	}
	// 用 where.exe 查系统 PATH（子进程继承的 PATH 可能比当前 Go 进程更全）
	if out, err := exec.Command("where.exe", exe).Output(); err == nil {
		line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
		if fileExists(line) {
			return line
		}
	}
	return ""
}
