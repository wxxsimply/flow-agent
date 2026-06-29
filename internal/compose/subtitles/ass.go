package subtitles

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// Style 竖屏 ASS 字幕样式。
type Style struct {
	MaxCharsPerLine int
	MaxLines        int
	FontSize        int
	MarginV         int
}

// DefaultVerticalStyle 抖音竖屏 1080×1920 默认（约 14 字 × 3 行）。
func DefaultVerticalStyle() Style {
	return Style{MaxCharsPerLine: 14, MaxLines: 3, FontSize: 44, MarginV: 200}
}

// DefaultLandscapeStyle 横屏 1920×1080 默认。
func DefaultLandscapeStyle() Style {
	return Style{MaxCharsPerLine: 20, MaxLines: 2, FontSize: 40, MarginV: 80}
}

// StyleForResolution 按 timeline 分辨率选择字幕样式。
func StyleForResolution(resolution string) Style {
	if strings.Contains(resolution, "1920x1080") {
		return DefaultLandscapeStyle()
	}
	return DefaultVerticalStyle()
}

// PlayResForResolution 返回 ASS PlayRes 宽高。
func PlayResForResolution(resolution string) (width, height int) {
	if strings.Contains(resolution, "1920x1080") {
		return 1920, 1080
	}
	return 1080, 1920
}

// Event 一条 ASS 字幕事件。
type Event struct {
	StartSec float64
	EndSec   float64
	Text     string
}

// WriteASS 生成多行 ASS 字幕（PlayRes 与成片分辨率对齐）。
func WriteASS(path string, events []Event, style Style, playResX, playResY int) error {
	if style.MaxCharsPerLine <= 0 {
		style.MaxCharsPerLine = 14
	}
	if style.MaxLines <= 0 {
		style.MaxLines = 3
	}
	if style.FontSize <= 0 {
		style.FontSize = 44
	}
	if style.MarginV <= 0 {
		style.MarginV = 200
	}
	if playResX <= 0 {
		playResX = 1080
	}
	if playResY <= 0 {
		playResY = 1920
	}
	var b strings.Builder
	b.WriteString("[Script Info]\n")
	b.WriteString("Title: FlowAgent\n")
	b.WriteString("ScriptType: v4.00+\n")
	b.WriteString(fmt.Sprintf("PlayResX: %d\n", playResX))
	b.WriteString(fmt.Sprintf("PlayResY: %d\n", playResY))
	b.WriteString("WrapStyle: 0\n")
	b.WriteString("\n")
	b.WriteString("[V4+ Styles]\n")
	b.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	b.WriteString(fmt.Sprintf("Style: Default,Microsoft YaHei,%d,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,3,1,2,48,48,%d,1\n",
		style.FontSize, style.MarginV))
	b.WriteString("\n")
	b.WriteString("[Events]\n")
	b.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	for _, ev := range events {
		text := WrapVerticalLines(ev.Text, style.MaxCharsPerLine, style.MaxLines)
		text = escapeASSText(text)
		if text == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			formatASSTime(ev.StartSec), formatASSTime(ev.EndSec), text))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WrapVerticalLines 将长句折成竖屏多行（ASS 用 \\N 换行）。
func WrapVerticalLines(text string, maxChars, maxLines int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	parts := splitNarrationLines(text)
	var lines []string
	for _, p := range parts {
		lines = append(lines, wrapRunes(p, maxChars, maxLines)...)
	}
	if len(lines) == 0 {
		lines = wrapRunes(text, maxChars, maxLines)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, `\N`)
}

func wrapRunes(text string, maxChars, maxLines int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	var lines []string
	for len(runes) > 0 && len(lines) < maxLines {
		n := maxChars
		if n > len(runes) {
			n = len(runes)
		}
		// 尽量在标点处断行
		cut := n
		if len(runes) > n {
			for i := n; i > n/2; i-- {
				if isBreakRune(runes[i-1]) {
					cut = i
					break
				}
			}
		}
		lines = append(lines, string(runes[:cut]))
		runes = runes[cut:]
	}
	if len(runes) > 0 && len(lines) == maxLines {
		lines[maxLines-1] = string([]rune(lines[maxLines-1])[:maxChars])
	}
	return lines
}

func isBreakRune(r rune) bool {
	switch r {
	case '，', '。', '！', '？', '；', '、', '：', '"', '」', '』', ' ':
		return true
	}
	return false
}

// EventsFromSegments 按镜旁白生成字幕事件；过长 narration 按标点拆条并多行折行。
func EventsFromSegments(segments []artifacts.AudioSegment, narrations map[string]string, style Style) []Event {
	var out []Event
	for _, seg := range segments {
		text := strings.TrimSpace(narrations[seg.ShotID])
		if text == "" {
			continue
		}
		parts := dedupeSubtitleParts(splitNarrationLines(text))
		if len(parts) <= 1 {
			out = append(out, Event{
				StartSec: seg.StartSec,
				EndSec:   seg.StartSec + seg.DurationSec,
				Text:     text,
			})
			continue
		}
		var weights []int
		var total int
		for _, p := range parts {
			n := utf8.RuneCountInString(p)
			if n < 1 {
				n = 1
			}
			weights = append(weights, n)
			total += n
		}
		start := seg.StartSec
		dur := seg.DurationSec
		for i, p := range parts {
			partDur := dur * float64(weights[i]) / float64(total)
			if i == len(parts)-1 {
				partDur = seg.StartSec + dur - start
			}
			ev := Event{
				StartSec: start,
				EndSec:   start + partDur,
				Text:     p,
			}
			out = appendSubtitleEvent(out, ev)
			start += partDur
		}
	}
	return out
}

func appendSubtitleEvent(out []Event, ev Event) []Event {
	text := strings.TrimSpace(ev.Text)
	if text == "" {
		return out
	}
	if len(out) > 0 {
		prev := strings.TrimSpace(out[len(out)-1].Text)
		if subtitleTextSame(prev, text) {
			return out
		}
	}
	return append(out, ev)
}

func dedupeSubtitleParts(parts []string) []string {
	var out []string
	var prevNorm string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n := normalizeSubtitleKey(p)
		if n != "" && n == prevNorm {
			continue
		}
		out = append(out, p)
		prevNorm = n
	}
	return out
}

func normalizeSubtitleKey(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '，', '。', '！', '？', '；', '、':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func subtitleTextSame(a, b string) bool {
	na, nb := normalizeSubtitleKey(a), normalizeSubtitleKey(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb || (len([]rune(na)) >= 6 && strings.Contains(nb, na)) || strings.Contains(na, nb)
}

func splitNarrationLines(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	for _, sep := range []string{"。", "！", "？", "；", "\n"} {
		if strings.Contains(text, sep) {
			var parts []string
			for _, p := range strings.Split(text, sep) {
				p = strings.TrimSpace(p)
				if p != "" {
					parts = append(parts, p+strings.TrimSpace(sep))
				}
			}
			parts = filterSubtitleParts(parts)
			if len(parts) > 1 {
				return parts
			}
		}
	}
	if utf8.RuneCountInString(text) > 28 {
		return splitLongLineAtComma(text, 28)
	}
	return []string{text}
}

func filterSubtitleParts(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || isPunctuationOnly(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func isPunctuationOnly(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	for _, r := range s {
		switch r {
		case '，', '。', '！', '？', '；', '、', '：', ',', '.', '!', '?', ' ':
			continue
		default:
			return false
		}
	}
	return true
}

func splitLongLineAtComma(text string, maxRunes int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return []string{text}
	}
	var parts []string
	start := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] != '，' && runes[i] != ',' {
			continue
		}
		if i+1-start > maxRunes && i > start {
			parts = append(parts, string(runes[start:i+1]))
			start = i + 1
		}
	}
	if start < len(runes) {
		parts = append(parts, string(runes[start:]))
	}
	if len(parts) == 0 {
		return wrapRunesAsParts(text, maxRunes)
	}
	return parts
}

func wrapRunesAsParts(text string, chunk int) []string {
	runes := []rune(text)
	var parts []string
	for i := 0; i < len(runes); i += chunk {
		end := i + chunk
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[i:end]))
	}
	return parts
}

func formatASSTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func escapeASSText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", `\N`)
	s = strings.ReplaceAll(s, "\n", `\N`)
	s = strings.ReplaceAll(s, "{", `\{`)
	return s
}
