package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunStoryboard 根据正文生成 storyboard.json 与 narration.ssml。
func RunStoryboard(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.DryRun {
		return runStoryboardDryRun(rc)
	}
	return runStoryboardLive(rc, v)
}

func runStoryboardDryRun(rc *runctx.Context) error {
	target := float64(rc.TargetDurationSec())
	pol := artifacts.KenBurnsShortDramaPolicy()
	videoNative := false
	if rc.App != nil && rc.App.Stack != nil {
		pol = storyboardPolicyForRun(rc)
		videoNative = pol.VideoNative()
	}
	if rc.Workflow == "micro-movie" {
		stackName := rc.Stack
		if stackName == "" && rc.App != nil && rc.App.Stack != nil {
			stackName = rc.App.Stack.Name
		}
		pol = artifacts.MicroMoviePolicy(stackName)
		videoNative = true
	}
	shotCount := 14
	if videoNative {
		shotCount = pol.MinShots
		if shotCount < 12 {
			shotCount = 12
		}
		if pol.MaxShots > 0 && shotCount > pol.MaxShots {
			shotCount = pol.MaxShots
		}
	}
	shotDur := 10.0
	if target/float64(shotCount) > 8 && target/float64(shotCount) < 12 {
		shotDur = target / float64(shotCount)
	}

	narrations := []string{
		"雨夜，那通电话改变了一切。",
		"三年前的误会，如今真相浮现。",
		"他在天台等她。",
		"手里握着从未寄出的信。",
		"她转身离去。",
		"复仇，从下集开始。",
		"真相，正在浮出水面。",
		"下集，绝不手软。",
	}

	shots := make([]artifacts.Shot, 0, shotCount)
	for i := 0; i < shotCount; i++ {
		narr := narrations[i%len(narrations)]
		vt := "ken_burns"
		ai := false
		vp := fmt.Sprintf("国漫爽文竖屏 9:16，都市夜景场景 %d", i+1)
		if videoNative {
			vt = "ai_video"
			ai = true
			vp = fmt.Sprintf("竖屏9:16，跟拍镜头，男主快步穿过雨夜街道，霓虹反射，国漫爽文风，场景 %d", i+1)
		}
		shots = append(shots, artifacts.Shot{
			ID:            fmt.Sprintf("s%02d", i+1),
			DurationSec:   shotDur,
			VisualType:    vt,
			AIVideoBudget: ai,
			VisualPrompt:  vp,
			Narration:     narr,
			Subtitle:      narr,
		})
	}

	sb := &artifacts.Storyboard{
		EpisodeNo:         rc.EpisodeNo,
		TargetDurationSec: target,
		Shots:             shots,
	}
	sb.NormalizeDurations(rc.TargetDurationSec())
	if err := sb.Validate(rc.EpisodeNo, rc.TargetDurationSec(), pol); err != nil {
		return err
	}
	ssml := buildSSMLFromShots(sb.Shots)
	return writeStoryboardArtifacts(rc, sb, ssml)
}

func runStoryboardLive(rc *runctx.Context, v *vault.SeriesVault) error {
	if rc.Providers == nil {
		return fmt.Errorf("dashscope api_key required for storyboard (or use --dry-run)")
	}
	if rc.App == nil || strings.TrimSpace(rc.App.Providers.DashScope.APIKey) == "" {
		return fmt.Errorf("dashscope api_key required for storyboard (or use --dry-run)")
	}

	chapter, err := loadChapterForStoryboard(rc)
	if err != nil {
		return err
	}
	hookJSON := "{}"
	if rc.ArtifactExists("artifacts/hook-plan.json") {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/hook-plan.json")); err == nil {
			hookJSON = string(data)
		}
	}
	spineJSON := "{}"
	if rc.ArtifactExists("artifacts/story-spine.json") {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/story-spine.json")); err == nil {
			spineJSON = string(data)
		}
	}
	bible, err := v.LoadBible()
	if err != nil {
		bible = ""
	}

	ref := rc.App.LLMRef("storyboard")
	pol := artifacts.KenBurnsShortDramaPolicy()
	videoNative := false
	videoOn := false
	if rc.App != nil && rc.App.Stack != nil {
		pol = storyboardPolicyForRun(rc)
		vid := rc.App.Stack.VideoConfig()
		videoOn = vid.Enabled
		videoNative = vid.VideoNative()
	}
	if rc.Workflow == "micro-movie" {
		stackName := rc.Stack
		if stackName == "" && rc.App != nil && rc.App.Stack != nil {
			stackName = rc.App.Stack.Name
		}
		pol = artifacts.MicroMoviePolicy(stackName)
		videoNative = true
		videoOn = true
	}
	microMovie := rc.Workflow == "micro-movie" || strings.HasPrefix(rc.Stack, "micro-movie")
	sys := prompts.StoryboardSystemKenBurns
	usr := prompts.StoryboardUserKenBurns(rc.EpisodeNo, rc.TargetDurationSec(), bible, hookJSON, string(chapter))
	if microMovie && videoNative {
		preset := VisualPresetForRun(rc)
		clipDur := 10.0
		if rc.App != nil && rc.App.Stack != nil {
			if d := rc.App.Stack.VideoConfig().ClipDurationSec; d > 0 {
				clipDur = float64(d)
			}
		}
		if pol.MaxShots <= 8 {
			sys = prompts.StoryboardSystemMicroMovieQuick(pol.MinShots, pol.MaxShots, clipDur)
		} else {
			sys = prompts.StoryboardSystemMicroMovie
		}
		if sheets, err := artifacts.LoadCharacterSheets(rc.ArtifactPath("artifacts/character-sheets.json")); err == nil {
			if b := strings.TrimSpace(sheets.AppearanceBlock()); b != "" {
				spineJSON += "\n\n## 角色三视图锁定\n" + b + "\n要求：分镜 visual_prompt 必须遵循三视图锁定，正侧背一致，禁止“正面头接背面”等不可能情况。"
			}
		}
		if pol.MaxShots <= 8 {
			usr = prompts.StoryboardUserMicroMovieQuick(rc.EpisodeNo, rc.TargetDurationSec(), pol.MinShots, pol.MaxShots, clipDur, spineJSON, string(chapter), preset.StoryboardHint)
		} else {
			usr = prompts.StoryboardUserMicroMovie(rc.EpisodeNo, rc.TargetDurationSec(), spineJSON, string(chapter), preset.StoryboardHint)
		}
	} else if videoNative {
		sys = prompts.StoryboardSystemVideoNative
		usr = prompts.StoryboardUserVideoNative(rc.EpisodeNo, rc.TargetDurationSec(), bible, hookJSON, string(chapter))
	} else if videoOn {
		sys = prompts.StoryboardSystem
		usr = prompts.StoryboardUser(rc.EpisodeNo, rc.TargetDurationSec(), bible, hookJSON, string(chapter))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	req := llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      sys,
		User:        usr,
		MaxTokens:   8192,
		Temperature: 0.4,
	}
	client := rc.Providers.LLMForStage(rc.App, "storyboard")

	var sb *artifacts.Storyboard
	var ssml string
	var lastParse error
	for attempt := 0; attempt < maxLLMAttempts; attempt++ {
		if attempt > 0 {
			slog.Warn("storyboard parse retry", "attempt", attempt+1, "err", lastParse)
		}
		res, err := completeJSONWithRetry(ctx, client, req)
		if err != nil {
			return fmt.Errorf("storyboard llm: %w", err)
		}
		rc.RecordLLM(res.Usage)
		sb, ssml, lastParse = parseStoryboardJSON(res.Text, rc.EpisodeNo, rc.TargetDurationSec())
		if lastParse == nil {
			break
		}
		if attempt == maxLLMAttempts-1 {
			return lastParse
		}
	}
	if sb == nil {
		return fmt.Errorf("storyboard: no output")
	}
	scriptLines := loadScriptNarrationLines(rc)
	if n := sb.RepairShots(string(chapter), scriptLines); n > 0 {
		slog.Warn("storyboard repaired missing fields", "shots_fixed", n)
	}
	if n := sb.DedupeNarrations(); n > 0 {
		slog.Info("storyboard deduped narrations", "shots_fixed", n)
	}
	sb.NormalizeDurations(rc.TargetDurationSec())
	if err := sb.Validate(rc.EpisodeNo, rc.TargetDurationSec(), pol); err != nil {
		// 再修一次（Normalize 后偶发空镜）
		if n := sb.RepairShots(string(chapter), scriptLines); n > 0 {
			slog.Warn("storyboard repair before re-validate", "shots_fixed", n)
		}
		sb.NormalizeDurations(rc.TargetDurationSec())
		if err2 := sb.Validate(rc.EpisodeNo, rc.TargetDurationSec(), pol); err2 != nil {
			return fmt.Errorf("storyboard validate: %w", err)
		}
	}

	if score, missing := artifacts.NarrationAlignmentScore(string(chapter), sb); score < 0.8 {
		slog.Warn("storyboard narration alignment low",
			"score", score,
			"missing_count", len(missing),
			"episode", rc.EpisodeNo,
		)
	}

	if strings.TrimSpace(ssml) == "" {
		ssml = buildSSMLFromShots(sb.Shots)
	}
	if report, err := applyStoryboardPostProcess(rc, sb); err != nil {
		return err
	} else if report != nil {
		if err := artifacts.SaveStoryboardReview(rc.ArtifactPath("artifacts/storyboard-review.json"), *report); err != nil {
			return err
		}
		rc.RecordArtifact("storyboard-review.json", "artifacts/storyboard-review.json", false)
		if len(report.Issues) > 0 {
			slog.Info("storyboard review", "issues", len(report.Issues), "auto_fixed", report.Fixed)
		}
	}
	return writeStoryboardArtifacts(rc, sb, ssml)
}

type storyboardLLMOutput struct {
	Storyboard    artifacts.Storyboard `json:"storyboard"`
	NarrationSSML string               `json:"narration_ssml"`
}

func parseStoryboardJSON(raw string, episodeNo, targetSec int) (*artifacts.Storyboard, string, error) {
	jsonStr := ExtractTopLevelJSON(raw)
	var out storyboardLLMOutput
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		snip := jsonStr
		if len(snip) > 500 {
			snip = snip[:500] + "..."
		}
		return nil, "", fmt.Errorf("parse storyboard json: %w\nmodel output snippet: %s", err, snip)
	}
	sb := &out.Storyboard
	sb.EpisodeNo = episodeNo
	sb.TargetDurationSec = float64(targetSec)
	sb.NormalizeShotIDs()
	return sb, out.NarrationSSML, nil
}

func buildSSMLFromShots(shots []artifacts.Shot) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><speak>`)
	for i, s := range shots {
		if i > 0 {
			b.WriteString(`<break time="500ms"/>`)
		}
		b.WriteString("<p>")
		b.WriteString(escapeSSML(s.Narration))
		b.WriteString("</p>")
	}
	b.WriteString("</speak>")
	return b.String()
}

func escapeSSML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func storyboardPolicyForRun(rc *runctx.Context) artifacts.StoryboardPolicy {
	return StoryboardPolicyForRun(rc)
}

func loadScriptNarrationLines(rc *runctx.Context) []string {
	if !rc.ArtifactExists("artifacts/script.json") {
		return nil
	}
	script, err := artifacts.LoadScript(rc.ArtifactPath("artifacts/script.json"))
	if err != nil {
		return nil
	}
	var lines []string
	for _, sc := range script.Scenes {
		if n := strings.TrimSpace(sc.Narration); n != "" {
			lines = append(lines, n)
		}
	}
	return lines
}

func loadChapterForStoryboard(rc *runctx.Context) ([]byte, error) {
	return os.ReadFile(artifacts.ResolveScriptPath(rc.RunDir))
}

func writeStoryboardArtifacts(rc *runctx.Context, sb *artifacts.Storyboard, ssml string) error {
	data, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/storyboard.json", data); err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/narration.ssml", []byte(ssml)); err != nil {
		return err
	}
	rc.RecordArtifact("storyboard.json", "artifacts/storyboard.json", true)
	rc.RecordArtifact("narration.ssml", "artifacts/narration.ssml", true)
	return nil
}
