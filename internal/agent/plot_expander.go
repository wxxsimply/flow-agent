package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunPlotExpander 将模糊剧情扩写为 story-spine.json。
func RunPlotExpander(rc *runctx.Context) error {
	plot := strings.TrimSpace(rc.PlotInput)
	if plot == "" {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/plot-input.md")); err == nil {
			plot = strings.TrimSpace(string(data))
		}
	}
	if plot == "" {
		return fmt.Errorf("plot input empty: use --plot or artifacts/plot-input.md")
	}
	if err := rc.WriteArtifact("artifacts/plot-input.md", []byte(plot)); err != nil {
		return err
	}

	if rc.DryRun {
		spine := &artifacts.StorySpine{
			Title:             "[dry-run] 微电影",
			Logline:           plot,
			Tone:              "悬疑",
			Mood:              "suspense",
			EmotionArc:        "由平静转向惊悚",
			TargetDurationSec: rc.TargetDurationSec(),
			Acts: []artifacts.StoryAct{
				{Act: 1, Summary: "开端", Beats: []string{"引入主角", "异常发生"}},
				{Act: 2, Summary: "发展", Beats: []string{"追逐真相"}},
				{Act: 3, Summary: "结局", Beats: []string{"反转收尾"}},
			},
			Characters: []artifacts.Character{
				{Name: "主角", Appearance: "二十多岁程序员，戴眼镜，深色卫衣"},
			},
		}
		if err := writeStorySpine(rc, spine); err != nil {
			return err
		}
		plan, _ := BuildBGMPlan(rc, spine)
		return PersistBGMPlan(rc, plan)
	}

	if rc.Providers == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return fmt.Errorf("deepseek api_key required for expand (or --dry-run)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ref := rc.App.LLMRef("planner")
	req := llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.ExpandSystem,
		User:        prompts.ExpandUser(plot, rc.TargetDurationSec(), styleLabel(rc)),
		MaxTokens:   4096,
		Temperature: 0.7,
		JSONMode:    true,
	}
	client := rc.Providers.LLMForStage(rc.App, "planner")
	res, err := completeJSONWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("expand llm: %w", err)
	}
	rc.RecordLLM(res.Usage)

	jsonStr := ExtractTopLevelJSON(res.Text)
	var spine artifacts.StorySpine
	if err := json.Unmarshal([]byte(jsonStr), &spine); err != nil {
		return fmt.Errorf("parse story-spine: %w", err)
	}
	if spine.TargetDurationSec <= 0 {
		spine.TargetDurationSec = rc.TargetDurationSec()
	}
	LoadCreativeOptionsFromRun(rc)
	if rc.Creative != nil {
		spine.AnimationStyle = rc.Creative.AnimationStyle
	}
	if strings.TrimSpace(spine.Mood) == "" {
		spine.Mood = spine.Tone
	}
	if err := writeStorySpine(rc, &spine); err != nil {
		return err
	}
	plan, _ := BuildBGMPlan(rc, &spine)
	return PersistBGMPlan(rc, plan)
}

func styleLabel(rc *runctx.Context) string {
	LoadCreativeOptionsFromRun(rc)
	return config.StyleLabelZH(rc.Creative)
}

func writeStorySpine(rc *runctx.Context, spine *artifacts.StorySpine) error {
	data, err := json.MarshalIndent(spine, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/story-spine.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("story-spine.json", "artifacts/story-spine.json", true)
	return nil
}
