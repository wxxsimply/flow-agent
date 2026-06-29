package artifacts

import (
	"strings"
	"unicode/utf8"
)

// DedupeNarrations 去除各镜旁白/字幕重复，返回修改镜数。
func (s *Storyboard) DedupeNarrations() int {
	if s == nil || len(s.Shots) == 0 {
		return 0
	}
	fixed := 0
	var prev string
	for i := range s.Shots {
		shot := &s.Shots[i]
		narr := strings.TrimSpace(shot.Narration)
		if narr == "" {
			continue
		}
		if prev != "" && NarrationsTooSimilar(narr, prev) {
			narr = UniqueNarrationForShot(*shot, prev, i+1)
			shot.Narration = narr
			fixed++
		}
		// 同镜内重复句（旁白里连着两句一样）
		shot.Narration = DedupeSentencesInText(narr)
		if sub := strings.TrimSpace(shot.Subtitle); sub != "" && NarrationsTooSimilar(sub, shot.Narration) {
			shot.Subtitle = truncateRunes(shot.Narration, 24)
		} else if strings.TrimSpace(shot.Subtitle) == "" {
			shot.Subtitle = truncateRunes(shot.Narration, 24)
		}
		prev = shot.Narration
	}
	return fixed
}

// NarrationsTooSimilar 判断两句旁白是否应视为重复。
func NarrationsTooSimilar(a, b string) bool {
	na, nb := normalizeNarrationKey(a), normalizeNarrationKey(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	// 一方包含另一方（较长句）
	short, long := na, nb
	if len([]rune(short)) > len([]rune(long)) {
		short, long = long, short
	}
	if len([]rune(short)) >= 6 && strings.Contains(long, short) {
		return true
	}
	return narrationOverlapRatio(na, nb) >= 0.82
}

func normalizeNarrationKey(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '，', '。', '！', '？', '；', '、', '：', '"', '\u2018', '\u2019', '…', '.', ',', '!', '?':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func narrationOverlapRatio(a, b string) float64 {
	if a == b {
		return 1
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	// 前缀相同比例
	n := len(ra)
	if len(rb) < n {
		n = len(rb)
	}
	same := 0
	for i := 0; i < n; i++ {
		if ra[i] == rb[i] {
			same++
		}
	}
	return float64(same) / float64(n)
}

// UniqueNarrationForShot 为重复旁白生成替代句。
func UniqueNarrationForShot(shot Shot, prevNarr string, shotNo int) string {
	for _, beat := range shot.ActionBeats {
		c := trimBeatForNarration(beat)
		if c != "" && !NarrationsTooSimilar(c, prevNarr) {
			return c
		}
	}
	if vp := strings.TrimSpace(shot.VisualPrompt); vp != "" {
		for _, chunk := range extractNarrationCandidates(vp) {
			if !NarrationsTooSimilar(chunk, prevNarr) {
				return chunk
			}
		}
	}
	if vp := strings.TrimSpace(shot.VisualPrompt); vp != "" {
		return truncateRunes(vp, 45)
	}
	return truncateRunes("故事在此继续推进。", 20)
}

func trimBeatForNarration(beat string) string {
	beat = strings.TrimSpace(beat)
	if beat == "" {
		return ""
	}
	// 去掉常见动作模板后缀
	for _, suf := range []string{"，动作起始姿态", "，动作进行中", "，动作结束姿态", "，肢体稳定", "，小幅位移"} {
		if idx := strings.Index(beat, suf); idx > 8 {
			beat = beat[:idx]
			break
		}
	}
	return truncateRunes(beat, 45)
}

func extractNarrationCandidates(visual string) []string {
	visual = strings.TrimSpace(visual)
	if visual == "" {
		return nil
	}
	var out []string
	for _, p := range strings.FieldsFunc(visual, func(r rune) bool {
		return r == '，' || r == '。' || r == '；' || r == '\n'
	}) {
		p = strings.TrimSpace(p)
		if utf8.RuneCountInString(p) >= 8 {
			out = append(out, truncateRunes(p, 45))
		}
	}
	r := []rune(visual)
	if len(r) > 12 {
		mid := len(r) / 3
		out = append(out, truncateRunes(string(r[mid:]), 45))
	}
	out = append(out, truncateRunes(visual, 45))
	return out
}

// DedupeSentencesInText 去掉同一段旁白内重复句子。
func DedupeSentencesInText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	parts := splitBySentence(text)
	if len(parts) <= 1 {
		return text
	}
	var kept []string
	var prevNorm string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n := normalizeNarrationKey(p)
		if n != "" && n == prevNorm {
			continue
		}
		kept = append(kept, p)
		prevNorm = n
	}
	if len(kept) == 0 {
		return text
	}
	return strings.Join(kept, "")
}

func splitBySentence(text string) []string {
	var parts []string
	var buf strings.Builder
	for _, r := range text {
		buf.WriteRune(r)
		switch r {
		case '。', '！', '？', '；', '\n':
			parts = append(parts, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}
