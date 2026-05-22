package pathtools

import (
	"os"
)

func createPath(path string) error {
	return os.MkdirAll(path, 0750)
}
