package artifacts

import (
	"strings"
	"unicode"
)

// NarrationAlignmentScore 检查各镜 narration 是否能在 chapter 正文中找到（子串匹配）。
// 返回 score ∈ [0,1] 与未命中的 narration 列表。
func NarrationAlignmentScore(chapter string, sb *Storyboard) (score float64, missing []string) {
	if sb == nil || len(sb.Shots) == 0 {
		return 0, nil
	}
	chNorm := normalizeNarrationText(chapter)
	if chNorm == "" {
		return 0, nil
	}
	hits := 0
	for _, shot := range sb.Shots {
		n := normalizeNarrationText(shot.Narration)
		if n == "" {
			missing = append(missing, shot.Narration)
			continue
		}
		if strings.Contains(chNorm, n) {
			hits++
			continue
		}
		// 取 narration 前 12 字再试（容忍 LLM 轻微改写）
		runes := []rune(n)
		if len(runes) >= 8 {
			prefix := string(runes[:min(12, len(runes))])
			if strings.Contains(chNorm, prefix) {
				hits++
				continue
			}
		}
		missing = append(missing, shot.Narration)
	}
	return float64(hits) / float64(len(sb.Shots)), missing
}

func normalizeNarrationText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
