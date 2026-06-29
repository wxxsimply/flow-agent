//go:build windows

package web

import (
	"fmt"
	"os/exec"
	"strings"
)

func pickFolderDialog(title string) (string, error) {
	escaped := strings.ReplaceAll(title, "'", "''")
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = '%s'
$d.ShowNewFolderButton = $true
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.SelectedPath }
`, escaped)
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) > 0 {
			return "", fmt.Errorf("folder picker: %s", strings.TrimSpace(string(exit.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
