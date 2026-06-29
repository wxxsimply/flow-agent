package artifacts

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reOneWeapon  = regexp.MustCompile(`(?i)(一把|一支|一柄|单手|一手|右手.*一|左手.*一)`)
	reTwoWeapons = regexp.MustCompile(`(?i)(两把|两支|双剑|双刀|双手各持|两手各持|双手持|双手握|双持)`)
	reWeapon     = regexp.MustCompile(`(?i)(剑|刀|枪|矛|匕首|长剑|短剑|武士刀|盾|杯|伞|信封|照片|手机|通讯器|护目镜|光球|剑鞘)`)
	reTwoPeople  = regexp.MustCompile(`(?i)(两人|二人|双胞胎|两个.*人|两个相同|duplicate|克隆)`)
	reLeftHand   = regexp.MustCompile(`(?i)(左手|左臂|左掌|左腕)`)
	reRightHand  = regexp.MustCompile(`(?i)(右手|右臂|右掌|右腕)`)
	reHandSwap   = regexp.MustCompile(`(?i)(换手|换到.*手|从.*手.*到.*手|交到.*手|递到.*手)`)
	reHandRelease = regexp.MustCompile(`(?i)(放下|松开|接过|接过|接来|换手|换握|转到.*手)`)
	reGripChange = regexp.MustCompile(`(?i)(由.*转.*握|转为.*握|换握|换手|从.*手.*到.*手)`)
)

// PropHands 左右手道具分配。
type PropHands struct {
	Left  string
	Right string
}

// ParsePropHands 从 held_props / visual 解析左右手持物。
func ParsePropHands(text string) PropHands {
	text = strings.TrimSpace(text)
	if text == "" {
		return PropHands{}
	}
	if hands := parseStructuredPropHands(text); hands.Left != "" || hands.Right != "" {
		return hands
	}
	return inferPropHandsFromText(text)
}

func parseStructuredPropHands(text string) PropHands {
	var hands PropHands
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '；' || r == ';' || r == '，' || r == ','
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		val := part
		for _, sep := range []string{"：", ":"} {
			if i := strings.Index(part, sep); i >= 0 {
				val = strings.TrimSpace(part[i+len(sep):])
				break
			}
		}
		val = sanitizePropValue(val)
		if val == "" {
			continue
		}
		switch {
		case strings.HasPrefix(lower, "左手") || strings.HasPrefix(lower, "左：") || strings.HasPrefix(lower, "左:"):
			if hands.Left == "" {
				hands.Left = val
			}
		case strings.HasPrefix(lower, "右手") || strings.HasPrefix(lower, "右：") || strings.HasPrefix(lower, "右:"):
			if hands.Right == "" {
				hands.Right = val
			}
		}
	}
	return hands
}

func inferPropHandsFromText(text string) PropHands {
	var hands PropHands
	segments := regexp.MustCompile(`[，,；;]`).Split(text, -1)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		prop := extractPropFromSegment(seg)
		if prop == "" {
			continue
		}
		if reLeftHand.MatchString(seg) && hands.Left == "" {
			hands.Left = prop
		} else if reRightHand.MatchString(seg) && hands.Right == "" {
			hands.Right = prop
		}
	}
	if hands.Left == "" && hands.Right == "" {
		if reOneWeapon.MatchString(text) || reWeapon.MatchString(text) {
			prop := extractPropFromSegment(text)
			if prop != "" {
				if reLeftHand.MatchString(text) {
					hands.Left = prop
				} else {
					hands.Right = prop
				}
			}
		}
	}
	return hands
}

func extractPropFromSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	for _, prefix := range []string{"左手", "右手", "左臂", "右臂"} {
		if strings.HasPrefix(seg, prefix) {
			seg = strings.TrimSpace(seg[len(prefix):])
			break
		}
	}
	for _, prefix := range []string{"紧握", "紧握", "单手持", "单手", "手持", "持握", "握着", "握住", "持", "握", "仅"} {
		if strings.HasPrefix(seg, prefix) {
			seg = strings.TrimSpace(seg[len([]rune(prefix)):])
		}
	}
	for _, suffix := range []string{
		"（单手持握，画面不得出现第二把）", "（单手持握）", "单手持握", "单手持", "向下", "朝上",
	} {
		seg = strings.TrimSuffix(seg, suffix)
	}
	seg = strings.TrimSpace(seg)
	if seg == "" || seg == "空" || seg == "无" {
		return ""
	}
	return seg
}

// FormatHeldProps 统一为「右手：X；左手：Y」格式。
func FormatHeldProps(hands PropHands) string {
	var parts []string
	if r := strings.TrimSpace(hands.Right); r != "" {
		parts = append(parts, "右手："+r)
	} else {
		parts = append(parts, "右手：空")
	}
	if l := strings.TrimSpace(hands.Left); l != "" {
		parts = append(parts, "左手："+l)
	} else {
		parts = append(parts, "左手：空")
	}
	return strings.Join(parts, "；")
}

// PropHandConflict 检测同手双物或 structured 文本中重复同手描述。
func PropHandConflict(text string) bool {
	hands := ParsePropHands(text)
	if hands.Left != "" && hands.Right != "" && hands.Left == hands.Right {
		return true
	}
	rightCount := strings.Count(strings.ToLower(text), "右手")
	leftCount := strings.Count(strings.ToLower(text), "左手")
	if rightCount > 1 || leftCount > 1 {
		return true
	}
	return false
}

// HeldPropsHasHandSide 是否含明确左右手。
func HeldPropsHasHandSide(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "左手") || strings.Contains(t, "右手") ||
		strings.Contains(t, "左：") || strings.Contains(t, "右：")
}

// InferHeldProps 从旁白/画面推断道具锁定描述。
func InferHeldProps(narration, visual string) string {
	combined := strings.TrimSpace(narration) + " " + strings.TrimSpace(visual)
	hands := ParsePropHands(combined)
	if hands.Left != "" || hands.Right != "" {
		return FormatHeldProps(hands)
	}
	if !reWeapon.MatchString(combined) {
		return ""
	}
	if reTwoWeapons.MatchString(narration) || reTwoWeapons.MatchString(visual) {
		return "左手：武器；右手：武器"
	}
	if reOneWeapon.MatchString(narration) || reOneWeapon.MatchString(visual) {
		prop := "武器"
		if reLeftHand.MatchString(combined) {
			return FormatHeldProps(PropHands{Left: prop})
		}
		return FormatHeldProps(PropHands{Right: prop})
	}
	if reWeapon.MatchString(narration) {
		return FormatHeldProps(PropHands{Right: "武器"})
	}
	return ""
}

func sanitizePropValue(val string) string {
	val = strings.TrimSpace(val)
	val = strings.ReplaceAll(val, "\ufffd", "")
	val = strings.ReplaceAll(val, "��", "")
	if val == "" || val == "空" || val == "无" {
		return ""
	}
	for _, r := range val {
		if r == '\uFFFD' {
			return ""
		}
	}
	if strings.ContainsAny(val, "@#$%^&*<>{}[]|\\`~") {
		return ""
	}
	return val
}

// SanitizeHeldPropsText 清理 LLM 乱码与无效 held_props 片段。
func SanitizeHeldPropsText(s string) string {
	s = strings.ReplaceAll(s, "\ufffd", "")
	s = strings.ReplaceAll(s, "��", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 截断 LLM 附加在空手描述后的无关后缀（如重复 visual 片段）
	for _, marker := range []string{" 2D动画", " 少年", " 王殿内", " 明日方舟"} {
		if idx := strings.Index(s, marker); idx > 0 && strings.Contains(s[:idx], "空") {
			s = strings.TrimSpace(s[:idx])
		}
	}
	hands := ParsePropHands(s)
	if hands.Right != "" {
		if clean := sanitizePropValue(hands.Right); clean == "" {
			hands.Right = ""
		} else {
			hands.Right = clean
		}
	}
	if hands.Left != "" {
		if clean := sanitizePropValue(hands.Left); clean == "" {
			hands.Left = ""
		} else {
			hands.Left = clean
		}
	}
	if hands.Left == "" && hands.Right == "" {
		return ""
	}
	return FormatHeldProps(hands)
}

// NormalizeHeldProps 补全并规范化 shot.HeldProps。
func NormalizeHeldProps(shot *Shot) bool {
	if shot == nil {
		return false
	}
	changed := false
	if sanitized := SanitizeHeldPropsText(shot.HeldProps.String()); sanitized != strings.TrimSpace(shot.HeldProps.String()) {
		if sanitized == "" {
			shot.HeldProps = FlexString("")
		} else {
			shot.HeldProps = FlexString(sanitized)
		}
		changed = true
	}
	combined := strings.TrimSpace(shot.HeldProps.String()) + " " + strings.TrimSpace(shot.VisualPrompt)
	hands := ParsePropHands(combined)
	if hands.Left == "" && hands.Right == "" {
		if inferred := InferHeldProps(shot.Narration, shot.VisualPrompt); inferred != "" {
			shot.HeldProps = FlexString(inferred)
			return true
		}
		return false
	}
	formatted := FormatHeldProps(hands)
	if strings.TrimSpace(shot.HeldProps.String()) != formatted {
		shot.HeldProps = FlexString(formatted)
		changed = true
	}
	return changed
}

// PropsHandConsistencyNeg 手-道具稳定性 negative。
func PropsHandConsistencyNeg(heldProps string) string {
	parts := []string{
		"prop hand swap", "道具换手", "object jumps between hands",
		"prop vanish", "道具消失", "held object disappears",
		"prop morph", "武器变形", "object melting", "道具扭曲",
		"same hand two objects", "同手双物",
	}
	if p := strings.TrimSpace(heldProps); p != "" {
		hands := ParsePropHands(p)
		if hands.Right != "" {
			parts = append(parts, "prop moves to left hand", "道具移到左手")
		}
		if hands.Left != "" {
			parts = append(parts, "prop moves to right hand", "道具移到右手")
		}
	}
	return strings.Join(parts, ", ")
}

// PropsHandConsistencyPos 手-道具稳定性正向约束。
func PropsHandConsistencyPos(heldProps string) string {
	p := strings.TrimSpace(heldProps)
	if p == "" {
		return ""
	}
	return "，全程道具固定在同一手，形状刚性不变，指节包裹握持，道具全程可见不消失"
}

// ActionBeatHandSwapRisk 检测 action_beats 是否有未明示放下的换手。
func ActionBeatHandSwapRisk(beats []string) (bool, string) {
	if len(beats) == 0 {
		return false, ""
	}
	combined := strings.Join(beats, " ")
	if !reGripChange.MatchString(combined) && !reHandSwap.MatchString(combined) {
		return false, ""
	}
	if reHandRelease.MatchString(combined) {
		return false, ""
	}
	return true, "action_beats 含握姿/换手变化但未写放下或接过，i2v 易换手或变形"
}

// PropLockBlock 供 produce 按镜注入。
func PropLockBlock(heldProps string) string {
	p := strings.TrimSpace(heldProps)
	if p == "" {
		return ""
	}
	return fmt.Sprintf("[PROP_LOCK] %s（全程保持，禁止换到另一手/消失/变形）", p)
}

// InferCharacterCount 推断镜头内主角数量提示。
func InferCharacterCount(visual string) int {
	v := strings.TrimSpace(visual)
	if reTwoPeople.MatchString(v) {
		return 2
	}
	return 1
}

// PropsConsistencyNeg 生成道具/人数相关 negative 片段。
func PropsConsistencyNeg(heldProps string, characterCount int) string {
	var parts []string
	parts = append(parts, "duplicate character", "two identical persons", "cloned person", "extra weapons", "多余武器", "画面中出现两名相同主角")
	if characterCount <= 1 {
		parts = append(parts, "only one protagonist in frame", "仅一名主角")
	}
	if strings.TrimSpace(heldProps) != "" {
		parts = append(parts, "weapon count mismatch", "道具数量与描述不符")
		parts = append(parts, PropsHandConsistencyNeg(heldProps))
	}
	return strings.Join(parts, ", ")
}

// PropsNarrationVisualConflict 旁白与画面道具数量是否冲突。
func PropsNarrationVisualConflict(narration, visual string) (bool, string) {
	nOne := reOneWeapon.MatchString(narration) && reWeapon.MatchString(narration)
	nTwo := reTwoWeapons.MatchString(narration)
	vTwo := reTwoWeapons.MatchString(visual)
	vOne := reOneWeapon.MatchString(visual)
	if nOne && vTwo {
		return true, "旁白为单武器，画面描述含双手/双持"
	}
	if nTwo && vOne && !vTwo {
		return true, "旁白为双武器，画面描述为单武器"
	}
	return false, ""
}

func scenesSimilarForProp(a, b string) bool {
	a = normalizeSceneTextForProp(a)
	b = normalizeSceneTextForProp(b)
	if a == b {
		return true
	}
	if a == "" || b == "" {
		return false
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

func normalizeSceneTextForProp(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("，", " ", "。", " ", "、", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func shotSceneSame(prev, curr *Shot) bool {
	if prev == nil || curr == nil {
		return false
	}
	a := strings.TrimSpace(prev.SceneBackground)
	b := strings.TrimSpace(curr.SceneBackground)
	if a != "" && b != "" {
		return scenesSimilarForProp(a, b)
	}
	return scenesSimilarForProp(prev.VisualPrompt, curr.VisualPrompt)
}

// ApplyPropLocks 填充镜级道具/人数字段并做轻量修复。
func (s *Storyboard) ApplyPropLocks() int {
	if s == nil {
		return 0
	}
	fixed := 0
	var prev *Shot
	for i := range s.Shots {
		shot := &s.Shots[i]
		if shot.CharacterCount <= 0 {
			shot.CharacterCount = InferCharacterCount(shot.VisualPrompt)
			if shot.CharacterCount <= 0 {
				shot.CharacterCount = 1
			}
			fixed++
		}
		if strings.TrimSpace(shot.HeldProps.String()) == "" && prev != nil && shotSceneSame(prev, shot) {
			if p := strings.TrimSpace(prev.HeldProps.String()); p != "" {
				shot.HeldProps = FlexString(p)
				fixed++
			}
		}
		if strings.TrimSpace(shot.HeldProps.String()) == "" {
			shot.HeldProps = FlexString(InferHeldProps(shot.Narration, shot.VisualPrompt))
			if shot.HeldProps != "" {
				fixed++
			}
		}
		if NormalizeHeldProps(shot) {
			fixed++
		}
		if PropHandConflict(shot.HeldProps.String()) {
			hands := ParsePropHands(shot.VisualPrompt)
			if hands.Left != "" || hands.Right != "" {
				shot.HeldProps = FlexString(FormatHeldProps(hands))
				fixed++
			}
		}
		if conflict, msg := PropsNarrationVisualConflict(shot.Narration, shot.VisualPrompt); conflict {
			prefix := fmt.Sprintf("【道具锁定：%s】", shot.HeldProps.String())
			if !strings.Contains(shot.VisualPrompt, prefix) && !strings.Contains(shot.VisualPrompt, "【道具锁定：") {
				shot.VisualPrompt = prefix + shot.VisualPrompt
				fixed++
			}
			_ = msg
		}
		prev = shot
	}
	return fixed
}
