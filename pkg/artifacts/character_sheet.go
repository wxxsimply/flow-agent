package artifacts

import (
	"encoding/json"
	"os"
	"strings"
)

// CharacterSheetEntry 单角色三视图设定。
type CharacterSheetEntry struct {
	Name             string              `json:"name"`
	Appearance       string              `json:"appearance"`
	TurnaroundPath   string              `json:"turnaround_path,omitempty"` // 正面图（兼容旧字段）
	TurnaroundPrompt string              `json:"turnaround_prompt,omitempty"`
	TurnaroundViews  *CharacterViewPaths `json:"turnaround_views,omitempty"`
}

// CharacterViewPaths 角色三视图分图路径（正面/侧面/背面各一张）。
type CharacterViewPaths struct {
	Front string `json:"front,omitempty"`
	Side  string `json:"side,omitempty"`
	Back  string `json:"back,omitempty"`
}

// CharacterSheets 角色三视图产物（produce 前锁定外观）。
type CharacterSheets struct {
	Characters []CharacterSheetEntry `json:"characters"`
}

// Save 写入 character-sheets.json。
func (c *CharacterSheets) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadCharacterSheets 读取角色三视图。
func LoadCharacterSheets(path string) (*CharacterSheets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c CharacterSheets
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// AppearanceBlock 供分镜/出图 prompt 注入。
func (c *CharacterSheets) AppearanceBlock() string {
	if c == nil || len(c.Characters) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[CHARACTERS] ")
	for i, ch := range c.Characters {
		if i > 0 {
			b.WriteString("；")
		}
		b.WriteString(ch.Name)
		b.WriteString("：")
		b.WriteString(strings.TrimSpace(ch.Appearance))
		if ch.hasTurnaroundViews() {
			b.WriteString("（已有正面/侧面/背面分图设定，外观须与对应视角一致）")
		} else if ch.TurnaroundPath != "" {
			b.WriteString("（已有三视图参考，前后侧一致）")
		}
	}
	return b.String()
}

func (ch CharacterSheetEntry) hasTurnaroundViews() bool {
	return ch.TurnaroundViews != nil &&
		(ch.TurnaroundViews.Front != "" || ch.TurnaroundViews.Side != "" || ch.TurnaroundViews.Back != "")
}

// TurnaroundViewForPrompt 根据画面描述推断应锁定的视角（front/side/back）。
func TurnaroundViewForPrompt(visual string) string {
	v := strings.ToLower(visual)
	switch {
	case strings.Contains(v, "背面") || strings.Contains(v, "背影") || strings.Contains(v, "背对") || strings.Contains(v, " back "):
		return "back"
	case strings.Contains(v, "侧面") || strings.Contains(v, "侧身") || strings.Contains(v, " profile") || strings.Contains(v, "侧影"):
		return "side"
	default:
		return "front"
	}
}

// ViewLockBlock 按视角注入角色锁定词（供关键帧/图生视频）；仅锁定本镜出现的角色。
func (c *CharacterSheets) ViewLockBlock(visual string) string {
	if c == nil || len(c.Characters) == 0 {
		return ""
	}
	matched := c.charactersForVisual(visual)
	if len(matched) == 0 {
		if excerpt := visualCharacterLockExcerpt(visual); excerpt != "" {
			return "[VIEW_LOCK] " + excerpt
		}
		return ""
	}
	view := TurnaroundViewForPrompt(visual)
	label := map[string]string{"front": "正面", "side": "侧面", "back": "背面"}[view]
	var b strings.Builder
	b.WriteString("[VIEW_LOCK] ")
	for i, ch := range matched {
		if i > 0 {
			b.WriteString("；")
		}
		b.WriteString(ch.Name)
		b.WriteString("须与")
		b.WriteString(label)
		b.WriteString("设定图一致：")
		b.WriteString(strings.TrimSpace(ch.Appearance))
		if ch.TurnaroundViews != nil {
			switch view {
			case "side":
				if ch.TurnaroundViews.Side != "" {
					b.WriteString("（侧面分图已生成）")
				}
			case "back":
				if ch.TurnaroundViews.Back != "" {
					b.WriteString("（背面分图已生成）")
				}
			default:
				if ch.TurnaroundViews.Front != "" {
					b.WriteString("（正面分图已生成）")
				}
			}
		}
	}
	return b.String()
}

func (c *CharacterSheets) charactersForVisual(visual string) []CharacterSheetEntry {
	if c == nil {
		return nil
	}
	visual = strings.TrimSpace(visual)
	var matched []CharacterSheetEntry
	for _, ch := range c.Characters {
		name := strings.TrimSpace(ch.Name)
		if name != "" && name != "主角" && strings.Contains(visual, name) {
			matched = append(matched, ch)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	if len(c.Characters) == 1 && c.Characters[0].Name == "主角" {
		if protagonistMatchesVisual(c.Characters[0].Appearance, visual) {
			return []CharacterSheetEntry{c.Characters[0]}
		}
		return nil
	}
	for _, ch := range c.Characters {
		if strings.TrimSpace(ch.Name) == "主角" {
			if protagonistMatchesVisual(ch.Appearance, visual) {
				matched = append(matched, ch)
			}
			continue
		}
		matched = append(matched, ch)
	}
	return matched
}

var altRoleKeywords = []string{"臣子", "侍卫", "大臣", "士兵", "侍女", "路人", "仆人", "信使", "刺客"}

func protagonistMatchesVisual(protagonistAppearance, visual string) bool {
	for _, role := range altRoleKeywords {
		if strings.Contains(visual, role) {
			return false
		}
	}
	protagonistRoles := []string{"国王", "女王", "骑士", "少年", "少女", "主角", "女孩", "男孩", "男子", "女子"}
	for _, role := range protagonistRoles {
		if strings.Contains(visual, role) {
			return strings.Contains(protagonistAppearance, role) || strings.Contains(protagonistAppearance, "主角")
		}
	}
	return true
}

func visualCharacterLockExcerpt(visual string) string {
	visual = strings.TrimSpace(visual)
	if visual == "" {
		return ""
	}
	r := []rune(visual)
	if len(r) > 120 {
		visual = string(r[:120]) + "…"
	}
	return "本镜角色外观须与画面描述一致：" + visual
}
