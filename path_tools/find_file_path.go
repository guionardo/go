package pathtools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FindFileInPath searches for a file in the paths from the PATH environment variable
// returns the first occurrence or error
// Handles OS-specific path separators
func FindFileInPath(filename string) (string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", errors.New("PATH environment variable not set")
	}

	paths := filepath.SplitList(pathEnv)

	for _, dir := range paths {
		fullPath := filepath.Join(dir, filename)

		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("file '%s' not found in PATH", filename)
}
