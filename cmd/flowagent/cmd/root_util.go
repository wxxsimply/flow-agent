package cmd

import (
	"fmt"

	"github.com/flow-agent/flow-agent/internal/config"
)

func resolveProjectRoot() (string, error) {
	root, err := config.FindRoot()
	if err != nil {
		return "", fmt.Errorf("find project root: %w", err)
	}
	return root, nil
}
