package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// RunHooks 执行某阶段 before/after 的 workflow hooks。
func RunHooks(ctx context.Context, rc *runctx.Context, phase, stageID string, all []workflow.StageHook) error {
	if rc.App == nil {
		return nil
	}
	hooks := workflow.HooksForPhase(all, phase, stageID)
	if len(hooks) == 0 {
		return nil
	}
	v := vault.ForSeries(rc.App, rc.SeriesID)
	for _, h := range hooks {
		for _, action := range h.Actions {
			if err := runHookAction(ctx, rc, v, stageID, action); err != nil {
				return fmt.Errorf("hook %s (%s): %w", action, h.Timing, err)
			}
		}
	}
	return nil
}

func runHookAction(ctx context.Context, rc *runctx.Context, v *vault.SeriesVault, stageID, action string) error {
	_ = ctx
	logger := slog.Default().With("hook", action, "stage", stageID)
	switch action {
	case "inject_l0_series_bible":
		if err := v.Ensure(); err != nil {
			return err
		}
		bible, err := v.LoadBible()
		if err != nil {
			return err
		}
		logger.Info("inject series bible", "chars", len([]rune(bible)))
	case "inject_publish_metrics":
		if rc.EpisodeNo > 1 {
			m, err := v.LoadPreviousPublishMetrics(rc.EpisodeNo)
			if err != nil {
				return err
			}
			if m != nil {
				logger.Info("inject publish metrics", "prev_episode", m.EpisodeNo, "views", m.Views24h)
			}
		}
	case "inject_l1_episode_brief":
		if stageID == "write" && !rc.ArtifactExists("artifacts/episode-brief.md") {
			return fmt.Errorf("episode-brief.md missing (run plan first)")
		}
		logger.Debug("inject episode brief ready")
	case "inject_l2_foreshadows_if_needed":
		if err := v.Ensure(); err != nil {
			return err
		}
		fp := filepath.Join(v.Dir, "foreshadows.yaml")
		if _, err := os.Stat(fp); err == nil {
			logger.Info("foreshadows available", "path", fp)
		}
	case "archive_episode_to_series_vault":
		if err := v.Ensure(); err != nil {
			return err
		}
		if rc.ArtifactExists("artifacts/chapter.md") {
			summary := buildHookEpisodeSummary(rc)
			if err := v.IndexEpisodeSummary(rc.EpisodeNo, summary); err != nil {
				return err
			}
			logger.Info("archived episode summary to vault", "episode", rc.EpisodeNo)
		}
	case "append_chapter_part", "merge_chapter_parts", "update_scene_summary":
		// Writer 流式阶段内部处理；runner 层仅记录已声明。
		logger.Debug("stream hook acknowledged (handled in writer agent)")
	default:
		logger.Warn("unknown hook action, skipped", "action", action)
	}
	return nil
}

func buildHookEpisodeSummary(rc *runctx.Context) string {
	data, err := os.ReadFile(rc.ArtifactPath("artifacts/chapter.md"))
	if err != nil {
		return fmt.Sprintf("# Episode %d\n\n(run %s)\n", rc.EpisodeNo, rc.RunID)
	}
	text := string(data)
	if len([]rune(text)) > 600 {
		runes := []rune(text)
		text = string(runes[:600]) + "…"
	}
	return fmt.Sprintf("# Episode %d summary\n\n%s\n", rc.EpisodeNo, text)
}
