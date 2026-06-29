package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/agent/prompts"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunWriter 按场景流式写入 chapter.parts 并合并为 chapter.md。
func RunWriter(rc *runctx.Context) error {
	if rc.DryRun {
		return runWriterDryRun(rc)
	}
	return runWriterLive(rc)
}

func runWriterDryRun(rc *runctx.Context) error {
	partsDir := rc.ArtifactPath("artifacts/chapter.parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		return err
	}

	scenes := []string{
		"雨夜，她接到那通改变命运的电话。",
		"回忆涌来，三年前的那场误会真相渐明。",
		"他在天台等她，手里握着未寄出的信。",
		"她转身离去——下一集，复仇正式开始。",
	}

	var full string
	full = "<!-- dry-run -->\n"
	for i, text := range scenes {
		part := fmt.Sprintf("## Scene %d\n\n%s\n", i+1, text)
		name := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", i+1))
		if err := rc.WriteArtifact(name, []byte(part)); err != nil {
			return err
		}
		full += part + "\n"
	}

	if err := rc.WriteArtifact("artifacts/chapter.md", []byte(full)); err != nil {
		return err
	}
	rc.RecordArtifact("chapter.md", "artifacts/chapter.md", true)
	return nil
}

func runWriterLive(rc *runctx.Context) error {
	if rc.Providers == nil {
		return fmt.Errorf("deepseek api_key required (or use --dry-run)")
	}
	if rc.App == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return fmt.Errorf("deepseek api_key required (or use --dry-run)")
	}

	hookPath := rc.ArtifactPath("artifacts/hook-plan.json")
	plan, err := artifacts.LoadHookPlan(hookPath)
	if err != nil {
		return fmt.Errorf("load hook-plan: %w (run plan stage first)", err)
	}
	if err := plan.Validate(rc.EpisodeNo); err != nil {
		return err
	}

	if rc.App != nil {
		_ = vault.ForSeries(rc.App, rc.SeriesID).EnsureCharacterStateFromBible()
	}

	briefExcerpt, _ := os.ReadFile(rc.ArtifactPath("artifacts/episode-brief.md"))
	excerpt := truncate(string(briefExcerpt), 800)

	partsDir := rc.ArtifactPath("artifacts/chapter.parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		return err
	}

	ref := rc.App.LLMRef("writer")
	maxTokens := ref.ChunkMaxTokens
	if maxTokens <= 0 {
		maxTokens = 600
	}

	rewriteHints, _ := artifacts.LoadRewriteHints(rc.ArtifactPath(artifacts.RewriteHintsRel))
	charStateJSON := loadCharacterStateJSON(rc)

	skipped := 0
	for _, scene := range plan.Scenes {
		partRel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", scene.ID))
		if rc.ArtifactExists(partRel) {
			skipped++
			continue
		}

		maxChars := scene.MaxChars
		if maxChars <= 0 {
			maxChars = 300
		}

		fixHint := ""
		if rewriteHints != nil {
			fixHint = rewriteHints.HintForScene(scene.ID)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		text, err := streamScene(ctx, rc, ref, scene, rc.EpisodeNo, maxChars, maxTokens, excerpt, fixHint, charStateJSON)
		cancel()
		if err != nil {
			return fmt.Errorf("scene %d: %w", scene.ID, err)
		}

		text = strings.TrimSpace(truncateRunes(text, maxChars))
		part := fmt.Sprintf("## Scene %d\n\n%s\n", scene.ID, text)
		if err := rc.WriteArtifact(partRel, []byte(part)); err != nil {
			return err
		}
	}

	if skipped == len(plan.Scenes) && len(plan.Scenes) > 0 {
		fmt.Fprintf(os.Stdout, "writer: all %d scenes already exist (skipped API); resume from write clears draft by default\n", skipped)
	}
	if err := EnforceChapterLength(rc, plan); err != nil {
		return fmt.Errorf("enforce chapter length: %w", err)
	}
	rc.RecordArtifact("chapter.md", "artifacts/chapter.md", true)
	return nil
}

func loadCharacterStateJSON(rc *runctx.Context) string {
	if rc.App == nil {
		return ""
	}
	v := vault.ForSeries(rc.App, rc.SeriesID)
	state, err := v.LoadCharacterState()
	if err != nil || len(state) == 0 {
		return ""
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func streamScene(ctx context.Context, rc *runctx.Context, ref config.LLMRef, scene artifacts.Scene, episodeNo, maxChars, maxTokens int, briefExcerpt, fixHint, charState string) (string, error) {
	user := prompts.WriterUserWithFix(episodeNo, scene.ID, scene.Title, scene.Goal, maxChars, briefExcerpt, fixHint, charState)
	var b strings.Builder
	usage, err := rc.Providers.LLMForStage(rc.App, "writer").Stream(ctx, llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.WriterSystem,
		User:        user,
		MaxTokens:   maxTokens,
		Temperature: 0.75,
	}, func(chunk string) error {
		b.WriteString(chunk)
		return nil
	})
	if err == nil {
		rc.RecordLLM(usage)
	}
	return b.String(), err
}

func mergeChapterParts(rc *runctx.Context, plan *artifacts.HookPlan) (string, error) {
	var full strings.Builder
	for _, scene := range plan.Scenes {
		partRel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", scene.ID))
		data, err := os.ReadFile(rc.ArtifactPath(partRel))
		if err != nil {
			return "", fmt.Errorf("missing %s: %w", partRel, err)
		}
		full.Write(data)
		full.WriteString("\n")
	}
	return full.String(), nil
}
