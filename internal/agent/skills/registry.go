package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/config"
)

// Registry 已加载的项目 skills。
type Registry struct {
	root   string
	Skills map[string]*Skill
}

var (
	defaultOnce sync.Once
	defaultReg  *Registry
	defaultErr  error
)

// Default 返回从项目根 `.cursor/skills` 加载的注册表（单例）。
func Default() (*Registry, error) {
	defaultOnce.Do(func() {
		root, err := config.FindRoot()
		if err != nil {
			defaultErr = err
			return
		}
		defaultReg, defaultErr = LoadFromRoot(root)
	})
	return defaultReg, defaultErr
}

// LoadFromRoot 加载 `<root>/.cursor/skills/*`。
func LoadFromRoot(root string) (*Registry, error) {
	skillsDir := filepath.Join(root, ".cursor", "skills")
	reg := &Registry{root: root, Skills: map[string]*Skill{}}
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return reg, nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk, loadErr := loadSkill(filepath.Join(skillsDir, e.Name()))
		if loadErr != nil {
			continue
		}
		reg.Skills[sk.Name] = sk
		if e.Name() != sk.Name {
			reg.Skills[e.Name()] = sk
		}
	}
	return reg, nil
}

// stageBindings 各阶段启用的 skill 与 reference 文件。
var stageBindings = map[Stage][]struct {
	skill string
	refs  []string
}{
	StageExpandBriefSegment: {
		{skill: "micro-movie-director", refs: []string{
			"camera-language.md", "narrative-shot-linkage.md", "costume-and-identity.md", "staging-and-detail.md", "character-performance.md",
		}},
	},
	StageExpandBriefContinue: {
		{skill: "micro-movie-director", refs: []string{
			"continue-camera-language.md", "camera-language.md",
			"continuity-ledger.md", "character-performance.md",
		}},
	},
	StageGenerateShots: {
		{skill: "micro-movie-director", refs: []string{
			"camera-language.md", "narrative-shot-linkage.md", "costume-and-identity.md", "staging-and-detail.md", "character-performance.md",
			"visual-polish.md", "physics-logic.md", "video-generation-forbidden.md",
			"prop-hand-lock.md", "ai-video-shot-template.md", "review-rubric.md",
		}},
		{skill: "physics-realism", refs: []string{
			"phyt2v-positive-rules.md", "physvid-negative.md",
		}},
	},
	StageReviewStoryboard: {
		{skill: "micro-movie-director", refs: []string{
			"review-rubric.md", "continuity-ledger.md", "physics-logic.md",
		}},
		{skill: "physics-realism", refs: []string{"videophy-material-cues.md"}},
	},
	StageProduceMotion: {
		{skill: "micro-movie-director", refs: []string{
			"produce-motion-checklist.md", "video-generation-forbidden.md", "prop-hand-lock.md",
		}},
		{skill: "physics-realism", refs: []string{
			"phyt2v-positive-rules.md", "physvid-negative.md",
		}},
	},
}

// stageMaxRefRunes 各阶段 reference 注入总字数上限（汉字约等于 rune）。
var stageMaxRefRunes = map[Stage]int{
	StageExpandBriefSegment:  6000,
	StageExpandBriefContinue: 4000,
	StageGenerateShots:       9000,
	StageReviewStoryboard:    6000,
}

// InjectSystem 将项目 skill 正文拼入 LLM system prompt（置于 base 之前）。
func InjectSystem(base string, stage Stage) string {
	reg, err := Default()
	if err != nil || reg == nil || len(reg.Skills) == 0 {
		return embeddedFallback(stage) + "\n\n" + base
	}
	var b strings.Builder
	b.WriteString("## 项目 Agent Skills（必须遵守）\n")
	used := reg.blockForStage(stage, &b)
	if used == 0 {
		b.WriteString(embeddedFallback(stage))
	}
	b.WriteString("\n\n")
	b.WriteString(base)
	return b.String()
}

func (r *Registry) blockForStage(stage Stage, b *strings.Builder) int {
	used := 0
	maxRunes := stageMaxRefRunes[stage]
	if maxRunes <= 0 {
		maxRunes = 8000
	}
	budget := maxRunes
	includeBody := stage != StageReviewStoryboard // 审查阶段省略 SKILL 正文减 token

	for _, bind := range stageBindings[stage] {
		sk := r.Skills[bind.skill]
		if sk == nil {
			continue
		}
		used++
		b.WriteString("\n### Skill: ")
		b.WriteString(sk.Name)
		if sk.Description != "" {
			b.WriteString(" — ")
			b.WriteString(sk.Description)
		}
		b.WriteString("\n")
		if includeBody && sk.Body != "" {
			body := truncateRunes(sk.Body, 800)
			b.WriteString(body)
			b.WriteString("\n")
		}
		for _, ref := range bind.refs {
			if budget <= 0 {
				b.WriteString("\n#### ")
				b.WriteString(ref)
				b.WriteString("\n…[truncated: stage ref budget]\n")
				continue
			}
			txt := sk.Reference(ref)
			if txt == "" {
				continue
			}
			refBudget := budget
			txt, consumed := truncateRunesWithConsumed(txt, refBudget)
			budget -= consumed
			b.WriteString("\n#### ")
			b.WriteString(ref)
			b.WriteString("\n")
			b.WriteString(txt)
			if consumed < utf8.RuneCountInString(sk.Reference(ref)) {
				b.WriteString("\n…[truncated]\n")
			}
			b.WriteString("\n")
		}
	}
	return used
}

// MotionPromptBlock produce 阶段追加到 i2v motion（非 LLM）。
func MotionPromptBlock() string {
	return MotionPromptBlockForProvider("")
}

// MotionPromptBlockForProvider 按视频 provider 选择 motion 约束长度（openai 用短句）。
func MotionPromptBlockForProvider(provider string) string {
	if strings.EqualFold(strings.TrimSpace(provider), "openai") {
		return motionPromptBlockSora()
	}
	reg, err := Default()
	if err != nil || reg == nil || len(reg.Skills) == 0 {
		return strings.TrimSpace(prompts.AnimationCraftPhysicsPrompt)
	}
	var pos, neg []string
	if sk := reg.Skills["micro-movie-director"]; sk != nil {
		pos = extractBulletsAfterHeading(sk.Reference("produce-motion-checklist.md"), "正向", 6)
		if len(pos) == 0 {
			pos = extractBullets(sk.Reference("produce-motion-checklist.md"), 6)
		}
		neg = extractBulletsAfterHeading(sk.Reference("produce-motion-checklist.md"), "负向", 4)
		propNeg := extractBulletsAfterHeading(sk.Reference("prop-hand-lock.md"), "负向", 2)
		neg = append(neg, propNeg...)
	}
	if sk := reg.Skills["physics-realism"]; sk != nil {
		if len(neg) < 4 {
			neg = append(neg, extractBulletsAfterHeading(sk.Reference("physvid-negative.md"), "通用负向", 4-len(neg))...)
		}
		if len(pos) < 4 {
			pos = append(pos, extractBulletsAfterHeading(sk.Reference("phyt2v-positive-rules.md"), "七类核心规则", 2)...)
		}
	}
	bullets := append(pos, prefixForbidden(neg)...)
	if len(bullets) == 0 {
		for _, bind := range stageBindings[StageProduceMotion] {
			sk := reg.Skills[bind.skill]
			if sk == nil {
				continue
			}
			for _, ref := range bind.refs {
				if s := summarizeRef(sk.Reference(ref)); s != "" {
					bullets = append(bullets, s)
				}
			}
		}
	}
	if len(bullets) == 0 {
		return strings.TrimSpace(prompts.AnimationCraftPhysicsPrompt)
	}
	if len(bullets) > 12 {
		bullets = bullets[:12]
	}
	return "，" + strings.Join(bullets, "，")
}

func motionPromptBlockSora() string {
	reg, err := Default()
	if err != nil || reg == nil {
		return "，慢推或固定镜头，主体微动，禁止多动作串联"
	}
	sk := reg.Skills["micro-movie-director"]
	if sk == nil {
		return "，慢推或固定镜头，主体微动，禁止多动作串联"
	}
	pos := extractBulletsAfterHeading(sk.Reference("sora-motion-brief.md"), "正向", 3)
	neg := extractBulletsAfterHeading(sk.Reference("sora-motion-brief.md"), "负向", 3)
	for i := range pos {
		if len([]rune(pos[i])) > 30 {
			pos[i] = string([]rune(pos[i])[:30])
		}
	}
	for i := range neg {
		if len([]rune(neg[i])) > 30 {
			neg[i] = string([]rune(neg[i])[:30])
		}
	}
	bullets := append(pos, prefixForbidden(neg)...)
	if len(bullets) == 0 {
		return "，慢推或固定镜头，主体微动，禁止多动作串联"
	}
	return "，" + strings.Join(bullets, "，")
}

func extractBulletsAfterHeading(md, heading string, max int) []string {
	if max <= 0 {
		return nil
	}
	start := 0
	if heading != "" {
		marker := "## " + heading
		idx := strings.Index(md, marker)
		if idx < 0 {
			return nil
		}
		start = idx + len(marker)
		next := strings.Index(md[start:], "\n## ")
		if next >= 0 {
			md = md[start : start+next]
		} else {
			md = md[start:]
		}
	}
	return extractBullets(md, max)
}

func prefixForbidden(bullets []string) []string {
	out := make([]string, 0, len(bullets))
	for _, b := range bullets {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if strings.HasPrefix(b, "禁止") {
			out = append(out, b)
			continue
		}
		out = append(out, "禁止"+strings.TrimPrefix(b, "禁止"))
	}
	return out
}

func extractBullets(md string, max int) []string {
	var out []string
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len([]rune(line)) > 40 {
			line = string([]rune(line)[:40])
		}
		out = append(out, line)
		if len(out) >= max {
			break
		}
	}
	return out
}

func summarizeRef(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		if len([]rune(line)) > 120 {
			line = string([]rune(line)[:120])
		}
		return line
	}
	return ""
}

func truncateRunes(s string, max int) string {
	out, _ := truncateRunesWithConsumed(s, max)
	return out
}

func truncateRunesWithConsumed(s string, max int) (string, int) {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return string(r), len(r)
	}
	return string(r[:max]), max
}

// AppliedReport 记录本次运行使用了哪些 skill。
type AppliedReport struct {
	Root    string              `json:"project_root"`
	Skills  []string            `json:"skills"`
	Stages  map[string][]string `json:"stages"`
	Refs    map[string][]string `json:"refs_by_stage,omitempty"`
}

// Report 返回当前注册表摘要。
func Report() AppliedReport {
	rep := AppliedReport{Stages: map[string][]string{}, Refs: map[string][]string{}}
	root, err := config.FindRoot()
	if err == nil {
		rep.Root = root
	}
	reg, err := Default()
	if err != nil || reg == nil {
		rep.Stages["fallback"] = []string{"embedded prompts.animation_craft"}
		return rep
	}
	for name := range reg.Skills {
		rep.Skills = append(rep.Skills, name)
	}
	for st, binds := range stageBindings {
		var names []string
		var refs []string
		for _, b := range binds {
			if reg.Skills[b.skill] != nil {
				names = append(names, b.skill)
				refs = append(refs, b.refs...)
			}
		}
		if len(names) > 0 {
			rep.Stages[string(st)] = names
			rep.Refs[string(st)] = refs
		}
	}
	return rep
}

func embeddedFallback(stage Stage) string {
	switch stage {
	case StageExpandBriefSegment, StageExpandBriefContinue, StageGenerateShots:
		return prompts.AnimationCraftPerformance + prompts.AnimationCraftPhysicsPrompt + prompts.AnimationCraftVisualPolish
	case StageReviewStoryboard:
		return prompts.StoryboardReviewSystem
	case StageProduceMotion:
		return prompts.AnimationCraftPhysicsPrompt
	default:
		return ""
	}
}
