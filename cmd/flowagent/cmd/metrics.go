package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
	"github.com/spf13/cobra"
)

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Publish metrics for series episodes (post-publish feedback)",
	}
	cmd.AddCommand(newMetricsSetCmd())
	cmd.AddCommand(newMetricsShowCmd())
	cmd.AddCommand(newMetricsListCmd())
	return cmd
}

func newMetricsSetCmd() *cobra.Command {
	var (
		series     string
		episode    int
		runID      string
		views      int64
		completion float64
		likes      int64
		keywords   string
		note       string
		platform   string
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Save playback metrics for an episode",
		Example: `  flowagent metrics set --series demo --episode 1 --views 24000 --completion 0.35 --keywords "爽点,雨夜"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if series == "" || episode <= 0 {
				return fmt.Errorf("--series and --episode are required")
			}
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, "")
			if err != nil {
				return err
			}
			kw := splitKeywords(keywords)
			m := &artifacts.PublishMetrics{
				EpisodeNo:       episode,
				SeriesID:        series,
				Platform:        platform,
				Views24h:        views,
				CompletionRate:  completion,
				Likes:           likes,
				CommentKeywords: kw,
				Note:            note,
			}
			if err := m.Normalize(); err != nil {
				return err
			}
			v := vault.ForSeries(app, series)
			if err := v.SavePublishMetrics(m); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "saved %s\n", v.PublishMetricsPath(episode))

			hints := agent.BuildNextHintsFromMetrics(episode+1, m)
			if err := v.SaveNextEpisodeHints(episode+1, hints); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "updated %s\n", v.NextHintsPath(episode+1))

			if runID != "" {
				rc, err := runctx.NewStore(app.RunsDir).LoadRun(runID)
				if err != nil {
					return err
				}
				data, err := json.MarshalIndent(m, "", "  ")
				if err != nil {
					return err
				}
				if err := rc.WriteArtifact("artifacts/metrics-snapshot.json", data); err != nil {
					return err
				}
				fmt.Fprintf(os.Stdout, "updated run %s artifacts/metrics-snapshot.json\n", runID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&series, "series", "", "series id")
	cmd.Flags().IntVar(&episode, "episode", 0, "episode number")
	cmd.Flags().StringVar(&runID, "run-id", "", "optional: also update run artifacts/metrics-snapshot.json")
	cmd.Flags().Int64Var(&views, "views", 0, "24h view count")
	cmd.Flags().Float64Var(&completion, "completion", 0, "completion rate 0-1")
	cmd.Flags().Int64Var(&likes, "likes", 0, "like count")
	cmd.Flags().StringVar(&keywords, "keywords", "", "comma-separated comment hot words")
	cmd.Flags().StringVar(&note, "note", "", "free-form note")
	cmd.Flags().StringVar(&platform, "platform", "douyin", "platform id")
	_ = cmd.MarkFlagRequired("series")
	_ = cmd.MarkFlagRequired("episode")
	return cmd
}

func newMetricsShowCmd() *cobra.Command {
	var series string
	var episode int
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show saved metrics for an episode",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, "")
			if err != nil {
				return err
			}
			v := vault.ForSeries(app, series)
			m, err := v.LoadPublishMetrics(episode)
			if err != nil {
				return err
			}
			if m == nil {
				fmt.Fprintf(os.Stdout, "no metrics for series=%s episode=%d\n", series, episode)
				return nil
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(m)
		},
	}
	cmd.Flags().StringVar(&series, "series", "", "series id")
	cmd.Flags().IntVar(&episode, "episode", 0, "episode number")
	_ = cmd.MarkFlagRequired("series")
	_ = cmd.MarkFlagRequired("episode")
	return cmd
}

func newMetricsListCmd() *cobra.Command {
	var series string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List episodes with saved publish metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if series == "" {
				return fmt.Errorf("--series is required")
			}
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, "")
			if err != nil {
				return err
			}
			v := vault.ForSeries(app, series)
			eps, err := v.ListPublishMetricsEpisodes()
			if err != nil {
				return err
			}
			if len(eps) == 0 {
				fmt.Fprintf(os.Stdout, "no metrics for series=%s\n", series)
				return nil
			}
			for _, ep := range eps {
				m, err := v.LoadPublishMetrics(ep)
				if err != nil || m == nil {
					continue
				}
				fmt.Fprintf(os.Stdout, "episode %d: views_24h=%d completion=%.2f keywords=%v\n",
					ep, m.Views24h, m.CompletionRate, m.CommentKeywords)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&series, "series", "", "series id")
	_ = cmd.MarkFlagRequired("series")
	return cmd
}

func splitKeywords(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
