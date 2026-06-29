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
	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunPublisher 生成 publish-pack.json 与封面，处理发布人工门禁。
func RunPublisher(rc *runctx.Context) error {
	pack, err := buildPublishPack(rc)
	if err != nil {
		return err
	}
	if err := pack.Validate(); err != nil {
		return fmt.Errorf("publish pack: %w", err)
	}
	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/publish-pack.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("publish-pack.json", "artifacts/publish-pack.json", true)

	if err := extractCover(rc); err != nil {
		slog.Warn("cover extract failed", "err", err)
	} else {
		rc.RecordArtifact("cover.jpg", "artifacts/cover.jpg", false)
	}

	if rc.AutoGate {
		rc.SetGate("final_cut_approved", true)
		rc.SetGate("publish_authorized", true)
	}
	return nil
}

func buildPublishPack(rc *runctx.Context) (*artifacts.PublishPackDoc, error) {
	if rc.DryRun {
		return publishPackDryRun(rc), nil
	}
	if pack, err := publishPackLive(rc); err == nil {
		return pack, nil
	} else {
		slog.Warn("publish llm failed, using template", "err", err)
	}
	return publishPackTemplate(rc), nil
}

func publishPackDryRun(rc *runctx.Context) *artifacts.PublishPackDoc {
	return publishPackTemplate(rc)
}

func publishPackTemplate(rc *runctx.Context) *artifacts.PublishPackDoc {
	title, desc, tags := templateFromArtifacts(rc)
	return &artifacts.PublishPackDoc{
		EpisodeNo:   rc.EpisodeNo,
		SeriesID:    rc.SeriesID,
		Title:       title,
		Description: desc,
		Hashtags:    tags,
		VideoPath:   "artifacts/master.mp4",
		CoverPath:   "artifacts/cover.jpg",
	}
}

func publishPackLive(rc *runctx.Context) (*artifacts.PublishPackDoc, error) {
	if rc.Providers == nil {
		return nil, fmt.Errorf("providers not initialized")
	}
	if rc.App == nil || strings.TrimSpace(rc.App.Providers.DeepSeek.APIKey) == "" {
		return nil, fmt.Errorf("deepseek api_key required for publish copy")
	}

	brief := readArtifactSnippet(rc, "artifacts/episode-brief.md", 1200)
	hookLine := hookLineFromPlan(rc)
	subtitle := firstStoryboardSubtitle(rc)

	ref := rc.App.LLMRef("planner")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := completeJSONWithRetry(ctx, rc.Providers.LLMForStage(rc.App, "planner"), llm.CompletionRequest{
		Model:       modelOrDefault(ref, "deepseek-v4-flash"),
		System:      prompts.PublisherSystem,
		User:        prompts.PublisherUser(rc.EpisodeNo, rc.SeriesID, brief, hookLine, subtitle),
		MaxTokens:   1024,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}
	rc.RecordLLM(res.Usage)

	var out struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Hashtags    []string `json:"hashtags"`
	}
	if err := json.Unmarshal([]byte(ExtractTopLevelJSON(res.Text)), &out); err != nil {
		return nil, fmt.Errorf("parse publish json: %w", err)
	}
	pack := &artifacts.PublishPackDoc{
		EpisodeNo:   rc.EpisodeNo,
		SeriesID:    rc.SeriesID,
		Title:       strings.TrimSpace(out.Title),
		Description: strings.TrimSpace(out.Description),
		Hashtags:    out.Hashtags,
		VideoPath:   "artifacts/master.mp4",
		CoverPath:   "artifacts/cover.jpg",
	}
	if pack.Title == "" {
		return nil, fmt.Errorf("empty title from llm")
	}
	return pack, nil
}

func templateFromArtifacts(rc *runctx.Context) (title, desc string, tags []string) {
	hook := hookLineFromPlan(rc)
	title = fmt.Sprintf("【第%d集】%s", rc.EpisodeNo, truncateRunes(hook, 18))
	if title == fmt.Sprintf("【第%d集】", rc.EpisodeNo) {
		title = fmt.Sprintf("【第%d集】雨夜反转｜下集揭晓真相", rc.EpisodeNo)
	}
	desc = "连续短剧日更，关注追更下集。"
	if brief := readArtifactSnippet(rc, "artifacts/episode-brief.md", 200); brief != "" {
		desc = truncateRunes(strings.ReplaceAll(brief, "\n", " "), 180)
	}
	tags = []string{"小说推文", "短剧", "爽文", "都市言情", "AI解说", "追更", "反转"}
	return title, desc, tags
}

func hookLineFromPlan(rc *runctx.Context) string {
	if !rc.ArtifactExists("artifacts/hook-plan.json") {
		return ""
	}
	plan, err := artifacts.LoadHookPlan(rc.ArtifactPath("artifacts/hook-plan.json"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(plan.HookLine)
}

func firstStoryboardSubtitle(rc *runctx.Context) string {
	if !rc.ArtifactExists("artifacts/storyboard.json") {
		return ""
	}
	sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil || len(sb.Shots) == 0 {
		return ""
	}
	return strings.TrimSpace(sb.Shots[0].Subtitle)
}

func readArtifactSnippet(rc *runctx.Context, rel string, maxRunes int) string {
	data, err := os.ReadFile(rc.ArtifactPath(rel))
	if err != nil {
		return ""
	}
	s := strings.TrimPrefix(string(data), "<!-- dry-run -->\n")
	return truncateRunes(s, maxRunes)
}

func extractCover(rc *runctx.Context) error {
	video := rc.ArtifactPath("artifacts/master.mp4")
	out := rc.ArtifactPath("artifacts/cover.jpg")
	if !ffmpeg.IsRealVideoFile(video) {
		return fmt.Errorf("master.mp4 is placeholder, skip cover")
	}
	at := 2.0
	if sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json")); err == nil && len(sb.Shots) > 0 {
		at = sb.Shots[0].DurationSec * 0.5
		if at < 0.5 {
			at = 0.5
		}
	}
	return ffmpeg.ExtractCoverFrame(video, out, at)
}
