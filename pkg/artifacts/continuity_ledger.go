package artifacts

import (
	"fmt"
	"strings"
)

// BuildContinuityLedger 从分镜规则生成跨镜连续性台账 markdown。
func BuildContinuityLedger(sb *Storyboard, brief string, propSheets *PropSheets) string {
	if sb == nil || len(sb.Shots) == 0 {
		return "# Continuity Ledger\n\n(empty storyboard)\n"
	}
	var b strings.Builder
	b.WriteString("# Continuity Ledger\n\n")
	b.WriteString("> 由 assemble 自动生成，供扩写续写与 produce 锁定外观。\n\n")

	if brief != "" {
		tail := brief
		if r := []rune(brief); len(r) > 400 {
			tail = string(r[:200]) + "\n…\n" + string(r[len(r)-200:])
		}
		b.WriteString("## 故事摘要\n\n")
		b.WriteString(tail)
		b.WriteString("\n\n")
	}

	lock := extractLockWords(sb.Shots[0].VisualPrompt)
	b.WriteString("## 角色：主角\n\n")
	b.WriteString("- 锁定词：`")
	b.WriteString(lock)
	b.WriteString("`\n")
	b.WriteString("- 来源镜：")
	b.WriteString(sb.Shots[0].ID)
	b.WriteString("\n- 本集换装：否（除非剧情明示）\n\n")

	if propSheets != nil && len(propSheets.Props) > 0 {
		b.WriteString("## 物体三视图\n\n")
		for _, pe := range propSheets.Props {
			b.WriteString(fmt.Sprintf("- **%s** (`%s`, %s)：%s\n", pe.Name, pe.ID, pe.Category, pe.Appearance))
			if pe.TurnaroundViews != nil {
				if pe.TurnaroundViews.Front != "" {
					b.WriteString("  - 正面：" + pe.TurnaroundViews.Front + "\n")
				}
				if pe.TurnaroundViews.Side != "" {
					b.WriteString("  - 侧面：" + pe.TurnaroundViews.Side + "\n")
				}
				if pe.TurnaroundViews.Back != "" {
					b.WriteString("  - 背面：" + pe.TurnaroundViews.Back + "\n")
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## 镜头列表\n\n")
	b.WriteString("| 镜号 | 景别 | physics_cues（摘要） | narration（前20字） |\n")
	b.WriteString("|------|------|----------------------|---------------------|\n")
	for _, sh := range sb.Shots {
		narr := sh.Narration
		if r := []rune(narr); len(r) > 20 {
			narr = string(r[:20]) + "…"
		}
		pc := sh.PhysicsCues
		if r := []rune(pc); len(r) > 40 {
			pc = string(r[:40]) + "…"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			sh.ID, NormalizeShotSize(sh.ShotSize), pc, narr))
	}
	b.WriteString("\n## 屏幕方向\n\n")
	b.WriteString("- 默认：建立镜后保持角色行进方向一致，跳轴须剧情明示。\n")

	if hasProps := sb.shotPropTable(propSheets); hasProps != "" {
		b.WriteString("\n## 各镜道具（produce 按镜 [PROP_LOCK] / [PROP_VIEW_LOCK] 注入）\n\n")
		b.WriteString(hasProps)
	}
	return b.String()
}

func (sb *Storyboard) shotPropTable(propSheets *PropSheets) string {
	var rows []string
	for _, sh := range sb.Shots {
		p := strings.TrimSpace(sh.HeldProps.String())
		if p == "" && len(sh.PropRefs) == 0 {
			continue
		}
		line := fmt.Sprintf("- %s: %s", sh.ID, p)
		if len(sh.PropRefs) > 0 {
			line += fmt.Sprintf(" → refs: %s", strings.Join(sh.PropRefs, ", "))
		}
		if propSheets != nil {
			for _, ref := range sh.PropRefs {
				if pe := propSheets.PropByID(ref); pe != nil {
					line += fmt.Sprintf(" (%s)", pe.Name)
				}
			}
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		return ""
	}
	return strings.Join(rows, "\n") + "\n"
}

func extractLockWords(visual string) string {
	visual = strings.TrimSpace(visual)
	if visual == "" {
		return "（见各镜 visual_prompt）"
	}
	if r := []rune(visual); len(r) > 120 {
		return string(r[:120]) + "…"
	}
	return visual
}

// BuildLightCharacterSheets 从 s01 生成轻量 character-sheets（director 无 character 阶段时）。
func BuildLightCharacterSheets(sb *Storyboard) *CharacterSheets {
	if sb == nil || len(sb.Shots) == 0 {
		return nil
	}
	app := extractLockWords(sb.Shots[0].VisualPrompt)
	if app == "（见各镜 visual_prompt）" {
		app = strings.TrimSpace(sb.Shots[0].VisualPrompt)
	}
	app += "；每镜仅一名主角入画，禁止重复分身与多余武器"
	return &CharacterSheets{
		Characters: []CharacterSheetEntry{{
			Name:       "主角",
			Appearance: app,
		}},
	}
}
