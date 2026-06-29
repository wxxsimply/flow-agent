//go:build !windows

package desktop

func showError(title, message string) {
	_, _ = title, message
}
