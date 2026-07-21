//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func atomicReplace(currentExe, newBinary string) error {
	backup := currentExe + ".bak"

	_ = os.Remove(backup)

	if err := os.Rename(currentExe, backup); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if err := os.Rename(newBinary, currentExe); err != nil {
		_ = os.Rename(backup, currentExe)

		return fmt.Errorf("replace failed: %w", err)
	}

	_ = os.Remove(backup)

	return nil
}

// relaunch replaces the current process with the given binary.
// Uses syscall.Exec on Unix (never returns on success).
//
//nolint:gosec // intentional process replacement by swapper
func relaunch(currentExe string, args, env []string) {
	if err := syscall.Exec(currentExe, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "relaunch failed: %v\n", err)
		os.Exit(1)
	}
}
