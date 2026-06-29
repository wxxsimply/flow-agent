package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/cost"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
	"github.com/spf13/cobra"
)

func newCostCmd() *cobra.Command {
	var (
		runID  string
		asJSON bool
		stack  string
	)
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show cost ledger for a run",
		Example: `  flowagent cost --run-id <uuid>
  flowagent cost --run-id <uuid> --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" {
				return fmt.Errorf("--run-id is required")
			}
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, stack)
			if err != nil {
				return err
			}
			rc, err := runctx.NewStore(app.RunsDir).LoadRun(runID)
			if err != nil {
				return err
			}
			ledger, err := loadCostLedger(rc)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(ledger)
			}
			st := app.Stack
			if rc.Stack != "" {
				stackFile := filepath.Join(root, "config", "stacks", rc.Stack+".yaml")
				if loaded, err := config.LoadStack(stackFile); err == nil {
					st = loaded
				}
			}
			var targets map[string][]float64
			if st != nil {
				targets = st.CostTargetsCNY
			}
			checks := cost.CompareTargets(ledger, targets)
			fmt.Print(cost.FormatReport(runID, ledger, checks))
			if cost.AnyOutOfRange(checks) {
				fmt.Fprintln(os.Stdout, "\n提示: 部分分项不在 cost_targets_cny 目标区间内（过低多为估算单价偏低，过高可减镜头数）。")
			}
			if ledger.TotalCNY == 0 && ledger.LLMPromptTokens == 0 && ledger.ImageCount == 0 {
				fmt.Fprintln(os.Stdout, "\n提示: 账本为空。若 run 在阶段 H 之前完成，需从 plan/produce 等阶段 resume 重跑才会写入用量。")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "run UUID")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output raw cost-ledger JSON")
	cmd.Flags().StringVar(&stack, "stack", "standard-tier", "stack profile for budget targets")
	_ = cmd.MarkFlagRequired("run-id")
	return cmd
}

func loadCostLedger(rc *runctx.Context) (*artifacts.CostLedger, error) {
	if data, err := os.ReadFile(rc.CostLedgerPath()); err == nil {
		var ledger artifacts.CostLedger
		if err := json.Unmarshal(data, &ledger); err == nil {
			return &ledger, nil
		}
	}
	if rc.Manifest != nil && rc.Manifest.Cost != nil {
		return rc.Manifest.Cost, nil
	}
	return nil, fmt.Errorf("read cost ledger: no cost-ledger.json or manifest.cost")
}
