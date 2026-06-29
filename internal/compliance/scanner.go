package compliance

import (
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// TextSource 待扫描文本片段。
type TextSource struct {
	Name string
	Text string
}

// ScanSources 对多个文本源做词库匹配。
func ScanSources(entries []WordEntry, sources []TextSource) *artifacts.ComplianceReport {
	report := &artifacts.ComplianceReport{
		Blocks:   []artifacts.ComplianceIssue{},
		Warnings: []artifacts.ComplianceIssue{},
	}
	if len(entries) == 0 {
		report.Recount()
		return report
	}

	seen := map[string]bool{}
	for _, src := range sources {
		text := src.Text
		if strings.TrimSpace(text) == "" {
			continue
		}
		lower := strings.ToLower(text)
		for _, e := range entries {
			term := e.Term
			haystack := text
			if isASCII(term) {
				haystack = lower
				term = strings.ToLower(term)
			}
			idx := strings.Index(haystack, term)
			if idx < 0 {
				continue
			}
			key := e.Severity + "|" + e.Term + "|" + src.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			issue := artifacts.ComplianceIssue{
				Severity: e.Severity,
				Word:     e.Term,
				Source:   src.Name,
				Snippet:  snippetAround(text, idx, len(e.Term)),
			}
			if e.Severity == "warning" {
				report.Warnings = append(report.Warnings, issue)
			} else {
				report.Blocks = append(report.Blocks, issue)
			}
		}
	}
	report.Recount()
	return report
}

func snippetAround(text string, idx, termLen int) string {
	const radius = 24
	runes := []rune(text)
	start := runeIndexAt(text, idx)
	if start < 0 {
		start = 0
	}
	end := runeIndexAt(text, idx+termLen)
	if end < 0 {
		end = len(runes)
	}
	from := start - radius
	if from < 0 {
		from = 0
	}
	to := end + radius
	if to > len(runes) {
		to = len(runes)
	}
	s := string(runes[from:to])
	s = strings.TrimSpace(s)
	if from > 0 {
		s = "…" + s
	}
	if to < len(runes) {
		s = s + "…"
	}
	return s
}

func runeIndexAt(text string, byteIdx int) int {
	if byteIdx <= 0 {
		return 0
	}
	if byteIdx >= len(text) {
		return len([]rune(text))
	}
	return len([]rune(text[:byteIdx]))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
