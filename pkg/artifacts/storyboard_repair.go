package artifacts

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// RepairShots 补全 LLM 分镜里缺失的 narration / subtitle / visual_prompt。
// scriptText 通常为 script.md 或 chapter.md 全文；scriptLines 为各场旁白（优先）。
// 返回修复的镜数。
func (s *Storyboard) RepairShots(scriptText string, scriptLines []string) int {
	if s == nil || len(s.Shots) == 0 {
		return 0
	}
	fixed := 0

	for i := range s.Shots {
		shot := &s.Shots[i]
		if strings.TrimSpace(shot.Narration) == "" {
			if sub := strings.TrimSpace(shot.Subtitle); sub != "" {
				shot.Narration = sub
				fixed++
			}
		}
		if strings.TrimSpace(shot.Subtitle) == "" {
			if n := strings.TrimSpace(shot.Narration); n != "" {
				shot.Subtitle = truncateRunes(n, 28)
			}
		}
		if strings.TrimSpace(shot.VisualPrompt) == "" {
			shot.VisualPrompt = "竖屏9:16，电影感画面，人物与环境清晰"
			fixed++
		}
		if shot.VisualType == "" {
			shot.VisualType = "ai_video"
		}
	}

	// 按场戏旁白依次填入空镜
	lineIdx := 0
	for i := range s.Shots {
		if strings.TrimSpace(s.Shots[i].Narration) != "" {
			continue
		}
		for lineIdx < len(scriptLines) {
			candidate := strings.TrimSpace(scriptLines[lineIdx])
			lineIdx++
			if candidate == "" {
				continue
			}
			s.Shots[i].Narration = candidate
			if strings.TrimSpace(s.Shots[i].Subtitle) == "" {
				s.Shots[i].Subtitle = truncateRunes(candidate, 28)
			}
			fixed++
			break
		}
	}

	// 从剧本文本切分补空镜
	var empty []int
	for i := range s.Shots {
		if strings.TrimSpace(s.Shots[i].Narration) == "" {
			empty = append(empty, i)
		}
	}
	if len(empty) > 0 {
		chunks := splitScriptIntoChunks(scriptText, len(empty))
		for j, idx := range empty {
			if j >= len(chunks) || strings.TrimSpace(chunks[j]) == "" {
				continue
			}
			s.Shots[idx].Narration = chunks[j]
			if strings.TrimSpace(s.Shots[idx].Subtitle) == "" {
				s.Shots[idx].Subtitle = truncateRunes(chunks[j], 28)
			}
			fixed++
		}
	}

	// 仍空：从画面描述/action 生成独立旁白，禁止复制上一镜
	for i := range s.Shots {
		if strings.TrimSpace(s.Shots[i].Narration) != "" {
			continue
		}
		prev := ""
		if i > 0 {
			prev = s.Shots[i-1].Narration
		}
		s.Shots[i].Narration = UniqueNarrationForShot(s.Shots[i], prev, i+1)
		if strings.TrimSpace(s.Shots[i].Subtitle) == "" {
			s.Shots[i].Subtitle = truncateRunes(s.Shots[i].Narration, 28)
		}
		fixed++
	}

	s.DedupeNarrations()
	return fixed
}

func splitScriptIntoChunks(script string, n int) []string {
	if n <= 0 {
		return nil
	}
	text := extractSpokenText(script)
	if text == "" {
		return nil
	}
	// 先按段落
	paras := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	var parts []string
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "**旁白：**")
		p = strings.TrimPrefix(p, "旁白：")
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) >= n {
		return parts[:n]
	}
	// 不足则按句号切分再合并成 n 段
	sents := regexp.MustCompile(`[。！？.!?]+`).Split(text, -1)
	var merged []string
	var buf strings.Builder
	per := (len(sents) + n - 1) / n
	if per < 1 {
		per = 1
	}
	count := 0
	for _, sent := range sents {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString("。")
		}
		buf.WriteString(sent)
		count++
		if count >= per {
			merged = append(merged, buf.String())
			buf.Reset()
			count = 0
			if len(merged) >= n {
				break
			}
		}
	}
	if buf.Len() > 0 && len(merged) < n {
		merged = append(merged, buf.String())
	}
	for len(merged) < n {
		merged = append(merged, "")
	}
	return merged
}

func extractSpokenText(script string) string {
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "**旁白：**") {
			b.WriteString(strings.TrimPrefix(line, "**旁白：**"))
			b.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "旁白：") {
			b.WriteString(strings.TrimPrefix(line, "旁白："))
			b.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "##") {
			continue
		}
		if strings.HasPrefix(line, ">") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
