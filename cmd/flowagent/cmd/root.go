// Package cmd 实现 flowagent 的 Cobra 子命令。
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/flow-agent/flow-agent/internal/console"
	"github.com/spf13/cobra"
)

var (
	workflowDir string // 覆盖默认工作流目录 docs/workflows
	verbose     bool   // 是否输出 debug 日志
)

// Execute 注册并运行根命令。
func Execute() error {
	root := &cobra.Command{
		Use:           "flowagent",
		Short:         "FlowAgent — novel stream → video → publish",
		SilenceUsage:  true,  // 出错时不自动打印 usage
		SilenceErrors: true,  // 错误由 RunE 返回给调用方
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			console.EnableUTF8()
			level := slog.LevelInfo // 默认 info
			if verbose {
				level = slog.LevelDebug // -v 时更详细
			}
			// 使用 stdout：PowerShell 会把 stderr 里的 INFO 当成 NativeCommandError 标红
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
		},
	}

	root.PersistentFlags().StringVar(&workflowDir, "workflow-dir", "", "override docs/workflows directory")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "debug logs")

	root.AddCommand(newConfigCmd())  // API Key：init / check
	root.AddCommand(newRunCmd())     // 跑完整工作流
	root.AddCommand(newResumeCmd())  // 从某阶段续跑
	root.AddCommand(newVaultCmd())   // 系列知识库
	root.AddCommand(newCostCmd())     // 成本账本
	root.AddCommand(newMetricsCmd())  // 发布指标回流
	root.AddCommand(newVersionCmd()) // 版本号
	root.AddCommand(newTestShotCmd()) // 单镜测试
	root.AddCommand(newCompareProplockCmd())
	root.AddCommand(newServeCmd())    // 本地 Web UI
	root.AddCommand(newDesktopCmd())  // 桌面 Agent 窗口

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err) // 真实错误写到 stderr
		return err
	}
	return nil
}

// newVersionCmd 打印当前骨架版本。
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("flowagent 0.1.0 (skeleton)")
		},
	}
}
