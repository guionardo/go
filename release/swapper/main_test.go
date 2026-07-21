package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	content := []byte("hello swapper test")
	filePath := filepath.Join(dir, "test.bin")

	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(content)
	expectedHex := hex.EncodeToString(sum[:])

	if err := verifyChecksum(filePath, expectedHex); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := verifyChecksum(filePath, "badchecksum"); err == nil {
		t.Error("expected error for bad checksum, got nil")
	}

	if err := verifyChecksum(filepath.Join(dir, "nonexistent"), expectedHex); err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestAtomicReplace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	currentExe := filepath.Join(dir, "current.exe")
	newBinary := filepath.Join(dir, "new.exe")

	if err := os.WriteFile(currentExe, []byte("current version"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(newBinary, []byte("new version"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := atomicReplace(currentExe, newBinary); err != nil {
		t.Fatalf("atomicReplace failed: %v", err)
	}

	//nolint:gosec // reading temp test file at known path
	data, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "new version" {
		t.Errorf("expected 'new version', got %q", string(data))
	}

	if _, err := os.Stat(newBinary); !os.IsNotExist(err) {
		t.Errorf("expected new binary to be removed, stat err: %v", err)
	}

	if _, err := os.Stat(currentExe + ".bak"); !os.IsNotExist(err) {
		t.Errorf("expected backup to be removed on success, stat err: %v", err)
	}
}

func TestRestoreBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	current := filepath.Join(dir, "current.exe")
	backup := current + ".bak"

	err := os.WriteFile(current, []byte("current"), 0o600)
	require.NoError(t, err)

	err = os.WriteFile(backup, []byte("backup"), 0o600)
	require.NoError(t, err)

	os.Remove(current)
	restoreBackup(current)

	data, err := os.ReadFile(current)
	require.NoError(t, err)
	require.Equal(t, "backup", string(data))
}

func TestRestoreBackup_NoBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	current := filepath.Join(dir, "current.exe")

	restoreBackup(current)

	_, err := os.Stat(current)
	require.True(t, os.IsNotExist(err))
}

func TestAtomicReplace_BackupRestoreOnFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	currentExe := filepath.Join(dir, "current.exe")
	newBinary := filepath.Join(dir, "nonexistent/new.exe")

	if err := os.WriteFile(currentExe, []byte("current version"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := atomicReplace(currentExe, newBinary)
	if err == nil {
		t.Fatal("expected error for nonexistent source, got nil")
	}

	//nolint:gosec // reading temp test file at known path
	data, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "current version" {
		t.Errorf("expected original content to be preserved, got %q", string(data))
	}
}
