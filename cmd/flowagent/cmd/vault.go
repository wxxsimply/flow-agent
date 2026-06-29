package cmd

import (
	"fmt"
	"os"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/spf13/cobra"
)

// newVaultCmd 系列知识库相关子命令。
func newVaultCmd() *cobra.Command {
	var series string

	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Series vault utilities",
	}

	search := &cobra.Command{
		Use:   "search",
		Short: "Full-text search series vault (FTS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")

			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, "")
			if err != nil {
				return err
			}

			v := vault.ForSeries(app, series)
			hits, err := v.Search(query)
			if err != nil {
				return fmt.Errorf("vault search: %w", err)
			}
			if len(hits) == 0 {
				fmt.Fprintln(os.Stdout, "(no matches)")
				return nil
			}
			for _, h := range hits {
				fmt.Fprintf(os.Stdout, "[%s] ep=%d %s\n  %s\n\n", h.Kind, h.EpisodeNo, h.Title, h.Snippet)
			}
			return nil
		},
	}
	search.Flags().String("query", "", "search query (required)")
	_ = search.MarkFlagRequired("query")
	cmd.AddCommand(search)

	cmd.PersistentFlags().StringVar(&series, "series", "", "series id")
	_ = cmd.MarkPersistentFlagRequired("series")

	return cmd
}
