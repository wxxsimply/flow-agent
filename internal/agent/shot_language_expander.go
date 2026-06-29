package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/agent/skills"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

const (
	defaultBriefRunesMin     = 2800
	defaultBriefRunesMax     = 3200
	defaultBriefRunesFloor   = 2200
	defaultBriefSegmentCount = 3
	maxBriefContinue         = 2
)

type briefExpandConfig struct {
	RunesMin       int
	RunesMax       int
	RunesFloor     int
	SegmentCount   int
}

func briefExpandConfigFor(rc *runctx.Context) briefExpandConfig {
	cfg := briefExpandConfig{
		RunesMin:     defaultBriefRunesMin,
		RunesMax:     defaultBriefRunesMax,
		RunesFloor:   defaultBriefRunesFloor,
		SegmentCount: defaultBriefSegmentCount,
	}
	if rc != nil && rc.App != nil {
		ac := rc.App.Stack.AssembleConfig()
		if ac.BriefRunesMin > 0 {
			cfg.RunesMin = ac.BriefRunesMin
		}
		if ac.BriefRunesMax > 0 {
			cfg.RunesMax = ac.BriefRunesMax
		}
		if ac.BriefRunesFloor > 0 {
			cfg.RunesFloor = ac.BriefRunesFloor
		}
		if ac.BriefSegmentCount > 0 {
			cfg.SegmentCount = ac.BriefSegmentCount
		}
	}
	return cfg
}

func useQuickBrief(rc *runctx.Context) bool {
	if rc == nil || rc.App == nil {
		return false
	}
	// Studio 审阅工作流（stop_after=assemble）必须完整扩写正文
	if rc.StopAfterStage == "assemble" {
		return false
	}
	return rc.App.Stack.AssembleConfig().QuickAssemble
}

type briefMeta struct {
	ShotLanguageBrief string `json:"shot_language_brief"`
	StoryBackground   string `json:"story_background"`
	Mood              string `json:"mood,omitempty"`
	Tone              string `json:"tone,omitempty"`
}

type briefSegment struct {
	SegmentIndex    int    `json:"segment_index"`
	StoryBackground string `json:"story_background,omitempty"`
	Mood            string `json:"mood,omitempty"`
	Tone            string `json:"tone,omitempty"`
	SegmentText     string `json:"segment_text"`
}

type briefContinue struct {
	Continuation string `json:"continuation"`
}

type shotsOnly struct {
	Shots []artifacts.ExpandedShotInput `json:"shots"`
}

// RunShotLanguageExpander 第一镜文本 → 约 3000 字扩写 → 自动生成全部分镜。
func RunShotLanguageExpander(rc *runctx.Context) (*artifacts.ShotLanguageExpand, error) {
	LoadCreativeOptionsFromRun(rc)
	opening := collectOpeningShotText(rc)
	if opening == "" {
		return nil, fmt.Errorf("empty input: provide opening shot text (--plot or first shot in creative-options)")
	}
	if err := rc.WriteArtifact("artifacts/opening-shot-input.md", []byte(opening)); err != nil {
		return nil, err
	}
	if err := rc.WriteArtifact("artifacts/plot-input.md", []byte(opening)); err != nil {
		return nil, err
	}

	minShots, maxShots := shotBoundsForRun(rc)

	if rc.DryRun {
		exp := dryRunShotLanguageExpand(rc, opening, minShots, maxShots)
		return exp, persistShotLanguageExpand(rc, exp, opening)
	}

	if rc.Providers == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return nil, fmt.Errorf("deepseek api_key required for shot language expand (or use --dry-run)")
	}

	var meta *briefMeta
	var err error
	if useQuickBrief(rc) {
		meta = quickBriefFromOpening(opening)
		slog.Info("quick assemble: skip brief expand", "max_shots", maxShots)
	} else {
		meta, err = expandOpeningBrief(rc, opening)
		if err != nil {
			return nil, err
		}
	}
	shots, err := generateShotsFromBrief(rc, opening, meta.ShotLanguageBrief, minShots, maxShots)
	if err != nil {
		return nil, err
	}
	lockFirstShot(&shots, opening)

	exp := &artifacts.ShotLanguageExpand{
		OpeningShot:       opening,
		ShotLanguageBrief: meta.ShotLanguageBrief,
		StoryBackground:   meta.StoryBackground,
		Mood:              meta.Mood,
		Tone:              meta.Tone,
		Shots:             shots,
	}
	normalizeExpandedShots(rc, exp, minShots, maxShots)
	dedupeExpandedNarrations(exp)
	return exp, persistShotLanguageExpand(rc, exp, opening)
}

func dedupeExpandedNarrations(exp *artifacts.ShotLanguageExpand) {
	if exp == nil || len(exp.Shots) == 0 {
		return
	}
	var prev string
	for i := range exp.Shots {
		s := &exp.Shots[i]
		narr := strings.TrimSpace(s.Narration)
		if narr == "" {
			narr = strings.TrimSpace(s.Dialogue)
		}
		if prev != "" && artifacts.NarrationsTooSimilar(narr, prev) {
			shot := artifacts.Shot{
				VisualPrompt: s.VisualPrompt,
				ActionBeats:  s.ActionBeats,
			}
			narr = artifacts.UniqueNarrationForShot(shot, prev, i+1)
			s.Narration = narr
		}
		s.Narration = artifacts.DedupeSentencesInText(narr)
		prev = s.Narration
	}
}

func shotBoundsForRun(rc *runctx.Context) (minShots, maxShots int) {
	stackName := rc.Stack
	clipDur := 10
	if rc.App != nil && rc.App.Stack != nil {
		if stackName == "" {
			stackName = rc.App.Stack.Name
		}
		if d := rc.App.Stack.VideoConfig().ClipDurationSec; d > 0 {
			clipDur = d
		}
	}
	pol := artifacts.MicroMoviePolicy(stackName)
	minShots, maxShots = pol.MinShots, pol.MaxShots
	target := rc.TargetDurationSec()
	if target > 0 {
		want := target / clipDur
		if want < 1 {
			want = 1
		}
		if want < minShots {
			minShots = want
		}
		if want > maxShots {
			maxShots = want
		}
	}
	if rc.App != nil && rc.App.Stack != nil {
		if cap := rc.App.Stack.VideoConfig().MaxProduceShots; cap > 0 && maxShots > cap {
			maxShots = cap
		}
		if minShots > maxShots {
			minShots = maxShots
		}
	}
	if minShots <= 0 {
		minShots = 12
	}
	if maxShots <= 0 {
		maxShots = 18
	}
	return minShots, maxShots
}

func quickBriefFromOpening(opening string) *briefMeta {
	opening = strings.TrimSpace(opening)
	brief := opening
	if utf8.RuneCountInString(brief) > 1200 {
		brief = truncateRunes(brief, 1200)
	}
	bg := opening
	if utf8.RuneCountInString(bg) > 200 {
		bg = truncateRunes(bg, 200)
	}
	return &briefMeta{
		ShotLanguageBrief: brief,
		StoryBackground:   bg,
		Mood:              "neutral",
		Tone:              "紧凑",
	}
}

func collectOpeningShotText(rc *runctx.Context) string {
	if rc.Creative != nil {
		if t := strings.TrimSpace(rc.Creative.OpeningShot); t != "" {
			return t
		}
	}
	text := strings.TrimSpace(rc.PlotInput)
	if text == "" && rc.Creative != nil {
		text = strings.TrimSpace(rc.Creative.Plot)
	}
	if text == "" {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/opening-shot-input.md")); err == nil {
			text = strings.TrimSpace(string(data))
		}
	}
	if text == "" {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/plot-input.md")); err == nil {
			text = strings.TrimSpace(string(data))
		}
	}
	if text == "" {
		inputs, _ := loadUserShots(rc)
		if len(inputs) > 0 {
			in := inputs[0]
			var b strings.Builder
			if n := strings.TrimSpace(in.Narration); n != "" {
				b.WriteString(n)
			}
			if v := strings.TrimSpace(in.VisualDesc); v != "" {
				if b.Len() > 0 {
					b.WriteString("。")
				}
				b.WriteString("画面：")
				b.WriteString(v)
			}
			text = b.String()
		}
	}
	return strings.TrimSpace(text)
}

func expandOpeningBrief(rc *runctx.Context, opening string) (*briefMeta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	bcfg := briefExpandConfigFor(rc)
	meta := &briefMeta{}
	var parts []string
	var prevTail string

	for seg := 1; seg <= bcfg.SegmentCount; seg++ {
		bs, err := callBriefSegment(ctx, rc, opening, seg, bcfg, prevTail)
		if err != nil {
			return nil, fmt.Errorf("brief segment %d: %w", seg, err)
		}
		text := strings.TrimSpace(bs.SegmentText)
		if text == "" {
			return nil, fmt.Errorf("brief segment %d: empty segment_text", seg)
		}
		if seg == 1 {
			meta.StoryBackground = strings.TrimSpace(bs.StoryBackground)
			meta.Mood = strings.TrimSpace(bs.Mood)
			meta.Tone = strings.TrimSpace(bs.Tone)
		}
		parts = append(parts, text)
		prevTail = tailRunes(text, 800)
		n := segmentRuneCount(text)
		slog.Info("brief segment done", "segment", seg, "runes", n, "target_min", bcfg.RunesMin)
	}

	meta.ShotLanguageBrief = strings.Join(parts, "\n\n")
	briefLen := utf8.RuneCountInString(meta.ShotLanguageBrief)

	for c := 0; briefLen < bcfg.RunesMin && c < maxBriefContinue; c++ {
		cont, err := callBriefContinue(ctx, rc, opening, meta.ShotLanguageBrief, briefLen, bcfg.RunesMin)
		if err != nil {
			slog.Warn("brief continue failed", "attempt", c+1, "err", err)
			break
		}
		cont = strings.TrimSpace(cont)
		if cont == "" {
			break
		}
		meta.ShotLanguageBrief += "\n\n" + cont
		briefLen = utf8.RuneCountInString(meta.ShotLanguageBrief)
		slog.Info("brief continue done", "attempt", c+1, "total_runes", briefLen)
	}

	if briefLen < bcfg.RunesFloor {
		return nil, fmt.Errorf("opening brief too short: %d runes (need ~%d; model may be rate-limited)", briefLen, bcfg.RunesMin)
	}
	if briefLen > bcfg.RunesMax+2000 {
		meta.ShotLanguageBrief = truncateRunes(meta.ShotLanguageBrief, bcfg.RunesMax)
	}
	if briefLen < bcfg.RunesMin {
		slog.Warn("brief below target but accepted", "runes", briefLen, "target", bcfg.RunesMin)
	}
	return meta, nil
}

func callBriefSegment(ctx context.Context, rc *runctx.Context, opening string, segIndex int, bcfg briefExpandConfig, prevTail string) (*briefSegment, error) {
	req := llm.CompletionRequest{
		System:      skills.InjectSystem(prompts.OpeningShotBriefSegmentSystem(bcfg.SegmentCount, bcfg.RunesMin), skills.StageExpandBriefSegment),
		User:        prompts.OpeningShotBriefSegmentUser(opening, segIndex, bcfg.SegmentCount, bcfg.RunesMin, rc.TargetDurationSec(), styleLabel(rc), prevTail),
		MaxTokens:   8192,
		Temperature: 0.55,
		JSONMode:    true,
	}
	res, err := completeJSONWithProviders(ctx, rc, req)
	if err != nil {
		return nil, err
	}
	rc.RecordLLM(res.Usage)
	var bs briefSegment
	if err := json.Unmarshal([]byte(ExtractTopLevelJSON(res.Text)), &bs); err != nil {
		return nil, err
	}
	return &bs, nil
}

func callBriefContinue(ctx context.Context, rc *runctx.Context, opening, fullBrief string, currentRunes, targetRunes int) (string, error) {
	req := llm.CompletionRequest{
		System:      skills.InjectSystem(prompts.OpeningShotBriefContinueSystem, skills.StageExpandBriefContinue),
		User:        prompts.OpeningShotBriefContinueUser(opening, tailRunes(fullBrief, 1200), currentRunes, targetRunes),
		MaxTokens:   8192,
		Temperature: 0.5,
		JSONMode:    true,
	}
	res, err := completeJSONWithProviders(ctx, rc, req)
	if err != nil {
		return "", err
	}
	rc.RecordLLM(res.Usage)
	var bc briefContinue
	if err := json.Unmarshal([]byte(ExtractTopLevelJSON(res.Text)), &bc); err != nil {
		return "", err
	}
	return bc.Continuation, nil
}

// completeJSONWithProviders 先 planner 阶段 LLM，失败再试 storyboard 阶段 LLM。
func completeJSONWithProviders(ctx context.Context, rc *runctx.Context, req llm.CompletionRequest) (llm.CompletionResult, error) {
	req.JSONMode = true
	client, model := llmClientForStage(rc, "planner", "deepseek-v4-flash")
	req.Model = model
	res, err := completeJSONWithRetry(ctx, client, req)
	if err == nil {
		return res, nil
	}
	client, model = llmClientForStage(rc, "storyboard", "deepseek-v4-flash")
	req.Model = model
	return completeJSONWithRetry(ctx, client, req)
}

func segmentRuneCount(s string) int {
	return utf8.RuneCountInString(strings.TrimSpace(s))
}

func tailRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

func generateShotsFromBrief(rc *runctx.Context, opening, brief string, minShots, maxShots int) ([]artifacts.ExpandedShotInput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	bcfg := briefExpandConfigFor(rc)
	maxTok := 16384
	if useQuickBrief(rc) {
		maxTok = 4096
	}
	req := llm.CompletionRequest{
		System:      skills.InjectSystem(prompts.ShotsFromBriefSystem(bcfg.RunesMin), skills.StageGenerateShots),
		User:        prompts.ShotsFromBriefUser(opening, brief, rc.TargetDurationSec(), styleLabel(rc), minShots, maxShots, bcfg.RunesMin),
		MaxTokens:   maxTok,
		Temperature: 0.45,
		JSONMode:    true,
	}
	res, err := completeJSONWithProviders(ctx, rc, req)
	if err != nil {
		return nil, fmt.Errorf("generate shots from brief llm: %w", err)
	}
	rc.RecordLLM(res.Usage)

	var out shotsOnly
	if err := json.Unmarshal([]byte(ExtractTopLevelJSON(res.Text)), &out); err != nil {
		return nil, fmt.Errorf("parse shots from brief: %w", err)
	}
	if len(out.Shots) == 0 {
		return nil, fmt.Errorf("generate shots: empty shots array")
	}
	return out.Shots, nil
}

func llmClientForStage(rc *runctx.Context, stage, defaultModel string) (llm.Client, string) {
	ref := rc.App.LLMRef(stage)
	return rc.Providers.LLMForStage(rc.App, stage), modelOrDefault(ref, defaultModel)
}

func lockFirstShot(shots *[]artifacts.ExpandedShotInput, opening string) {
	if shots == nil || len(*shots) == 0 {
		return
	}
	s := &(*shots)[0]
	s.ID = "s01"
	opening = strings.TrimSpace(opening)
	if strings.TrimSpace(s.Narration) == "" {
		s.Narration = openingNarration(opening)
	}
	if strings.TrimSpace(s.VisualPrompt) == "" {
		s.VisualPrompt = opening
	}
	// 第一镜 visual 必须包含用户原文核心
	if !strings.Contains(s.VisualPrompt, truncateRunes(opening, 20)) && opening != "" {
		s.VisualPrompt = opening + "。" + s.VisualPrompt
	}
}

func openingNarration(opening string) string {
	r := []rune(strings.TrimSpace(opening))
	if len(r) <= 50 {
		return string(r)
	}
	return string(r[:50])
}

func normalizeExpandedShots(rc *runctx.Context, exp *artifacts.ShotLanguageExpand, minShots, maxShots int) {
	if exp == nil {
		return
	}
	quick := false
	clipDur := 10.0
	if rc != nil && rc.App != nil && rc.App.Stack != nil {
		quick = rc.App.Stack.AssembleConfig().QuickAssemble
		if d := rc.App.Stack.VideoConfig().ClipDurationSec; d > 0 {
			clipDur = float64(d)
		}
	}
	for i := range exp.Shots {
		s := &exp.Shots[i]
		if strings.TrimSpace(s.ID) == "" {
			s.ID = fmt.Sprintf("s%02d", i+1)
		}
		s.ShotSize = artifacts.NormalizeShotSize(s.ShotSize)
		if s.DurationSec <= 0 {
			s.DurationSec = clipDur
		}
		if strings.TrimSpace(s.VisualPrompt) == "" {
			s.VisualPrompt = joinVisualFromFields(*s)
		}
		if strings.TrimSpace(s.Narration) == "" && strings.TrimSpace(s.Dialogue) != "" {
			s.Narration = s.Dialogue
		}
		base := strings.TrimSpace(s.VisualPrompt)
		if quick {
			if strings.TrimSpace(s.SceneBackground) == "" && i > 0 && i%2 == 1 {
				s.SceneBackground = "新场景：" + truncateRunes(base, 40)
			}
			if len(s.ActionBeats) == 0 && base != "" {
				hints := []string{"独立构图", "新机位", "可切换场景", "不同背景"}
				s.ActionBeats = []string{base + "，" + hints[i%len(hints)]}
			}
		} else if len(s.ActionBeats) < 3 {
			s.ActionBeats = []string{
				base + "，动作起始姿态，肢体稳定",
				base + "，动作进行中，小幅位移",
				base + "，动作结束姿态，与起始一致",
			}
		}
	}
	if len(exp.Shots) > maxShots && maxShots > 0 {
		exp.Shots = exp.Shots[:maxShots]
	}
}

func joinVisualFromFields(s artifacts.ExpandedShotInput) string {
	var parts []string
	for _, p := range []string{s.SceneBackground, s.CameraAngle, s.CharacterMotion, s.ActionBehavior, s.MicroExpression} {
		if t := strings.TrimSpace(p); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "，")
}

func persistShotLanguageExpand(rc *runctx.Context, exp *artifacts.ShotLanguageExpand, opening string) error {
	if exp == nil {
		return fmt.Errorf("nil expand result")
	}
	if err := exp.Save(rc.ArtifactPath("artifacts/shot-language-expand.json")); err != nil {
		return err
	}
	md := exp.ShotLanguageBrief
	header := "# 第一镜输入\n\n" + opening + "\n\n"
	if strings.TrimSpace(exp.StoryBackground) != "" {
		header += "# 故事背景\n\n" + exp.StoryBackground + "\n\n---\n\n"
	}
	md = header + md
	if err := rc.WriteArtifact("artifacts/shot-language-brief.md", []byte(md)); err != nil {
		return err
	}
	rc.RecordArtifact("opening-shot-input.md", "artifacts/opening-shot-input.md", false)
	rc.RecordArtifact("shot-language-expand.json", "artifacts/shot-language-expand.json", true)
	rc.RecordArtifact("shot-language-brief.md", "artifacts/shot-language-brief.md", false)

	spine := &artifacts.StorySpine{
		Title:             "用户微电影",
		Logline:           exp.StoryBackground,
		Tone:              exp.Tone,
		Mood:              exp.Mood,
		EmotionArc:        exp.Tone,
		TargetDurationSec: rc.TargetDurationSec(),
	}
	if spine.Logline == "" {
		spine.Logline = truncateRunes(opening, 80)
	}
	if spine.Mood == "" {
		spine.Mood = "neutral"
	}
	_ = writeStorySpine(rc, spine)
	if rc.App != nil {
		plan, _ := BuildBGMPlan(rc, spine)
		_ = PersistBGMPlan(rc, plan)
	}
	return nil
}

func dryRunShotLanguageExpand(rc *runctx.Context, opening string, minShots, maxShots int) *artifacts.ShotLanguageExpand {
	n := minShots
	if !useQuickBrief(rc) && n < 12 {
		n = 12
	}
	if maxShots > 0 && n > maxShots {
		n = maxShots
	}
	clipDur := 10.0
	if rc != nil && rc.App != nil {
		if d := rc.App.Stack.VideoConfig().ClipDurationSec; d > 0 {
			clipDur = float64(d)
		}
	}
	shots := make([]artifacts.ExpandedShotInput, 0, n)
	for i := 0; i < n; i++ {
		narr := fmt.Sprintf("dry-run 第%d镜旁白。", i+1)
		vis := fmt.Sprintf("dry-run 画面 %d", i+1)
		if i == 0 {
			narr = openingNarration(opening)
			vis = opening
		}
		shots = append(shots, artifacts.ExpandedShotInput{
			ID: sID(i), ShotSize: "medium", Narration: narr, VisualPrompt: vis,
			CameraAngle: "中景", SceneBackground: "dry-run 场景",
			DurationSec: clipDur,
			ActionBeats: []string{vis + " 连贯动作"},
		})
	}
	brief := opening
	if !useQuickBrief(rc) {
		bcfg := briefExpandConfigFor(rc)
		brief = opening + strings.Repeat("\n\n（dry-run：人物剧情、服装细节、整体形象、镜头语言、场景光影。）", 80)
		if utf8.RuneCountInString(brief) < bcfg.RunesMin {
			brief += strings.Repeat("细节。", bcfg.RunesMin-utf8.RuneCountInString(brief))
		}
	}
	return &artifacts.ShotLanguageExpand{
		OpeningShot:       opening,
		ShotLanguageBrief: brief,
		StoryBackground:   truncateRunes(opening, 80),
		Mood:              "neutral",
		Tone:              "史诗",
		Shots:             shots,
	}
}

func sID(i int) string {
	return fmt.Sprintf("s%02d", i+1)
}

func expandedToStoryboard(rc *runctx.Context, exp *artifacts.ShotLanguageExpand) (*artifacts.Storyboard, error) {
	shots := make([]artifacts.Shot, 0, len(exp.Shots))
	for _, s := range exp.Shots {
		narr := strings.TrimSpace(s.Narration)
		if narr == "" {
			narr = strings.TrimSpace(s.Dialogue)
		}
		sub := narr
		if len([]rune(sub)) > 24 {
			sub = string([]rune(sub)[:24])
		}
		vp := strings.TrimSpace(s.VisualPrompt)
		if joined := joinVisualFromFields(s); joined != "" {
			if vp == "" || (strings.TrimSpace(s.CameraAngle) != "" && !strings.Contains(vp, strings.TrimSpace(s.CameraAngle))) {
				if vp != "" {
					vp = joined + "，" + vp
				} else {
					vp = joined
				}
			}
		}
		shots = append(shots, artifacts.Shot{
			ID:               strings.TrimSpace(s.ID),
			DurationSec:      s.DurationSec,
			VisualType:       "ai_video",
			AIVideoBudget:    true,
			VisualPrompt:     vp,
			Narration:        narr,
			Subtitle:         sub,
			ShotSize:         artifacts.NormalizeShotSize(s.ShotSize),
			CameraAngle:      strings.TrimSpace(s.CameraAngle),
			NarrativeBeat:    strings.TrimSpace(s.NarrativeBeat),
			BriefExcerpt:     strings.TrimSpace(s.BriefExcerpt),
			ActionBeats:      s.ActionBeats,
			PhysicsCues:      strings.TrimSpace(s.PhysicsCues),
			ForbiddenPhysics: strings.TrimSpace(s.ForbiddenPhysics),
			CharacterCount:   s.CharacterCount,
			HeldProps:        artifacts.FlexString(artifacts.SanitizeHeldPropsText(strings.TrimSpace(s.HeldProps.String()))),
			SceneBackground:  strings.TrimSpace(s.SceneBackground),
			Expanded:         true,
		})
	}
	sb := &artifacts.Storyboard{
		EpisodeNo:         rc.EpisodeNo,
		TargetDurationSec: float64(rc.TargetDurationSec()),
		Shots:             shots,
	}
	sb.SyncTotalNarrationSec()
	return sb, nil
}
