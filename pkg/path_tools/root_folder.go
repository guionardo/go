package pathtools

import (
	"errors"
	"os"
	"path"
	"runtime"
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

	for len(base) > 0 {
		if FileExists(path.Join(base, "go.mod")) {
			return base, nil
		}

		base = path.Dir(base)
		if (runtime.GOOS == "windows" && len(base) == 3) || len(base) == 1 {
			base = ""
		}
	}

	return "", ErrNotAGoProject
}
