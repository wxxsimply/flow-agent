package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// LLMReviewPatch LLM 审查后建议的字段补丁。
type LLMReviewPatch struct {
	ShotID           string `json:"shot_id"`
	VisualPrompt     string `json:"visual_prompt,omitempty"`
	Narration        string `json:"narration,omitempty"`
	PhysicsCues      string `json:"physics_cues,omitempty"`
	ForbiddenPhysics string `json:"forbidden_physics,omitempty"`
	ActionBeats      []string `json:"action_beats,omitempty"`
}

type llmReviewOut struct {
	Patches []LLMReviewPatch `json:"patches"`
	Summary string           `json:"summary"`
}

// ApplyLLMStoryboardReview 使用 micro-movie-director skill 对分镜做 LLM 审查并应用补丁。
func ApplyLLMStoryboardReview(ctx context.Context, rc *runctx.Context, sb *artifacts.Storyboard) (string, error) {
	if rc == nil || sb == nil || rc.DryRun || rc.Providers == nil {
		return "", nil
	}
	if strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return "", nil
	}

	data, err := json.Marshal(sb)
	if err != nil {
		return "", err
	}
	user := fmt.Sprintf("审查以下 storyboard JSON，仅输出 patches 数组（无则空数组）：\n%s", string(data))
	req := llm.CompletionRequest{
		System: InjectSystem(prompts.StoryboardReviewSystem+`

只输出 JSON：
{"patches":[{"shot_id":"s01","physics_cues":"...","forbidden_physics":"...","action_beats":["","",""]}],"summary":"..."}
不要输出 markdown。`, StageReviewStoryboard),
		User:        user,
		MaxTokens:   4096,
		Temperature: 0.2,
		JSONMode:    true,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	client := rc.Providers.LLMForStage(rc.App, "storyboard")
	ref := rc.App.LLMRef("storyboard")
	model := ref.Model
	if model == "" {
		model = "deepseek-v4-flash"
	}
	req.Model = model
	res, err := client.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	rc.RecordLLM(res.Usage)

	var out llmReviewOut
	if err := json.Unmarshal([]byte(extractTopLevelJSON(res.Text)), &out); err != nil {
		return "", fmt.Errorf("parse skill review: %w", err)
	}
	applied := 0
	for _, p := range out.Patches {
		for i := range sb.Shots {
			if sb.Shots[i].ID != p.ShotID {
				continue
			}
			if v := strings.TrimSpace(p.VisualPrompt); v != "" {
				sb.Shots[i].VisualPrompt = v
			}
			if v := strings.TrimSpace(p.Narration); v != "" {
				sb.Shots[i].Narration = v
			}
			if v := strings.TrimSpace(p.PhysicsCues); v != "" {
				sb.Shots[i].PhysicsCues = v
			}
			if v := strings.TrimSpace(p.ForbiddenPhysics); v != "" {
				sb.Shots[i].ForbiddenPhysics = v
			}
			if len(p.ActionBeats) >= 3 {
				sb.Shots[i].ActionBeats = p.ActionBeats
			}
			applied++
			break
		}
	}
	return out.Summary, nil
}
