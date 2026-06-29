package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
	"github.com/flow-agent/flow-agent/internal/wanshot"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration utilities",
	}

	cmd.AddCommand(newConfigCheckCmd())
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigTestAPICmd())
	cmd.AddCommand(newConfigTestKlingCmd())
	cmd.AddCommand(newConfigTestKlingTextCmd())
	cmd.AddCommand(newConfigTestWanVideoCmd())
	cmd.AddCommand(newConfigTestSoraCmd())
	cmd.AddCommand(newConfigTestVolcengineTTSCmd())
	return cmd
}

func newConfigTestVolcengineTTSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-volcengine-tts",
		Short: "Probe Volcengine OpenSpeech TTS with multiple voice types",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return err
			}
			results := tts.ProbeVoices(providers)
			fmt.Fprint(os.Stdout, tts.FormatProbeReport(providers, results))
			probeStacks := []string{"micro-movie-seedance", "micro-movie-cap5", "micro-movie-wan-flash"}
			anyStackOK := false
			for _, name := range probeStacks {
				app, err := config.Load(root, name)
				if err != nil {
					continue
				}
				ok, detail := provider.ProbeStackTTS(app)
				fmt.Fprintf(os.Stdout, "--- Stack TTS probe (%s) ---\n", name)
				fmt.Fprint(os.Stdout, provider.FormatStackTTSProbeReport(ok, detail))
				if ok {
					anyStackOK = true
				}
			}
			if anyStackOK {
				return nil
			}
			for _, r := range results {
				if r.OK {
					return nil
				}
			}
			return fmt.Errorf("volcengine tts probe failed")
		},
	}
}

// newConfigTestWanVideoCmd 单镜万相图生视频冒烟（等同 test-shot，输出到临时目录）。
func newConfigTestWanVideoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-wan-video",
		Short: "Smoke test: one Wanxiang t2i frame + i2v clip (same as test-shot)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			app, err := config.Load(root, "micro-movie-wan-flash")
			if err != nil {
				return err
			}
			if app.Providers.DashScope.APIKey == "" {
				return fmt.Errorf("dashscope api_key is empty; edit config/providers.local.yaml")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			outDir := filepath.Join(os.TempDir(), "flowagent-wan-video-test")
			start := time.Now()
			res := wanshot.Run(ctx, app, "2d", "", "", 5, outDir)
			fmt.Fprint(os.Stdout, wanshot.FormatReport(res, time.Since(start)))
			if res.Err != nil {
				return res.Err
			}
			return nil
		},
	}
}

// newConfigTestSoraCmd 单镜 Sora 图生视频冒烟（micro-movie-sora stack）。
func newConfigTestSoraCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "test-sora",
		Short: "Smoke test: Wan t2i keyframe + OpenAI Sora i2v (micro-movie-sora stack)",
		Long: `Verifies openai.api_key and image provider before a full micro-movie run.
Requires openai.api_key (or OPENAI_API_KEY) plus dashscope for wan2.6-t2i keyframes.

Example:
  flowagent config test-sora
  flowagent config test-sora --out ./tmp/sora-test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, config.FormatSoraReadinessReport(providers))
			if missing := config.SoraStackReadiness(providers); len(missing) > 0 {
				return fmt.Errorf("sora stack not ready: %s", missing[0])
			}
			app, err := config.Load(root, "micro-movie-sora")
			if err != nil {
				return err
			}
			if outDir == "" {
				outDir = filepath.Join(root, "runs", "sora-smoke-"+time.Now().Format("20060102-150405"))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			start := time.Now()
			res := wanshot.Run(ctx, app, "2d", "竖屏9:16，王殿内国王端坐王座，电影感", "小幅连贯动作", 8, outDir)
			fmt.Fprint(os.Stdout, wanshot.FormatReport(res, time.Since(start)))
			if res.Err != nil {
				return res.Err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outDir, "out", "", "output directory")
	return cmd
}

// newConfigCheckCmd 检查 API Key 是否已填写（不调用外网）。
func newConfigCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check API keys in providers.local.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return fmt.Errorf("read %s: %w\n请先运行: flowagent config init", localPath, err)
			}
			report := config.FormatCheckReport(localPath, config.ValidateProviders(providers))
			if providers.DashScope.APIKey != "" {
				report += fmt.Sprintf("dashscope resolved base_url: %s\n", providers.ResolveDashScopeBaseURL())
			}
			fmt.Fprint(os.Stdout, report)
			if !providers.OpenAIEnabled() {
				fmt.Fprint(os.Stdout, "\n--- Sora stack (micro-movie-sora) ---\n")
				fmt.Fprint(os.Stdout, config.FormatSoraReadinessReport(providers))
			}

			// deepseek 为 MVP 最小必需
			for _, s := range config.ValidateProviders(providers) {
				if s.Name == "deepseek" && !s.OK {
					return fmt.Errorf("deepseek api_key is required for next steps")
				}
			}
			return nil
		},
	}
}

// newConfigInitCmd 从 example 复制 providers.local.yaml（不覆盖已有文件）。
func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create providers.local.yaml from example if missing",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			dst := filepath.Join(root, "config", "providers.local.yaml")
			if _, err := os.Stat(dst); err == nil {
				fmt.Fprintf(os.Stdout, "already exists: %s\n", dst)
				fmt.Fprintln(os.Stdout, "edit it and run: flowagent config check")
				return nil
			}
			src := filepath.Join(root, "config", "providers.local.yaml.example")
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dst, data, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "created: %s\n", dst)
			fmt.Fprintln(os.Stdout, "请编辑文件填入 API Key，然后运行: flowagent config check")
			return nil
		},
	}
}

// newConfigTestAPICmd 探测各官方百炼端点，找出当前 Key 可用的地域。
func newConfigTestAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-api",
		Short: "Test Bailian (DashScope) endpoints with your API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return err
			}
			if providers.DashScope.APIKey == "" {
				return fmt.Errorf("dashscope api_key is empty; edit %s first", localPath)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			var results []config.BailianTestResult
			recommended := ""
			for _, reg := range config.AllBailianRegions() {
				r := config.TestBailianChat(ctx, providers.DashScope.APIKey, reg)
				results = append(results, r)
				if r.OK && recommended == "" {
					recommended = string(reg)
				}
			}
			fmt.Fprint(os.Stdout, config.FormatBailianTestReport(results, recommended))
			if recommended == "" {
				return fmt.Errorf("no endpoint succeeded; check Key 地域与百炼控制台是否一致")
			}
			return nil
		},
	}
}

// newConfigTestKlingTextCmd 探测文生视频可用的 model_name。
func newConfigTestKlingTextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-kling-text",
		Short: "Probe Kling text2video model_name values for your account",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return err
			}
			if !providers.KlingEnabled() {
				return fmt.Errorf("kling.access_key / secret_key not configured")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			base := strings.TrimRight(providers.Kling.BaseURL, "/")
			results := config.ProbeKlingText2VideoModels(ctx, providers, base)
			fmt.Fprint(os.Stdout, config.FormatKlingText2VideoProbeReport(results))
			for _, r := range results {
				if r.OK {
					return nil
				}
			}
			return fmt.Errorf("no working text2video model_name on %s", base)
		},
	}
}

// newConfigTestKlingCmd 验证可灵 AK/SK 与 API 域名（K2 动画前置检查）。
func newConfigTestKlingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-kling",
		Short: "Test Kling JWT auth and account balance (K2 image-to-video)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.FindRoot()
			if err != nil {
				return err
			}
			localPath := filepath.Join(root, "config", "providers.local.yaml")
			providers, err := config.LoadProviders(localPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			results := config.ProbeKling(ctx, providers)
			fmt.Fprint(os.Stdout, config.FormatKlingProbeReport(results, providers.Kling.AccessKey))
			for _, r := range results {
				if r.OK {
					return nil
				}
			}
			return fmt.Errorf("kling authentication failed on all endpoints")
		},
	}
}
