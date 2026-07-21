---
plan: 02
status: complete
---

# Plan 04-02 Summary: Swapper Binary

## Objective
Create the standalone swapper binary that performs atomic binary replacement with backup/rollback and cross-platform relaunch.

## Files Created

| File | Purpose |
|------|---------|
| `release/swapper/main.go` | Entry point with flag parsing (`--new-binary`, `--checksum`), path traversal validation, orchestration: verify → swap → re-verify → relaunch |
| `release/swapper/swap_unix.go` | `atomicReplace` with backup/restore + `relaunch` using `syscall.Exec` (build tag: `!windows`) |
| `release/swapper/swap_windows.go` | `atomicReplace` with backup/restore + `relaunch` using `os.StartProcess` (build tag: `windows`) |
| `release/swapper/main_test.go` | Tests for `verifyChecksum` (valid/ invalid/ nonexistent), `atomicReplace` success, and backup restore on failure |

## Verification Results

- **Build:** linux/amd64 ✓ darwin/amd64 ✓ darwin/arm64 ✓ windows/amd64 ✓
- **Tests (with -race):** 3/3 passed ✓
- **golangci-lint:** clean ✓

## Key Decisions Implemented
- D-01: Two-phase verification — pre-swap and post-swap SHA256 checksum verification
- D-02: Backup restored on any failure
- D-04: `--new-binary` + `--checksum` flags, original args via `flag.Args()`
- D-05: spawn → exit → swap → exec flow (relaunch replaces process on Unix, spawns child on Windows)
- D-06: Backup old binary before swap (`currentExe.bak`), restore on failure
- D-07: Backup cleaned up after successful swap
- D-13: Errors to stderr, exit code 1 on failure
- D-14: Cross-platform via build tags

## Threat Mitigations Applied
- T-04-05: Path traversal detection (reject `..` and null bytes in `--new-binary`)
- T-04-06: Symlink resolution via `filepath.EvalSymlinks` before rename
- T-04-07: Backup removed on success, restored on failure
- T-04-09: Short-lived process — concurrent swap self-correcting

## Functions

| Function | File | Signature |
|----------|------|-----------|
| `main` | `main.go` | `func main()` |
| `verifyChecksum` | `main.go` | `func verifyChecksum(filePath, expectedHex string) error` |
| `restoreBackup` | `main.go` | `func restoreBackup(currentExe string)` |
| `atomicReplace` | `swap_unix.go` / `swap_windows.go` | `func atomicReplace(currentExe, newBinary string) error` |
| `relaunch` | `swap_unix.go` | `func relaunch(currentExe string, args, env []string)` (Unix: `syscall.Exec`) |
| `relaunch` | `swap_windows.go` | `func relaunch(currentExe string, args, env []string)` (Windows: `os.StartProcess`) |
