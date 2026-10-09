package projectprobe

import (
	"errors"
	"fmt"
	"syscall"
)

var (
	// ErrFolderNotFound is returned when the probed folder does not exist.
	ErrFolderNotFound = errors.New("probe: folder not found")

	// ErrNotDirectory is returned when the probed path is not a directory.
	ErrNotDirectory = errors.New("probe: not a directory")

	// ErrPermissionDenied is returned when the probed folder cannot be
	// accessed due to OS permission checks.
	ErrPermissionDenied = errors.New("probe: permission denied")
)

// mapFolderError maps stat/readdir failures onto the sentinel contract
// (D-04): ENOENT → ErrFolderNotFound, ENOTDIR → ErrNotDirectory, EACCES or
// EPERM → ErrPermissionDenied, everything else → the raw error wrapped with
// the failing folder path (D-02).
func mapFolderError(folder string, err error) error {
	switch {
	case errors.Is(err, syscall.ENOENT):
		return fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("probe %s: %w", folder, ErrNotDirectory)
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
		return fmt.Errorf("probe %s: %w", folder, ErrPermissionDenied)
	default:
		return fmt.Errorf("probe %s: %w", folder, err)
	}
}
