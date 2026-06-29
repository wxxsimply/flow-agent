//go:build windows

package console

import "golang.org/x/sys/windows"

// EnableUTF8 将控制台输入/输出设为 UTF-8（避免 PowerShell 5 中文乱码）。
func EnableUTF8() {
	const cpUTF8 = 65001
	_ = windows.SetConsoleOutputCP(cpUTF8)
	_ = windows.SetConsoleCP(cpUTF8)
}
