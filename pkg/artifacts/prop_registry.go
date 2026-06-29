package artifacts

import (
	"fmt"
	"sort"
	"strings"
)

// heroScenePropKeywords 跨镜重复出现的重要场景物名词。
var heroScenePropKeywords = []string{
	"王座", " throne", "信封", "照片", "光球", "烛台", "宝箱", "王冠", "权杖",
	"古剑", "长剑", "短剑", "匕首", "圆盾", "盾牌", "酒杯", "茶杯", "手机",
	"通讯器", "护目镜", "剑鞘", "卷轴", "地图", "钥匙", "项链", "戒指", "面具",
}

type propCandidate struct {
	name       string
	category   string
	appearance string
}

// CollectPropsFromStoryboard 从分镜提取手持物与 hero 场景物，生成 PropSheets（无三视图 PNG）。
func CollectPropsFromStoryboard(sb *Storyboard) *PropSheets {
	if sb == nil || len(sb.Shots) == 0 {
		return nil
	}
	heldNames := map[string]propCandidate{}
	heroCounts := map[string]map[string]int{} // scene -> propName -> count
	firstVisual := map[string]string{}         // propName -> first visual excerpt

	for _, shot := range sb.Shots {
		hands := ParsePropHands(shot.HeldProps.String())
		for _, name := range []string{hands.Right, hands.Left} {
			name = strings.TrimSpace(name)
			if name == "" || name == "空" || name == "武器" {
				continue
			}
			if _, ok := heldNames[name]; !ok {
				heldNames[name] = propCandidate{
					name:       name,
					category:   PropCategoryHeld,
					appearance: extractPropAppearance(name, shot.VisualPrompt),
				}
				firstVisual[name] = shot.VisualPrompt
			}
		}
		scene := sceneKeyForProp(shot)
		for _, kw := range heroScenePropKeywords {
			if strings.Contains(shot.VisualPrompt, kw) {
				propName := strings.TrimSpace(kw)
				if heroCounts[scene] == nil {
					heroCounts[scene] = map[string]int{}
				}
				heroCounts[scene][propName]++
				if _, ok := firstVisual[propName]; !ok {
					firstVisual[propName] = shot.VisualPrompt
				}
			}
		}
	}

	heroNames := map[string]bool{}
	for _, counts := range heroCounts {
		for name, cnt := range counts {
			if cnt >= 2 {
				heroNames[name] = true
			}
		}
	}

	// merge: held takes precedence for category
	ordered := []string{}
	seen := map[string]bool{}
	for name := range heldNames {
		if !seen[name] {
			seen[name] = true
			ordered = append(ordered, name)
		}
	}
	for name := range heroNames {
		if !seen[name] {
			seen[name] = true
			ordered = append(ordered, name)
		}
	}
	sort.Strings(ordered)

	if len(ordered) == 0 {
		return nil
	}

	sheets := &PropSheets{}
	for i, name := range ordered {
		cat := PropCategoryHeroScene
		app := extractPropAppearance(name, firstVisual[name])
		if c, ok := heldNames[name]; ok {
			cat = c.category
			if c.appearance != "" {
				app = c.appearance
			}
		}
		if app == "" {
			app = defaultPropAppearance(name, cat)
		}
		id := fmt.Sprintf("p%02d-%s", i+1, propSafeFileStem(name))
		sheets.Props = append(sheets.Props, PropSheetEntry{
			ID:         id,
			Name:       name,
			Appearance: app,
			Category:   cat,
		})
	}
	return sheets
}

func sceneKeyForProp(shot Shot) string {
	if s := strings.TrimSpace(shot.SceneBackground); s != "" {
		return normalizeSceneTextForProp(s)
	}
	return normalizeSceneTextForProp(shot.VisualPrompt)
}

func extractPropAppearance(propName, visual string) string {
	visual = strings.TrimSpace(visual)
	if visual == "" || propName == "" {
		return ""
	}
	runes := []rune(visual)
	propRunes := []rune(propName)
	if len(propRunes) == 0 {
		return ""
	}
	idx := indexRunes(runes, propRunes)
	if idx < 0 {
		return ""
	}
	start := idx - 30
	if start < 0 {
		start = 0
	}
	end := idx + len(propRunes) + 30
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		start = end
	}
	excerpt := strings.TrimSpace(string(runes[start:end]))
	if len([]rune(excerpt)) > 120 {
		excerpt = string([]rune(excerpt)[:120]) + "…"
	}
	return excerpt
}

func indexRunes(haystack, needle []rune) int {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func defaultPropAppearance(name, category string) string {
	switch category {
	case PropCategoryHeld:
		return name + "，手持道具，外观须全程一致"
	default:
		return name + "，场景重要物件，外观须跨镜一致"
	}
}

func propSafeFileStem(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	if s == "" {
		return "prop"
	}
	return s
}

// ApplyPropRefs 为每镜填充 prop_refs（held 物 + 画面内 hero 物）。
func ApplyPropRefs(sb *Storyboard, sheets *PropSheets) int {
	if sb == nil || sheets == nil || len(sheets.Props) == 0 {
		return 0
	}
	fixed := 0
	nameToID := map[string]string{}
	for _, pe := range sheets.Props {
		nameToID[pe.Name] = pe.ID
	}
	for i := range sb.Shots {
		shot := &sb.Shots[i]
		var refs []string
		seen := map[string]bool{}
		addRef := func(id string) {
			if id != "" && !seen[id] {
				seen[id] = true
				refs = append(refs, id)
			}
		}
		hands := ParsePropHands(shot.HeldProps.String())
		for _, name := range []string{hands.Right, hands.Left} {
			name = strings.TrimSpace(name)
			if name == "" || name == "空" {
				continue
			}
			if id, ok := nameToID[name]; ok {
				addRef(id)
			}
		}
		for _, pe := range sheets.Props {
			if pe.Category != PropCategoryHeroScene {
				continue
			}
			if strings.Contains(shot.VisualPrompt, pe.Name) {
				addRef(pe.ID)
			}
		}
		if len(refs) > 0 {
			shot.PropRefs = refs
			fixed++
		}
	}
	return fixed
}

// HeldPropIDForHand 返回某手当前绑定的 prop ID。
func HeldPropIDForHand(heldProps string, hand string, sheets *PropSheets) string {
	if sheets == nil {
		return ""
	}
	hands := ParsePropHands(heldProps)
	var name string
	switch hand {
	case "left":
		name = hands.Left
	case "right":
		name = hands.Right
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "空" {
		return ""
	}
	if pe := sheets.PropByName(name); pe != nil {
		return pe.ID
	}
	return ""
}

// ReviewPropContinuity 检测跨镜物体切换问题。
func ReviewPropContinuity(sb *Storyboard, sheets *PropSheets) []StoryboardReviewItem {
	if sb == nil || len(sb.Shots) < 2 {
		return nil
	}
	var issues []StoryboardReviewItem
	var prev *Shot
	for i := range sb.Shots {
		shot := &sb.Shots[i]
		if prev != nil && shotSceneSame(prev, shot) {
			issues = append(issues, checkHandPropSwitch(prev, shot, sheets)...)
			issues = append(issues, checkHeroPropSwitch(prev, shot, sheets)...)
		}
		prev = shot
	}
	return issues
}

func checkHandPropSwitch(prev, curr *Shot, sheets *PropSheets) []StoryboardReviewItem {
	var issues []StoryboardReviewItem
	for _, hand := range []struct{ key, label string }{
		{"right", "右手"},
		{"left", "左手"},
	} {
		prevID := HeldPropIDForHand(prev.HeldProps.String(), hand.key, sheets)
		currID := HeldPropIDForHand(curr.HeldProps.String(), hand.key, sheets)
		if prevID == "" || currID == "" {
			continue
		}
		if prevID == currID {
			continue
		}
		combined := strings.Join(curr.ActionBeats, " ") + " " + curr.Narration + " " + curr.VisualPrompt
		if reHandRelease.MatchString(combined) {
			continue
		}
		prevName := propNameByID(prevID, sheets)
		currName := propNameByID(currID, sheets)
		issues = append(issues, StoryboardReviewItem{
			ShotID:   curr.ID,
			Severity: "error",
			Field:    "prop_refs",
			Message:  fmt.Sprintf("同场景 %s 物体从「%s」变为「%s」但未写放下/接过，建议拆镜或补充动作", hand.label, prevName, currName),
		})
	}
	return issues
}

func checkHeroPropSwitch(prev, curr *Shot, sheets *PropSheets) []StoryboardReviewItem {
	if sheets == nil {
		return nil
	}
	var issues []StoryboardReviewItem
	prevHero := heroPropIDsInShot(*prev, sheets)
	currHero := heroPropIDsInShot(*curr, sheets)
	prevSet := map[string]bool{}
	for _, id := range prevHero {
		prevSet[id] = true
	}
	for id := range prevSet {
		if !containsString(currHero, id) {
			name := propNameByID(id, sheets)
			combined := strings.Join(curr.ActionBeats, " ") + " " + curr.VisualPrompt
			if !reHandRelease.MatchString(combined) && name != "" {
				issues = append(issues, StoryboardReviewItem{
					ShotID:   curr.ID,
					Severity: "warn",
					Field:    "prop_refs",
					Message:  fmt.Sprintf("同场景 hero 物体「%s」在上一镜出现但本镜消失", name),
				})
			}
		}
	}
	return issues
}

func heroPropIDsInShot(shot Shot, sheets *PropSheets) []string {
	var ids []string
	for _, pe := range sheets.Props {
		if pe.Category != PropCategoryHeroScene {
			continue
		}
		if strings.Contains(shot.VisualPrompt, pe.Name) {
			ids = append(ids, pe.ID)
		}
	}
	return ids
}

func propNameByID(id string, sheets *PropSheets) string {
	if sheets == nil {
		return id
	}
	if pe := sheets.PropByID(id); pe != nil {
		return pe.Name
	}
	return id
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// AlignHeldPropsToRegistry 将 held_props 名称对齐到 registry canonical name。
func AlignHeldPropsToRegistry(sb *Storyboard, sheets *PropSheets) int {
	if sb == nil || sheets == nil {
		return 0
	}
	fixed := 0
	for i := range sb.Shots {
		shot := &sb.Shots[i]
		hands := ParsePropHands(shot.HeldProps.String())
		changed := false
		for _, hand := range []*string{&hands.Right, &hands.Left} {
			name := strings.TrimSpace(*hand)
			if name == "" || name == "空" {
				continue
			}
			if pe := sheets.PropByName(name); pe != nil {
				if *hand != pe.Name {
					*hand = pe.Name
					changed = true
				}
				continue
			}
			for _, pe := range sheets.Props {
				if strings.Contains(name, pe.Name) || strings.Contains(pe.Name, name) {
					*hand = pe.Name
					changed = true
					break
				}
			}
		}
		if changed {
			shot.HeldProps = FlexString(FormatHeldProps(hands))
			fixed++
		}
	}
	return fixed
}
