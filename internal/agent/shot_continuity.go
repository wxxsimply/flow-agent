package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// shouldLinkToPreviousShot 是否将上一镜信息注入本镜 prompt（同场景弱关联；换场景则独立成镜）。
func shouldLinkToPreviousShot(prev, curr artifacts.Shot) bool {
	if strings.TrimSpace(prev.ID) == "" {
		return false
	}
	return !sceneChanged(prev, curr)
}

// sceneChanged 判断是否为场景切换（允许硬切、不复用上一镜构图/末帧）。
func sceneChanged(prev, curr artifacts.Shot) bool {
	a := strings.TrimSpace(prev.SceneBackground)
	b := strings.TrimSpace(curr.SceneBackground)
	if a != "" && b != "" {
		return !scenesSimilar(a, b)
	}
	return !visualPromptsSameScene(prev.VisualPrompt, curr.VisualPrompt)
}

func scenesSimilar(a, b string) bool {
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
		"the": true, "a": true, "an": true, "in": true, "on": true, "at": true,
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

func visualPromptsSameScene(prevVis, currVis string) bool {
	prevVis = strings.TrimSpace(prevVis)
	currVis = strings.TrimSpace(currVis)
	if prevVis == "" || currVis == "" {
		return true
	}
	// 取前段环境描述做粗比对
	prevHead := truncateRunes(prevVis, 48)
	currHead := truncateRunes(currVis, 48)
	return scenesSimilar(prevHead, currHead)
}

// crossShotContinuitySuffix 镜间 prompt 后缀：同场景弱关联；换场景允许硬切。
func crossShotContinuitySuffix(prev, curr artifacts.Shot) string {
	if strings.TrimSpace(prev.ID) == "" {
		return ""
	}
	if sceneChanged(prev, curr) {
		return "，同一主角与服装延续，本镜为新场景/新构图，可与上一镜硬切，禁止复制上一镜背景与结束姿态，禁止换人"
	}
	return "，同一场景内切镜：角色/服装一致，可换景别机位运镜，不必承接上一镜结束画面，禁止换人"
}

// crossShotContinuitySuffixHard 硬衔接：需承接上一镜末态（仅 chain_shot_mode=hard）。
func crossShotContinuitySuffixHard(prev, curr artifacts.Shot) string {
	end := lastActionBeat(prev)
	if end == "" {
		end = trimRunes(strings.TrimSpace(prev.VisualPrompt), 60)
	}
	base := crossShotContinuitySuffix(prev, curr)
	if end == "" {
		return base + "，动作从上一镜末态自然延续"
	}
	_ = curr
	return base + "，从上一镜末态「" + end + "」延续"
}

func trimRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func lastActionBeat(shot artifacts.Shot) string {
	for i := len(shot.ActionBeats) - 1; i >= 0; i-- {
		if b := strings.TrimSpace(shot.ActionBeats[i]); b != "" {
			return b
		}
	}
	return ""
}

