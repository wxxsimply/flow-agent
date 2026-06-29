package artifacts

import (
	"strings"
	"unicode/utf8"
)

// 180 秒档正文字数基准（与历史门禁一致）。
const (
	chapterBaseDurationSec = 180
	chapterBaseMinChars      = 600
	chapterBaseMaxChars      = 1800
)

// ChapterCharBounds 按目标视频时长返回 chapter 正文字数允许范围（约 3.3～10 字/秒）。
func ChapterCharBounds(targetSec int, _ *HookPlan) (min, max int) {
	if targetSec <= 0 {
		targetSec = chapterBaseDurationSec
	}
	min = targetSec * chapterBaseMinChars / chapterBaseDurationSec
	max = targetSec * chapterBaseMaxChars / chapterBaseDurationSec
	if min < 250 {
		min = 250
	}
	if max < min+150 {
		max = min + 150
	}
	return min, max
}

// CountChapterBodyRunes 统计 chapter 正文（去掉 scene 标题行与 dry-run 标记）。
func CountChapterBodyRunes(markdown string) int {
	text := stripDryRunComment(markdown)
	var body strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "##") {
			continue
		}
		body.WriteString(trim)
	}
	return utf8.RuneCountInString(body.String())
}

func stripDryRunComment(s string) string {
	const prefix = "<!-- dry-run -->\n"
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
