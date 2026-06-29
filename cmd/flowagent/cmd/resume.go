package cmd

import (
	"context"
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
	"github.com/spf13/cobra"
)

// newResumeCmd 从已有 run 的指定阶段继续执行（API 中断或改纲后重跑）。
func newResumeCmd() *cobra.Command {
	var (
		runID       string
		from        string
		stack       string
		dryRun      bool
		autoGate    bool
		keepChapter bool
	)

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume a workflow run from a stage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" || from == "" {
				return fmt.Errorf("--run-id and --from-stage are required")
			}

			root, err := config.FindRoot()
			if err != nil {
				return err
			}

			store := runctx.NewStore(filepath.Join(root, "runs"))
			rc, err := store.LoadRun(runID)
			if err != nil {
				return err
			}

			stackName := resolveStackForResume(stack, rc)
			app, err := config.Load(root, stackName)
			if err != nil {
				return err
			}
			rc.Stack = stackName

			wfDir := app.WorkflowsDir
			if workflowDir != "" {
				wfDir = workflowDir
			}
			def, err := workflow.Load(wfDir, rc.Workflow)
			if err != nil {
				return err
			}
			if !dryRun && app.Providers.DeepSeek.APIKey == "" {
				return fmt.Errorf("deepseek api_key required for real run (edit config/providers.local.yaml or use --dry-run)")
			}

			rc.App = app
			agent.LoadCreativeOptionsFromRun(rc)
			rc.InitCostRecorder()
			rc.Providers = provider.NewBundle(app)
			rc.Def = def
			rc.AutoGate = autoGate
			rc.SeriesDir = filepath.Join(app.SeriesDir, rc.SeriesID)

			if err := agent.PrepareResumeFromStage(rc, from, keepChapter); err != nil {
				return fmt.Errorf("prepare resume: %w", err)
			}

			if from == "produce" {
				cp := agent.LoadProduceCheckpointForResume(rc)
				if len(cp.CompletedShots) > 0 {
					fmt.Fprintf(os.Stdout, "produce checkpoint: skip %d completed shot(s): %s\n",
						len(cp.CompletedShots), strings.Join(cp.CompletedShots, ", "))
				}
			}

			fmt.Fprintf(os.Stdout, "resuming run_id=%s from stage=%s stack=%s\n", runID, from, stackName)
			ctx := context.Background()
			if err := runner.RunWorkflow(ctx, rc, runner.Options{
				FromStage: from,
				DryRun:    dryRun,
				AutoGate:  autoGate,
			}); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "\nOK workflow finished.\nartifacts: %s\n", rc.ArtifactsDir)
			if rc.Manifest != nil && rc.Manifest.Cost != nil && rc.Manifest.Cost.TotalCNY > 0 {
				fmt.Fprintf(os.Stdout, "total_cost_cny=%.2f (see: flowagent cost --run-id %s)\n",
					rc.Manifest.Cost.TotalCNY, runID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&runID, "run-id", "", "existing run uuid")
	cmd.Flags().StringVar(&from, "from-stage", "", "stage id to resume from")
	cmd.Flags().StringVar(&stack, "stack", "", "stack profile (default: use stack saved in run manifest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "dry run")
	cmd.Flags().BoolVar(&autoGate, "auto-gate", false, "auto-approve human gates")
	cmd.Flags().BoolVar(&keepChapter, "keep-chapter", false, "keep existing chapter.parts when resuming from write")
	return cmd
}

// resolveStackForResume 续跑时优先使用 manifest 中记录的 stack。
func resolveStackForResume(flagStack string, rc *runctx.Context) string {
	saved := strings.TrimSpace(rc.Stack)
	flag := strings.TrimSpace(flagStack)
	if flag != "" && flag != "video-native-short" {
		return flag
	}
	if saved != "" {
		return saved
	}
	if rc.Workflow == "micro-movie" {
		return "micro-movie-wan-flash"
	}
	if flag != "" {
		return flag
	}
	return "video-native-short"
}
