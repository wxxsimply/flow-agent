//go:build !windows

package web

import "fmt"

func pickFolderDialog(title string) (string, error) {
	_, _ = title
	return "", fmt.Errorf("native folder picker not available on this platform")
}
