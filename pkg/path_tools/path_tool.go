package pathtools

import (
	"os"
)

func DirExists(pathName string) bool {
	stat, err := os.Stat(pathName)
	return err == nil && stat.IsDir()
}

// Create full path, with permissions updated from parent folder.
func CreatePath(path string) error {
	if DirExists(path) {
		return nil
	}
	return createPath(path)
}
