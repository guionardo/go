---
status: diagnosed
phase: 16-batch-surface-multi-process-hardening
source: [16-VERIFICATION.md]
started: 2026-10-08T20:21:00Z
updated: 2026-10-09T00:21:22Z
---

## Current Test

[testing complete]

## Tests

### 1. CI two-process E2E matrix run (remote GitHub Actions)
expected: Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows.
result: issue
reported: "failed only on windows test"
severity: blocker

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-16-1
  truth: "Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows."
  status: failed
  reason: "User reported: failed only on windows test"
  severity: blocker
  test: 1
  root_cause: "release/update.go DownloadUpdate removes the still-open os.Create file on the digest-mismatch error path (defer f.Close() runs after Remove) and discards the Remove error; Windows denies delete of an open handle (no FILE_SHARE_DELETE), so the partial 0-byte file remains and TestDownloadUpdate_DigestMismatch require.NoFileExists fails Windows-only. Latent since 2026-07-21; first CI exposure via PR #48."
  artifacts:
    - path: "release/update.go"
      issue: "os.Remove(filePath) while f is still open, on asset.Download error path; error discarded"
    - path: "release/release.go"
      issue: "digest mismatch returns before any write — leftover is the 0-byte created file"
  missing:
    - "Close the file before removing it on the error path (mirror self_update.go close-then-remove)"
    - "Surface the os.Remove error instead of discarding it"
  debug_session: ".planning/debug/ci-two-process-e2e-windows-failure.md"
