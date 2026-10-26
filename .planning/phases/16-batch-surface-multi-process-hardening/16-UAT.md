---
status: complete
phase: 16-batch-surface-multi-process-hardening
source: [16-VERIFICATION.md]
started: 2026-10-08T20:21:00Z
updated: 2026-10-26T17:01:36.762Z
---

## Current Test

[testing complete]

## Tests

### 1. CI two-process E2E matrix run (remote GitHub Actions)
expected: Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows.
result: pass
verified_at: 2026-10-26
evidence: "PR #51 CI run 40340478439 (conclusion: success) — ubuntu-latest: Run tests ✓, SQLite two-process E2E ✓, SQLite race detector ✓; macos-latest: Run tests ✓, E2E ✓, race ✓; windows-latest: Run tests ✓ (TestDownloadUpdate_DigestMismatch green — G-16-1 reconciled), E2E ✓ (executed, no cascade-skip), race skipped by design; Go test coverage check ✓."
reported: "failed only on windows test" (2026-10-09 — before the 16-05 fix)
fix_applied: "Plan 16-05 closed the code side: release/update.go now closes the os.Create handle BEFORE removeFile(filePath), routes removal through the removeFile seam, and surfaces a cleanup failure via errors.Join; TestDownloadUpdate_CleanupFailureSurfaced pins it. Verified 14/14 must-haves (16-VERIFICATION.md, 2026-10-26)."

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-16-1
  truth: "Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows."
  status: resolved
  resolved_by: 16-05-PLAN.md
  resolved_at: 2026-10-26
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
