---
status: testing
phase: 16-batch-surface-multi-process-hardening
source: [16-VERIFICATION.md]
started: 2026-10-08T20:21:00Z
updated: 2026-10-26T14:26:15Z
---

## Current Test

number: 1
name: CI two-process E2E matrix run (remote GitHub Actions) — G-16-1 re-test
expected: |
  windows-latest "Run tests" leg green (TestDownloadUpdate_DigestMismatch require.NoFileExists now passes);
  "SQLite two-process E2E" step passes on ubuntu, macos and windows;
  "SQLite race detector" step passes on ubuntu + macos and is skipped on windows.
awaiting: remote CI run on the next push (external service)

## Tests

### 1. CI two-process E2E matrix run (remote GitHub Actions)
expected: Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows.
result: pending
reported: "failed only on windows test" (2026-10-09 — before the 16-05 fix)
fix_applied: "Plan 16-05 closed the code side: release/update.go now closes the os.Create handle BEFORE removeFile(filePath), routes removal through the removeFile seam, and surfaces a cleanup failure via errors.Join; TestDownloadUpdate_CleanupFailureSurfaced pins it. Verified 14/14 must-haves (16-VERIFICATION.md, 2026-10-26)."
retest_note: "Only the remote windows-latest leg can confirm; local POSIX runs (all 6 DownloadUpdate tests, coverage gate 82.1%, both e2e arms, race clean) are green."

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps

- gap_id: G-16-1
  truth: "Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows."
  status: fix-applied-awaiting-retest
  reason: "User reported: failed only on windows test"
  severity: blocker
  test: 1
  root_cause: "release/update.go DownloadUpdate removed the still-open os.Create file on the digest-mismatch error path (defer f.Close() ran after Remove) and discarded the Remove error; Windows denies delete of an open handle (no FILE_SHARE_DELETE), so the partial 0-byte file remained and TestDownloadUpdate_DigestMismatch require.NoFileExists failed Windows-only. Latent since 2026-07-21; first CI exposure via PR #48."
  artifacts:
    - path: "release/update.go"
      issue: "os.Remove(filePath) while f is still open, on asset.Download error path; error discarded"
    - path: "release/release.go"
      issue: "digest mismatch returns before any write — leftover is the 0-byte created file"
  missing:
    - "Close the file before removing it on the error path (mirror self_update.go close-then-remove)"
    - "Surface the os.Remove error instead of discarding it"
  resolution:
    - "Plan 16-05 (commit d81bed2) — explicit _ = f.Close() before removeFile(filePath); removeFile = os.Remove seam in a grouped var block; errors.Join surfaces the removal failure"
    - "TestDownloadUpdate_CleanupFailureSurfaced (commit cb2da46) — pins the surfaced cleanup error"
    - "Backstop unchanged: TestDownloadUpdate_DigestMismatch require.NoFileExists remains the Windows CI regression guard"
  debug_session: ".planning/debug/ci-two-process-e2e-windows-failure.md"
