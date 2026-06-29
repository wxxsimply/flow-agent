// Package main 是 flowagent 命令行程序的入口。
package main

import (
	"os"

	"github.com/flow-agent/flow-agent/cmd/flowagent/cmd"
)

// main 启动 CLI；出错时以非零状态码退出。
func main() {
	if err := cmd.Execute(); err != nil { // 执行根命令（run / resume / vault 等）
		os.Exit(1) // 向 shell 返回失败
	}
}
