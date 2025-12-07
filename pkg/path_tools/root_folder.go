package pathtools

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrNotAGoProject = errors.New("not a golang project")

// GetRootFolder returns the base folder of a golang project (finding the go.mod file)
func GetRootFolder(base string) (rootFolder string, err error) {
	if len(base) == 0 {
		base, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	if !DirExists(base) {
		return "", os.ErrNotExist
	}

	for {
		if FileExists(filepath.Join(base, "go.mod")) {
			return base, nil
		}

		parent := filepath.Dir(base)
		// Stop if we've reached the root directory
		if parent == base {
			break
		}

		base = parent
	}

	return "", ErrNotAGoProject
}
