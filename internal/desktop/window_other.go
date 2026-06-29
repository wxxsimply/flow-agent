//go:build !windows

package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func showWindow(url string) error {
	for _, bin := range []string{"google-chrome", "chromium", "microsoft-edge"} {
		if p, err := exec.LookPath(bin); err == nil {
			cmd := exec.Command(p, "--app="+url)
			if err := cmd.Start(); err != nil {
				return err
			}
			fmt.Println("FlowAgent Studio 已打开。按 Ctrl+C 退出服务。")
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
			return nil
		}
	}
	return exec.Command("xdg-open", url).Start()
}
