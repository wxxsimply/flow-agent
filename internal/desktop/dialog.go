package desktop

// ShowError 在桌面环境弹出错误提示（Windows 为消息框）。
func ShowError(title, message string) {
	showError(title, message)
}
