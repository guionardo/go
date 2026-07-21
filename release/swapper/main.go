package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//nolint:cyclop
func main() {
	var newBinary, expectedChecksum string

	flag.StringVar(&newBinary, "new-binary", "", "Path to the new binary (required)")
	flag.StringVar(&expectedChecksum, "checksum", "", "Expected SHA256 hex checksum")
	flag.Parse()

	if newBinary == "" {
		fmt.Fprintln(os.Stderr, "Usage: swapper --new-binary=<path> [--checksum=<sha256>] [original args...]")
		os.Exit(1)
	}

	if strings.Contains(newBinary, "..") || strings.Contains(newBinary, "\x00") {
		fmt.Fprintln(os.Stderr, "invalid --new-binary path: path traversal detected")
		os.Exit(1)
	}

	currentExe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve symlinks: %v\n", err)
		os.Exit(1)
	}

	if expectedChecksum != "" {
		if err := verifyChecksum(newBinary, expectedChecksum); err != nil {
			fmt.Fprintf(os.Stderr, "pre-swap checksum verification failed: %v\n", err)
			os.Exit(1)
		}
	}

	if err := atomicReplace(currentExe, newBinary); err != nil {
		fmt.Fprintf(os.Stderr, "swap failed: %v\n", err)
		os.Exit(1)
	}

	if expectedChecksum != "" {
		if err := verifyChecksum(currentExe, expectedChecksum); err != nil {
			fmt.Fprintf(os.Stderr, "post-swap verification failed: %v\n", err)
			restoreBackup(currentExe)
			os.Exit(1)
		}
	}

	originalArgs := flag.Args()
	relaunchArgs := append([]string{currentExe}, originalArgs...)
	relaunch(currentExe, relaunchArgs, os.Environ())
}

func restoreBackup(currentExe string) {
	backupPath := currentExe + ".bak"

	if _, err := os.Stat(backupPath); err != nil {
		return
	}

	if err := os.Rename(backupPath, currentExe); err != nil {
		fmt.Fprintf(os.Stderr, "backup restore also failed: %v\n", err)
	}
}

func verifyChecksum(filePath, expectedHex string) error {
	//nolint:gosec // path is validated against traversal before being passed here
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])

	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("checksum mismatch: got %s, expected %s", got, expectedHex)
	}

	return nil
}
