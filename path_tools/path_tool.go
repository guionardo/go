package pathtools

import (
	"os"
)

// DirExists simply returns true if the pathName is a existing directory
func DirExists(pathName string) bool {
	stat, err := os.Stat(pathName)
	return err == nil && stat.IsDir()
}

// CreatePath Create full path, with permissions updated from parent folder.
func CreatePath(path string) error {
	if DirExists(path) {
		return nil
	}

	return createPath(path)
}

// FileExists symply returns true if the fileName is a existing file
func FileExists(fileName string) bool {
	stat, err := os.Stat(fileName)
	return err == nil && !stat.IsDir()
}
