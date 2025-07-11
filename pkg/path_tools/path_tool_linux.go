package pathtools

import (
	"os"
	"syscall"
)

func createPath(path string) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)
	return os.MkdirAll(path, os.ModeSticky|os.ModePerm)
}
