---
phase: 16-batch-surface-multi-process-hardening
reviewed: 2026-10-26T14:19:38Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - release/update.go
  - release/update_test.go
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 16: Code Review Report

**Reviewed:** 2026-10-26T14:19:38Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed the phase 16 gap-closure wave for `DownloadUpdate` (plan 16-05, UAT gap G-16-1). The wave
adds a close-before-remove error path, routes cleanup through the `removeFile` test seam, and surfaces
a removal failure via `errors.Join`. The core change is correct: the file handle is closed before
`os.Remove` (necessary on Windows), the removal failure is joined with the download error, and the new
test `TestDownloadUpdate_CleanupFailureSurfaced` correctly pins the surfaced error via
`require.ErrorIs`. No critical issues were found in the changed lines themselves; one pre-existing
security hardening gap (unsanitized asset filename) and three minor quality items are reported below.

## Warnings

### WR-01: Unsanitized `asset.Name` enables path traversal / arbitrary file write

**File:** `release/update.go:139`
**Issue:** `filePath := filepath.Join(targetDir, asset.Name)` passes a filename sourced from remote,
attacker-influenceable GitHub release metadata straight into `os.Create` (line 140) without validation.
`filepath.Join` does not neutralize `..` segments or absolute paths, so an asset named
`../../../tmp/evil_linux_amd64.tar.gz` (which still satisfies `findAsset`'s `Contains(goos/goarch)`
matching) escapes `targetDir` and writes to an arbitrary filesystem location. The existing
`TestDownloadUpdate_CreateError` (which feeds `Name: "sub/..."`) confirms that path separators in the
name are accepted by design rather than rejected. This is pre-existing and defense-in-depth — the
self-update flow already trusts the release to supply an executable binary — but as an exported library
function the `DownloadUpdate` contract ("write inside `targetDir`") is silently violated.
**Fix:** Validate the name is a clean base name before joining, e.g.:

```go
name := asset.Name
if name == "" || filepath.Base(name) != name || filepath.IsAbs(name) || name == "." || name == ".." {
    return "", fmt.Errorf("invalid asset name: %q", name)
}
filePath := filepath.Join(targetDir, name)
```

## Info

### IN-01: Redundant double-close of the file handle on the error path

**File:** `release/update.go:144,150`
**Issue:** The explicit `_ = f.Close()` on the download-error branch is followed by the already-registered
`defer func() { _ = f.Close() }()` (line 144) when the function returns. The second `Close` is a harmless
no-op returning an ignored error, but the deliberate first close is easy to misread as a missing-close
fix when the `defer` already covers it. Not a bug, but the control flow is now split across two close
sites.
**Fix:** Consolidate the close into a single site. For example, drop the `defer` and make both the
success and error paths close explicitly, or keep the `defer` and document why the early close must
precede `removeFile` (as the existing comment already does) — just avoid implying the defer is not there.

### IN-02: `ctx` parameter is unused in `DownloadUpdate`

**File:** `release/update.go:129`
**Issue:** `DownloadUpdate(ctx context.Context, ...)` accepts a context but never uses it. The actual
fetch happens in `Asset.Download` (`release/release.go:128`), which calls `downloadClient.Get(...)` with
no context, so a caller-supplied deadline or cancellation is silently ignored and a hung download cannot
be aborted. Pre-existing, not introduced by this wave.
**Fix:** Thread the context into the download, e.g. change `Asset.Download` to
`Download(ctx context.Context, w io.Writer)` and use `http.NewRequestWithContext(ctx, http.MethodGet, ...)`,
or remove the unused parameter if cancellation is out of scope.

### IN-03: `removeFile` test seam is a mutable package global without the shared mutex

**File:** `release/update.go:32`, `release/update_test.go:495`
**Issue:** `removeFile` is swapped directly in `TestDownloadUpdate_CleanupFailureSurfaced` without the
`mu sync.Mutex` that guards the sibling global `githubAPIBase`. It is currently race-free only because
the test omits `t.Parallel()` (sequential tests complete before parallel tests run) and every other
`DownloadUpdate` test runs under `t.Parallel()` afterward. This invariant is implicit and fragile: a
future parallel test that also swaps `removeFile` would introduce a data race.
**Fix:** Document the invariant on the seam (mirror the existing `//nolint:paralleltest` rationale) or
guard the swap with the existing `mu`, or prefer an injectable seam (function field on `Config`) so tests
do not mutate shared package state at all.

---

_Reviewed: 2026-10-26T14:19:38Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
