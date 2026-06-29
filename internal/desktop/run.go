// Package desktop 启动原生 WebView 窗口内的 Agent Studio。
package desktop

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/web"
)

// Run 启动本地 HTTP 服务并在桌面窗口中打开 Agent 前端。
func Run(preferredAddr string) error {
	root, err := config.FindRoot()
	if err != nil {
		return fmt.Errorf("找不到项目目录（请将 FlowAgent.exe 放在含 config/ 的目录旁）: %w", err)
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("chdir project root: %w", err)
	}
	if err := initDesktopDataDir(root); err != nil {
		return fmt.Errorf("init user data dir: %w", err)
	}

	addr := preferredAddr
	if addr == "" {
		addr, err = web.PickListenAddr("127.0.0.1:0")
		if err != nil {
			return err
		}
	}

	srv := web.New(root, addr)
	srv.DesktopMode = true
	if err := srv.Start(); err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	baseURL := srv.BaseURL()
	if err := waitServerReady(baseURL, 15*time.Second); err != nil {
		return err
	}
	return showWindow(baseURL)
}

func initDesktopDataDir(repoRoot string) error {
	if strings.TrimSpace(os.Getenv("FLOWAGENT_DATA_DIR")) == "" {
		userDir, err := config.UserDataDir()
		if err != nil {
			return err
		}
		if err := os.Setenv("FLOWAGENT_DATA_DIR", userDir); err != nil {
			return err
		}
	}
	if err := config.EnsureUserDataDir(); err != nil {
		return err
	}
	return config.MigrateLegacyUserPrefs(repoRoot)
}

func waitServerReady(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "api/config/status")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("本地服务启动超时，请检查端口占用或防火墙")
}
