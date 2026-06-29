package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunScreenwriter 根据 story-spine 生成 script.json / script.md。
func RunScreenwriter(rc *runctx.Context) error {
	spinePath := rc.ArtifactPath("artifacts/story-spine.json")
	spineData, err := os.ReadFile(spinePath)
	if err != nil {
		return fmt.Errorf("read story-spine: %w", err)
	}

	if rc.DryRun {
		script := &artifacts.Script{
			Title:   "[dry-run] 微电影",
			Logline: "dry-run logline",
			Scenes: []artifacts.ScriptScene{
				{ID: 1, Heading: "深夜办公室", Action: "程序员盯着屏幕，咖啡冒热气", Narration: "深夜十一点，办公室只剩键盘声。"},
				{ID: 2, Heading: "异常", Action: "显示器里伸出一只苍白的手", Narration: "突然，屏幕里有什么东西在动。"},
				{ID: 3, Heading: "逃离", Action: "他撞翻椅子冲向门口", Narration: "他来不及思考，转身就跑。"},
			},
		}
		return writeScript(rc, script)
	}

	if rc.Providers == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return fmt.Errorf("deepseek api_key required for script (or --dry-run)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	ref := rc.App.LLMRef("writer")
	req := llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.ScreenwriterSystem,
		User:        prompts.ScreenwriterUser(string(spineData), rc.TargetDurationSec()),
		MaxTokens:   8192,
		Temperature: 0.65,
		JSONMode:    true,
	}
	client := rc.Providers.LLMForStage(rc.App, "writer")
	res, err := completeJSONWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("script llm: %w", err)
	}
	rc.RecordLLM(res.Usage)

	jsonStr := ExtractTopLevelJSON(res.Text)
	var script artifacts.Script
	if err := json.Unmarshal([]byte(jsonStr), &script); err != nil {
		return fmt.Errorf("parse script: %w", err)
	}
	return writeScript(rc, &script)
}

func writeScript(rc *runctx.Context, script *artifacts.Script) error {
	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/script.json", data); err != nil {
		return err
	}
	md := script.ToMarkdown()
	if err := rc.WriteArtifact("artifacts/script.md", []byte(md)); err != nil {
		return err
	}
	rc.RecordArtifact("script.json", artifacts.CanonicalWriteRel("artifacts/script.json"), true)
	rc.RecordArtifact("script.md", artifacts.CanonicalWriteRel("artifacts/script.md"), true)
	if rc.Workflow != "micro-movie" {
		if err := rc.WriteArtifact("artifacts/chapter.md", []byte(md)); err != nil {
			return err
		}
		rc.RecordArtifact("chapter.md", "artifacts/chapter.md", true)
	}
	return nil
}
