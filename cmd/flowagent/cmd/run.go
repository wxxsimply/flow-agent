package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/runner"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
	"github.com/spf13/cobra"
)

// newRunCmd 创建「执行工作流」子命令。
func newRunCmd() *cobra.Command {
	var (
		series    string // 系列 ID，对应 series/<id>/
		episode   int    // 集数
		stack     string // 技术栈配置名，如 standard-tier
		plot      string // 微电影：模糊剧情
		plotFile  string // 微电影：剧情文件
		style2D3D string // 2d | 3d
		orientation string // portrait | landscape
		theme       string // arknights | generic
		bgmMode   string // auto | off
		bgmFile   string // 自定义 BGM 路径
		narratorVoice  string
		targetDuration int
		inputMode      string
		shotsFile      string
		dryRun         bool
		autoGate       bool
		stopAfter      string
	)

	cmd := &cobra.Command{
		Use:   "run [workflow]",
		Short: "Run a content workflow end-to-end",
		Args:  cobra.ExactArgs(1), // 第一个参数为工作流名
		Example: `  flowagent run novel-short-douyin --series demo --episode 1 --dry-run --auto-gate
  flowagent run micro-movie --plot "程序员深夜加班，显示器伸出一只手" --auto-gate --dry-run
  flowagent test-shot --image-prompt "竖屏9:16，雨夜办公室"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wfName := args[0]
			if series == "" && wfName != "micro-movie" {
				return fmt.Errorf("--series is required for workflow %q", wfName)
			}

			root, err := config.FindRoot() // 向上查找 go.mod 定位项目根
			if err != nil {
				return fmt.Errorf("find project root: %w", err)
			}
			if stack == "" {
				stack = "video-native-short"
			}
			wfDir := filepath.Join(root, "docs", "workflows")
			if workflowDir != "" {
				wfDir = workflowDir
			}
			def, err := workflow.Load(wfDir, wfName)
			if err != nil {
				return err
			}
			if def.Context != nil {
				if sp, ok := def.Context["stack_profile"].(string); ok && sp != "" {
					if stack == "" || stack == "video-native-short" {
						stack = sp
					}
				}
			}
			if wfName == "micro-movie" && stack == "video-native-short" {
				stack = "micro-movie-wan-flash"
			}
			app, err := config.Load(root, stack)
			if err != nil {
				return err
			}
			if workflowDir != "" {
				app.WorkflowsDir = workflowDir
			}

			if err := os.MkdirAll(app.RunsDir, 0o755); err != nil {
				return err
			}
			if err := os.MkdirAll(app.SeriesDir, 0o755); err != nil {
				return err
			}

			var userShots []artifacts.UserShotInput
			movieMode := ""
			if wfName == "micro-movie" {
				if series == "" {
					series = "micro-movie"
				}
				if plot == "" && plotFile != "" {
					data, readErr := os.ReadFile(plotFile)
					if readErr != nil {
						return fmt.Errorf("read plot file: %w", readErr)
					}
					plot = string(data)
				}
				if shotsFile != "" {
					data, readErr := os.ReadFile(shotsFile)
					if readErr != nil {
						return fmt.Errorf("read shots file: %w", readErr)
					}
					if err := json.Unmarshal(data, &userShots); err != nil {
						return fmt.Errorf("parse shots file: %w", err)
					}
				}
				movieMode = strings.ToLower(strings.TrimSpace(inputMode))
				if movieMode == "" {
					if len(userShots) > 0 {
						movieMode = "director"
					} else {
						movieMode = "director"
					}
				}
				if movieMode == "director" {
					if plot == "" && !dryRun {
						return fmt.Errorf("director mode requires --plot as opening shot text (第一镜；or use --dry-run)")
					}
				} else if plot == "" && !dryRun {
					return fmt.Errorf("auto mode requires --plot or --plot-file (or use --dry-run)")
				}
			}

			if !dryRun && app.Providers.DeepSeek.APIKey == "" && wfName == "micro-movie" {
				return fmt.Errorf("deepseek api_key required for micro-movie (镜头语言扩写与分镜；或 use --dry-run)")
			}
			if !dryRun && app.Providers.DeepSeek.APIKey == "" && wfName != "micro-movie" {
				return fmt.Errorf("deepseek api_key required for real run (edit config/providers.local.yaml or use --dry-run)")
			}

			store := runctx.NewStore(app.RunsDir)
			rc, err := store.CreateRun(series, episode, wfName, stack, dryRun) // 新建 run_id 目录
			if err != nil {
				return err
			}
			rc.App = app
			rc.InitCostRecorder()
			rc.Providers = provider.NewBundle(app)
			rc.Def = def
			rc.AutoGate = autoGate
			rc.PlotInput = plot
			rc.SeriesDir = filepath.Join(app.SeriesDir, series)
			if wfName == "micro-movie" {
				im := movieMode
				if im == "" {
					im = "director"
				}
				rc.Creative = &artifacts.CreativeOptions{
					InputMode:         im,
					AnimationStyle:    style2D3D,
					Orientation:       orientation,
					VisualTheme:       theme,
					Plot:              plot,
					Shots:             userShots,
					BGMMode:           bgmMode,
					BGMPath:           bgmFile,
					NarratorVoice:     narratorVoice,
					TargetDurationSec: targetDuration,
				}
				rc.Creative.Normalize()
				_ = agent.SaveCreativeOptions(rc)
			}
			if plot != "" {
				_ = rc.WriteArtifact("artifacts/plot-input.md", []byte(plot))
			}

			fmt.Fprintf(os.Stdout, "run_id=%s\nrun_dir=%s\n", rc.RunID, rc.RunDir)
			if rc.Creative != nil {
				fmt.Fprintf(os.Stdout, "style=%s bgm=%s\n", rc.Creative.AnimationStyle, rc.Creative.BGMMode)
			}
			if dryRun {
				fmt.Fprintln(os.Stdout, "mode=dry-run (placeholder artifacts, no external APIs)")
			}

			ctx := context.Background()
			if err := runner.RunWorkflow(ctx, rc, runner.Options{
				DryRun:         dryRun,
				AutoGate:       autoGate,
				StopAfterStage: stopAfter,
			}); err != nil {
				return err
			}
			if stopAfter != "" && rc.Manifest != nil && rc.Manifest.Stage == "awaiting_review" {
				fmt.Fprintf(os.Stdout, "\nPaused for review (stage=%s). Edit artifacts then:\n  flowagent resume --run-id %s --from-stage produce\n", stopAfter, rc.RunID)
				return nil
			}
			fmt.Fprintf(os.Stdout, "\nOK workflow finished.\nartifacts: %s\n", rc.ArtifactsDir)
			fmt.Fprintln(os.Stdout, "key files: episode-brief.md, chapter.md, storyboard.json, master.mp4, publish-pack.json")
			return nil
		},
	}

	cmd.Flags().StringVar(&series, "series", "", "series id (required)")
	cmd.Flags().IntVar(&episode, "episode", 1, "episode number")
	cmd.Flags().StringVar(&stack, "stack", "video-native-short", "stack profile under config/stacks/ (video-native-short = full-motion AI clips)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "skip external APIs; placeholder artifacts")
	cmd.Flags().BoolVar(&autoGate, "auto-gate", false, "auto-approve human gates (dev only)")
	cmd.Flags().StringVar(&plot, "plot", "", "micro-movie: fuzzy plot text")
	cmd.Flags().StringVar(&plotFile, "plot-file", "", "micro-movie: read plot from file")
	cmd.Flags().StringVar(&style2D3D, "style", "2d", "animation style: 2d or 3d")
	cmd.Flags().StringVar(&orientation, "orientation", "portrait", "video orientation: portrait | landscape")
	cmd.Flags().StringVar(&theme, "theme", "arknights", "visual theme: arknights | generic")
	cmd.Flags().StringVar(&bgmMode, "bgm", "auto", "background music: auto (mood from plot) | off")
	cmd.Flags().StringVar(&bgmFile, "bgm-file", "", "use custom BGM mp3 instead of mood library")
	cmd.Flags().StringVar(&narratorVoice, "voice", "epic_male", "narrator voice: epic_male | documentary_male | warm_male | narrator_female")
	cmd.Flags().IntVar(&targetDuration, "target-duration", 150, "target video length in seconds (120-180)")
	cmd.Flags().StringVar(&inputMode, "input-mode", "", "micro-movie: director (逐镜) | auto (LLM分镜)")
	cmd.Flags().StringVar(&shotsFile, "shots-file", "", "micro-movie director: JSON array of shots")
	cmd.Flags().StringVar(&stopAfter, "stop-after", "", "pause after stage (e.g. assemble) for manual review")
	return cmd
}
