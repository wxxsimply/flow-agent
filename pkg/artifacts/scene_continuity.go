package artifacts

import "strings"

// ScenesSimilar 粗判两场景描述是否同一地点（供连贯性审查）。
func ScenesSimilar(a, b string) bool {
	a = normalizeSceneText(a)
	b = normalizeSceneText(b)
	if a == b {
		return true
	}
	if a == "" || b == "" {
		return false
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	ta := sceneTokens(a)
	tb := sceneTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	overlap := 0
	for t := range ta {
		if tb[t] {
			overlap++
		}
	}
	minLen := len(ta)
	if len(tb) < minLen {
		minLen = len(tb)
	}
	return overlap >= 1 && float64(overlap)/float64(minLen) >= 0.34
}

func normalizeSceneText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("，", " ", "。", " ", "、", " ", "；", " ", "：", " ", ",", " ", ".", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func sceneTokens(s string) map[string]bool {
	stop := map[string]bool{
		"的": true, "在": true, "与": true, "和": true, "中": true, "里": true, "上": true, "下": true,
	}
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		w = strings.TrimSpace(w)
		if len([]rune(w)) < 2 || stop[w] {
			continue
		}
		out[w] = true
	}
	return out
}

// SceneChanged 判断是否场景切换。
func SceneChanged(prev, curr Shot) bool {
	a := strings.TrimSpace(prev.SceneBackground)
	b := strings.TrimSpace(curr.SceneBackground)
	if a != "" && b != "" {
		return !ScenesSimilar(a, b)
	}
	return !ScenesSimilar(truncateRunesScene(prev.VisualPrompt, 48), truncateRunesScene(curr.VisualPrompt, 48))
}

func truncateRunesScene(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max])
}
