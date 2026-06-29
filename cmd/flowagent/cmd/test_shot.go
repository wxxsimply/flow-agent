package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/wanshot"
	"github.com/spf13/cobra"
)

// newTestShotCmd 单镜测试：万相 t2i 关键帧 + i2v 一段 mp4（不跑完整工作流）。
func newTestShotCmd() *cobra.Command {
	var (
		stackName    string
		style2D3D    string
		imagePrompt  string
		motionPrompt string
		duration     int
		outDir       string
	)

	cmd := &cobra.Command{
		Use:   "test-shot",
		Short: "Single-shot test: stack t2i keyframe + i2v clip",
		Long: `Generates one still frame and one short video clip using the stack's image/video providers.
Use this before running the full micro-movie workflow to verify API keys and billing.

Example:
  flowagent test-shot --image-prompt "竖屏9:16，雨夜街道，霓虹反射"
  flowagent test-shot --stack micro-movie-wan-flash --duration 5 --out ./tmp/shot-test
  flowagent test-shot --stack micro-movie-sora --duration 8 --out ./tmp/sora-shot
  flowagent test-shot --stack micro-movie-veo-lite --duration 6 --out ./tmp/veo-shot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, stackName)
			if err != nil {
				return err
			}
			if outDir == "" {
				outDir = filepath.Join(root, "runs", "test-shot-"+time.Now().Format("20060102-150405"))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			start := time.Now()
			res := wanshot.Run(ctx, app, style2D3D, imagePrompt, motionPrompt, duration, outDir)
			fmt.Fprint(os.Stdout, wanshot.FormatReport(res, time.Since(start)))
			if res.Err != nil {
				return res.Err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&stackName, "stack", "micro-movie-wan-flash", "stack profile")
	cmd.Flags().StringVar(&style2D3D, "style", "2d", "animation style: 2d or 3d")
	cmd.Flags().StringVar(&imagePrompt, "image-prompt", "", "text for wan2.6-t2i keyframe")
	cmd.Flags().StringVar(&motionPrompt, "motion-prompt", "", "motion description for i2v")
	cmd.Flags().IntVar(&duration, "duration", 5, "i2v clip duration seconds (2-15)")
	cmd.Flags().StringVar(&outDir, "out", "", "output directory (default runs/test-shot-<timestamp>)")
	return cmd
}
