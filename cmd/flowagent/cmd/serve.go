package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flow-agent/flow-agent/internal/desktop"
	"github.com/flow-agent/flow-agent/internal/web"
	"github.com/spf13/cobra"
)

// newDesktopCmd 以原生窗口打开 Agent Studio（非浏览器）。
func newDesktopCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Open FlowAgent Studio in a native desktop window",
		RunE: func(cmd *cobra.Command, args []string) error {
			return desktop.Run(addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "", "listen address (default: random 127.0.0.1 port)")
	return cmd
}

// newServeCmd 启动本地 Web 服务（浏览器访问）。
func newServeCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve FlowAgent Studio web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveProjectRoot()
			if err != nil {
				return err
			}
			if addr == "" {
				if v := os.Getenv("FLOWAGENT_BIND"); v != "" {
					addr = v
				} else {
					addr = "127.0.0.1:8080"
				}
			}
			srv := web.New(root, addr)
			if err := srv.Start(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "FlowAgent Studio: http://%s/\n", srv.Addr)
			if srv.AuthCfg.Enabled {
				fmt.Fprintf(os.Stdout, "Auth: SMS login enabled (provider=%s)\n", srv.AuthCfg.SMS.Provider)
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "", "listen address (default: FLOWAGENT_BIND or 127.0.0.1:8080)")
	return cmd
}
