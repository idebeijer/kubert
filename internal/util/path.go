package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ and environment variables in file paths.
// Only a lone "~" or a path that starts with "~/" is treated as the
// current user's home directory. Forms like "~otheruser/..." are left
// unchanged so they are not silently rewritten against the wrong home.
func ExpandPath(path string) (string, error) {
	// Expand environment variables first
	path = os.ExpandEnv(path)

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory for path %s: %w", path, err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory for path %s: %w", path, err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
