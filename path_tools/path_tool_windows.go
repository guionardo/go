package pathtools

import (
	"os"
)

func createPath(path string) error {
	return os.MkdirAll(path, os.ModeSticky|os.ModePerm)
}
