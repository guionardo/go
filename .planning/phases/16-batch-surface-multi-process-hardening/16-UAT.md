---
status: testing
phase: 16-batch-surface-multi-process-hardening
source: [16-VERIFICATION.md]
started: 2026-10-08T20:21:00Z
updated: 2026-10-08T20:21:00Z
---

## Current Test

number: 1
name: CI two-process E2E matrix run (remote GitHub Actions)
expected: |
  Push the milestone branch to GitHub and open the workflow run for the 'SQLite
  two-process E2E' step (go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/
  -v -count=1 -timeout=180s) on ubuntu, macos, and windows, and the 'SQLite
  race detector' step (go test -race ./cache/sqlite/) on the non-Windows
  runners. Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass
  on all three OSes; the race step passes on ubuntu and macos and is skipped on
  windows.
awaiting: user response

## Tests

### 1. CI two-process E2E matrix run (remote GitHub Actions)
expected: Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps