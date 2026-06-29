package artifacts

import "strings"

// AssignShotTiers 为分镜标注 hero/standard（tiered 制片用）。
func AssignShotTiers(sb *Storyboard, heroCount int) {
	if sb == nil || len(sb.Shots) == 0 {
		return
	}
	if heroCount <= 0 {
		heroCount = 3
	}
	if heroCount > len(sb.Shots) {
		heroCount = len(sb.Shots)
	}
	for i := range sb.Shots {
		if strings.TrimSpace(sb.Shots[i].Tier) != "" {
			continue
		}
		sb.Shots[i].Tier = "standard"
	}
	heroIdx := make(map[int]bool)
	heroIdx[0] = true
	if len(sb.Shots) > 1 {
		heroIdx[len(sb.Shots)-1] = true
	}
	// 中间再选一镜作为高潮（偏后）
	if heroCount > 2 && len(sb.Shots) > 3 {
		mid := len(sb.Shots) * 2 / 3
		if mid > 0 && mid < len(sb.Shots)-1 {
			heroIdx[mid] = true
		}
	}
	for i := range sb.Shots {
		if heroIdx[i] {
			sb.Shots[i].Tier = "hero"
		}
	}
	// 若 heroCount 更大，从后往前补 standard→hero
	for n := 0; n < len(sb.Shots) && countHero(sb.Shots) < heroCount; n++ {
		i := len(sb.Shots) - 1 - n
		if sb.Shots[i].Tier != "hero" {
			sb.Shots[i].Tier = "hero"
		}
	}
}

func countHero(shots []Shot) int {
	c := 0
	for _, s := range shots {
		if strings.EqualFold(strings.TrimSpace(s.Tier), "hero") {
			c++
		}
	}
	return c
}

// IsHeroShot 是否高潮/锚点镜。
func IsHeroShot(shot Shot) bool {
	return strings.EqualFold(strings.TrimSpace(shot.Tier), "hero")
}
