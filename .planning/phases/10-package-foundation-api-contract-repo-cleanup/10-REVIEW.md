---
phase: 10-package-foundation-api-contract-repo-cleanup
reviewed: 2026-09-28T14:30:00Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - project_probe/doc.go
  - project_probe/errors.go
  - project_probe/project.go
  - project_probe/probe.go
  - project_probe/registry.go
  - project_probe/ignore.go
  - project_probe/manifest.go
  - project_probe/errors_test.go
  - project_probe/probe_test.go
  - project_probe/registry_test.go
  - project_probe/example_test.go
  - project_probe/manifest_test.go
findings:
  critical: 1
  warning: 2
  info: 2
  total: 5
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-09-28T14:30:00Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

The `projectprobe` package foundation was reviewed against the locked contracts (D-01..D-12, OQ-1, ROBT-02) in 10-CONTEXT.md and the patterns in 10-RESEARCH.md / 10-PATTERNS.md. Structure, style, determinism, the never-fail content path, the 1 MB manifest cap, the 13-entry ignore list, and test discipline (no `t.Parallel()` on registry-mutating tests) all check out; `go test -race`, `go vet` (darwin/linux/windows), and `go build ./...` are green locally.

However, the phase's central deliverable — the D-04 errno→sentinel mapping — is **broken on Windows in 3 of its 4 branches**. The implementation assumes `errors.Is(err, syscall.ENOENT/ENOTDIR/EACCES)` behaves identically on all platforms (research claim A1/Pattern 3), but Go's Windows `syscall` constants are NOT the real Windows error codes: `syscall.EACCES` is an invented `APPLICATION_ERROR+1` value, and `syscall.ENOTDIR` is an alias of `ERROR_PATH_NOT_FOUND`. Verified against the Go 1.27 stdlib source (details in CR-01). The test suite cannot catch this because every errno test uses synthetic `syscall.*` constants (self-consistent on Windows CI) and the only real-filesystem permission test is skipped on Windows. A second robustness gap: `readManifest` can hang forever on a FIFO/special file at the manifest path despite its "regular file" doc claim.

## Critical Issues

### CR-01: D-04 errno mapping is broken on Windows — 3 of 4 branches violate the locked contract

**File:** `project_probe/errors.go:25-36` (and the contract claims in `project_probe/doc.go:29-33`, `project_probe/probe.go:25-37`)

**Issue:** `mapFolderError` classifies `os.Stat`/`os.ReadDir` errors via `errors.Is(err, syscall.ENOENT/ENOTDIR/EACCES/EPERM)`. This is correct on Unix (where the `syscall` constants ARE the errno values and `Errno.Is` handles the rest) but wrong on Windows, verified against Go 1.27 stdlib source:

- `$GOROOT/src/syscall/zerrors_windows.go:8-9`: `ENOENT = ERROR_FILE_NOT_FOUND (2)`, **`ENOTDIR = ERROR_PATH_NOT_FOUND (3)`** — not a "not a directory" code at all.
- `$GOROOT/src/syscall/zerrors_windows.go:15-18`: **`EACCES = APPLICATION_ERROR + iota = 0x20000001`** — an invented value, NOT `ERROR_ACCESS_DENIED (5)`.
- `$GOROOT/src/syscall/syscall_windows.go:189-210`: Windows `Errno.Is` only matches `oserror.ErrPermission/ErrExist/ErrNotExist/ErrUnsupported` targets — never `syscall.ENOENT/ENOTDIR/EACCES` — so only plain equality can match.
- `$GOROOT/src/os/stat_windows.go:27,57,92`: `os.Stat` wraps the **raw** errno (`Errno(2)`, `Errno(3)`, `Errno(5)`, `Errno(267)`) in `*PathError` — no normalization to the syscall constants.

Consequences on Windows (Linux/macOS behave correctly for the same inputs):

1. **`ErrPermissionDenied` is unreachable.** A real permission failure returns `Errno(5)`; `Errno(5) == EACCES (0x20000001)` is false and `Errno(5).Is(EACCES)` is false → falls to the raw wrap (`probe C:\x: Access is denied.`). The EACCES/EPERM arm never fires for real OS errors.
2. **Missing intermediate directory → wrong sentinel.** `Probe("C:\\missing\\parent\\sub")` returns `ERROR_PATH_NOT_FOUND (3)`, which matches the `syscall.ENOTDIR` arm → `ErrNotDirectory`, when the contract (D-04, doc.go) promises `ErrFolderNotFound` for any missing path.
3. **File-in-path → raw error instead of `ErrNotDirectory`.** `Probe("C:\\file.txt\\sub")` returns `ERROR_DIRECTORY (267)` (not exported by `syscall`, matched by no arm) → raw wrapped error, while the same input on Linux/macOS yields `ErrNotDirectory`.

The test suite stays green on Windows CI: `TestMapFolderError` (errors_test.go:29-51) feeds synthetic `syscall.*` constants that equal the constants used inside `mapFolderError` (self-consistent regardless of platform), `TestProbe`'s `missing_folder` row only exercises the single-level case (`TempDir()` exists, final component missing → `Errno(2)` → matches), and `TestProbePermissionDenied` (errors_test.go:61) is explicitly skipped on Windows. The tests encode the research's incorrect assumption A1 ("ERROR_PATH_NOT_FOUND → ENOENT") — that mapping holds for `fs.ErrNotExist` via `Errno.Is`, but not for `errors.Is(err, syscall.ENOENT)`.

**Fix:** classify by platform-independent semantics; use `fs.ErrPermission` (matches EACCES/EPERM on Unix AND `ERROR_ACCESS_DENIED` on Windows via `Errno.Is`), and fold the Windows `ENOTDIR == ERROR_PATH_NOT_FOUND` conflation into the not-found arm:

```go
import "io/fs"

func mapFolderError(folder string, err error) error {
	switch {
	case errors.Is(err, syscall.ENOENT) || (runtime.GOOS == "windows" && errors.Is(err, syscall.ENOTDIR)):
		// On Windows syscall.ENOTDIR == ERROR_PATH_NOT_FOUND — a not-exist
		// error, not a not-a-directory error.
		return fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("probe %s: %w", folder, ErrNotDirectory)
	case errors.Is(err, fs.ErrPermission):
		// EACCES/EPERM on Unix; ERROR_ACCESS_DENIED on Windows.
		return fmt.Errorf("probe %s: %w", folder, ErrPermissionDenied)
	default:
		return fmt.Errorf("probe %s: %w", folder, err)
	}
}
```

Residual (document it or match `syscall.Errno(267)`): the Windows `ERROR_DIRECTORY` file-in-path case still falls to the raw wrap — acceptable as a documented limitation since `Probe`'s `!info.IsDir()` branch already covers the common "path is a file" case before errno mapping is ever reached.

## Warnings

### WR-01: `readManifest` can block forever on a FIFO/special file at the manifest path

**File:** `project_probe/manifest.go:19-35`

**Issue:** The doc comment (lines 16-18) promises "exists, is a regular file, and is at most maxManifestSize bytes", but no regular-file check exists. `os.Open` (O_RDONLY) on a named pipe blocks until a writer appears; a socket/device behaves similarly (or yields garbage reads). A folder containing a FIFO named `go.mod`/`package.json` would hang `readManifest` indefinitely — and since phases 11-13 call this helper on every probed folder, a trivial "hang the probe" DoS becomes reachable from untrusted folder content, contradicting the never-fail spirit (a hang is worse than a failure). The `path_is_directory` test row passes only because reading a directory errors immediately — FIFO/device cases are untested.

**Fix:** stat the opened handle and reject non-regular files before reading:

```go
f, err := os.Open(filepath.Join(folder, name))
if err != nil {
	return nil, false
}
defer f.Close() //nolint: errcheck

info, err := f.Stat()
if err != nil || !info.Mode().IsRegular() {
	return nil, false
}

content, err = io.ReadAll(io.LimitReader(f, maxManifestSize+1))
...
```

### WR-02: Recovered detector panics are silently discarded — no log trace

**File:** `project_probe/registry.go:28-37`

**Issue:** `callDetector` swallows panics and treats them as a non-match with no logging at all. The design skeleton in 10-RESEARCH.md Pattern 2 (line 224) explicitly includes "log at debug level" for the recovery path, and the repo has a `log/slog` per-package logger convention (CONVENTIONS.md §Logging). As written, a panicking detector (phases 11-13 parse untrusted input) is invisible in production — the failure mode the entire panic-recovery design exists for leaves zero diagnostic trail, making detector bugs nearly impossible to find.

**Fix:** log the recovered panic through a package logger, then return the non-match:

```go
defer func() {
	if r := recover(); r != nil {
		log().Debug("detector panic, treating as non-match", slog.Any("panic", r))
		pd, ok = ProjectData{}, false
	}
}()
```

## Info

### IN-01: `Probe("")` error message contains a double space

**File:** `project_probe/probe.go:20`

**Issue:** The empty-input guard produces `fmt.Errorf("probe %s: %w", "", ErrFolderNotFound)` → `"probe : folder not found"`. Cosmetic, but the odd message is the very first thing a consumer hitting the OQ-1 guard sees.

**Fix:** `return ProjectData{}, fmt.Errorf("probe %q: %w", folder, ErrFolderNotFound)` for the empty branch (or a dedicated message), keeping the D-02 wrap shape.

### IN-02: `ExampleProbe` ignores the `MkdirTemp` error

**File:** `project_probe/example_test.go:13`

**Issue:** `dir, _ := os.MkdirTemp("", "probe")` — if temp-dir creation ever fails, `dir` is `""`, `Probe("")` returns `ErrFolderNotFound`, and the example's `// Output` (`err=<nil>`) fails. Practically unreachable today, but it is the only non-deterministic input in the example and contradicts the research anti-pattern note (RESEARCH.md line 284) about deterministic example output.

**Fix:** guard the error:

```go
dir, err := os.MkdirTemp("", "probe")
if err != nil {
	return
}
```

---

_Reviewed: 2026-09-28T14:30:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_