package artifacts

import (
	"encoding/json"
	"os"
	"strings"
)

const (
	PropCategoryHeld      = "held"
	PropCategoryHeroScene = "hero_scene"
)

// PropSheetEntry 单物体三视图设定。
type PropSheetEntry struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Appearance      string              `json:"appearance"`
	Category        string              `json:"category"` // held | hero_scene
	TurnaroundPath  string              `json:"turnaround_path,omitempty"`
	TurnaroundViews *CharacterViewPaths `json:"turnaround_views,omitempty"`
}

// PropSheets 物体三视图产物（produce 前锁定外观）。
type PropSheets struct {
	Props []PropSheetEntry `json:"props"`
}

// Save 写入 prop-sheets.json。
func (p *PropSheets) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadPropSheets 读取物体三视图。
func LoadPropSheets(path string) (*PropSheets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p PropSheets
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// PropByID 按 ID 查找物体。
func (p *PropSheets) PropByID(id string) *PropSheetEntry {
	if p == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	for i := range p.Props {
		if p.Props[i].ID == id {
			return &p.Props[i]
		}
	}
	return nil
}

// PropByName 按名称查找物体（canonical name 匹配）。
func (p *PropSheets) PropByName(name string) *PropSheetEntry {
	if p == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	for i := range p.Props {
		if p.Props[i].Name == name {
			return &p.Props[i]
		}
	}
	return nil
}

// PropsForShot 返回镜头绑定的物体条目（prop_refs 优先，否则按 held_props 名称匹配）。
func (p *PropSheets) PropsForShot(shot Shot) []PropSheetEntry {
	if p == nil || len(p.Props) == 0 {
		return nil
	}
	if len(shot.PropRefs) > 0 {
		var out []PropSheetEntry
		for _, ref := range shot.PropRefs {
			if entry := p.PropByID(ref); entry != nil {
				out = append(out, *entry)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	hands := ParsePropHands(shot.HeldProps.String())
	seen := map[string]bool{}
	var out []PropSheetEntry
	for _, name := range []string{hands.Right, hands.Left} {
		name = strings.TrimSpace(name)
		if name == "" || name == "空" {
			continue
		}
		if entry := p.PropByName(name); entry != nil && !seen[entry.ID] {
			seen[entry.ID] = true
			out = append(out, *entry)
		}
	}
	return out
}

func (pe PropSheetEntry) hasTurnaroundViews() bool {
	return pe.TurnaroundViews != nil &&
		(pe.TurnaroundViews.Front != "" || pe.TurnaroundViews.Side != "" || pe.TurnaroundViews.Back != "")
}

// AppearanceBlock 供分镜/出图 prompt 注入。
func (p *PropSheets) AppearanceBlock(props []PropSheetEntry) string {
	if p == nil || len(props) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[PROPS] ")
	for i, pe := range props {
		if i > 0 {
			b.WriteString("；")
		}
		b.WriteString(pe.Name)
		b.WriteString("：")
		b.WriteString(strings.TrimSpace(pe.Appearance))
		if pe.hasTurnaroundViews() {
			b.WriteString("（已有正面/侧面/背面分图设定，外观须与对应视角一致）")
		} else if pe.TurnaroundPath != "" {
			b.WriteString("（已有三视图参考，前后侧一致）")
		}
	}
	return b.String()
}

// PropViewLockBlock 按视角注入物体锁定词（供关键帧/图生视频）。
func (p *PropSheets) PropViewLockBlock(visual string, propRefs []string) string {
	if p == nil || len(p.Props) == 0 {
		return ""
	}
	var matched []PropSheetEntry
	if len(propRefs) > 0 {
		for _, ref := range propRefs {
			if entry := p.PropByID(ref); entry != nil {
				matched = append(matched, *entry)
			}
		}
	}
	if len(matched) == 0 {
		return ""
	}
	view := TurnaroundViewForPrompt(visual)
	label := map[string]string{"front": "正面", "side": "侧面", "back": "背面"}[view]
	var b strings.Builder
	b.WriteString("[PROP_VIEW_LOCK] ")
	for i, pe := range matched {
		if i > 0 {
			b.WriteString("；")
		}
		b.WriteString(pe.Name)
		b.WriteString("须与")
		b.WriteString(label)
		b.WriteString("设定图一致：")
		b.WriteString(strings.TrimSpace(pe.Appearance))
		if pe.TurnaroundViews != nil {
			switch view {
			case "side":
				if pe.TurnaroundViews.Side != "" {
					b.WriteString("（侧面分图已生成）")
				}
			case "back":
				if pe.TurnaroundViews.Back != "" {
					b.WriteString("（背面分图已生成）")
				}
			default:
				if pe.TurnaroundViews.Front != "" {
					b.WriteString("（正面分图已生成）")
				}
			}
		}
	}
	return b.String()
}
