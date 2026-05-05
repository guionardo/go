package pathtools

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var pathBaseFunc = path.Base

func init() {
	if runtime.GOOS == "windows" {
		pathBaseFunc = windowsPathBaseFunc
	}
}

// IsRootDirectory returns true if the directory is the root directory
// In windows, the root directory is the drive letter (e.g. C:\)
// In the other OS, the root directory is the root directory (e.g. /)
func IsRootDirectory(directory string) bool {
	return filepath.Clean(directory) == pathBaseFunc(directory)
}

// windowsPathBaseFunc is a replacement for path.Base func
func windowsPathBaseFunc(path string) string {
	const windowsPathDel = "\\"

	path, _ = strings.CutSuffix(path, windowsPathDel)

	paths := strings.Split(path, windowsPathDel)
	if len(paths) > 1 {
		paths = paths[:len(paths)-1]
	} else {
		paths = append(paths, "")
	}

	return strings.Join(paths, windowsPathDel)
}
