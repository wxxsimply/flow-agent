// FlowAgent Studio 桌面入口：双击 exe 弹出 Agent 前端窗口。
package main

import (
	"os"

	"github.com/flow-agent/flow-agent/internal/console"
	"github.com/flow-agent/flow-agent/internal/desktop"
)

func main() {
	console.EnableUTF8()
	if err := desktop.Run(""); err != nil {
		desktop.ShowError("FlowAgent 启动失败", err.Error())
		os.Exit(1)
	}
}
