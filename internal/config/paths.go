package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const fileName = "config.json"

func DefaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(root, "launchline", fileName), nil
}
