package pathtools

import (
	"os"
)

const directoryPermission = 0750

func createPath(path string) error {
	return os.MkdirAll(path, directoryPermission)
}
