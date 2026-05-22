package pathtools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindFileInPath searches for a file in the paths from the PATH environment variable
// returns the first occurrence or error
// Handles OS-specific path separators
func FindFileInPath(filename string) (string, error) {
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") {
		return "", errors.New("filename must not contain path separators or parent directory references")
	}

	pathEnv, ok := os.LookupEnv("PATH")
	if !ok {
		// Try to read PATH from the system environment
		for _, env := range os.Environ() {
			if value, ok := strings.CutPrefix(env, "PATH="); ok {
				pathEnv = value
				break
			}
		}
	}
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
