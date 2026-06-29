package agent

import "strings"

// ExtractTopLevelJSON 从可能包含 markdown / 多余文本的输出里提取首个完整 JSON 对象。
func ExtractTopLevelJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	start := strings.Index(raw, "{")
	if start < 0 {
		return raw
	}
	depth := 0
	inStr := false
	escape := false
	for i := start; i < len(raw); i++ {
		c := raw[i]
		if inStr {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[start : i+1]
			}
		}
	}
	return raw[start:]
}
