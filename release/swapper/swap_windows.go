//go:build windows

package main

import (
	"fmt"
	"os"
)

// atomicReplace atomically replaces currentExe with newBinary.
// Backs up currentExe to currentExe + ".bak", then renames newBinary to currentExe.
// The backup is removed on success.
func atomicReplace(currentExe, newBinary string) error {
	backup := currentExe + ".bak"

	os.Remove(backup)

	if err := os.Rename(currentExe, backup); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if err := os.Rename(newBinary, currentExe); err != nil {
		os.Rename(backup, currentExe)
		return fmt.Errorf("replace failed: %w", err)
	}

	os.Remove(backup)
	return nil
}

// relaunch starts a new process with the given binary on Windows.
// Uses os.StartProcess (syscall.Exec is Unix-only).
func relaunch(currentExe string, args, env []string) {
	proc, err := os.StartProcess(currentExe, args, &os.ProcAttr{Env: env, Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
	if err != nil {
		fmt.Fprintf(os.Stderr, "relaunch failed: %v\n", err)
		os.Exit(1)
	}
	proc.Release()
	os.Exit(0)
}
