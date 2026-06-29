package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/flow-agent/flow-agent/internal/compareshot"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/spf13/cobra"
)

func newCompareProplockCmd() *cobra.Command {
	var (
		outDir       string
		imagePrompt  string
		motionPrompt string
	)

	cmd := &cobra.Command{
		Use:   "compare-proplock",
		Short: "Compare PropLock i2v: Seedance vs Gemini Veo Lite on the same keyframe",
		Long: `Generates one Seedream keyframe, then runs i2v with micro-movie-seedance (Seedance)
and micro-movie-veo-lite (Gemini Veo 3.1 Lite) using the same PropLock motion prompt.

Requires volcengine.api_key (Seedream + Seedance) and gemini.api_key (Veo).

Example:
  flowagent compare-proplock --out ./tmp/proplock-compare`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			start := time.Now()
			res := compareshot.Run(ctx, root, outDir, imagePrompt, motionPrompt)
			fmt.Fprint(os.Stdout, compareshot.FormatReport(res, time.Since(start)))
			if res.Err != nil {
				return res.Err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out", "", "output directory (default runs/proplock-compare-<timestamp>)")
	cmd.Flags().StringVar(&imagePrompt, "image-prompt", "", "keyframe prompt (default PropLock dagger scene)")
	cmd.Flags().StringVar(&motionPrompt, "motion-prompt", "", "motion prompt with PropLock constraints")
	return cmd
}
