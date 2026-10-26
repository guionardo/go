---
status: complete
phase: 17-Delivery Hardening
source: [17-VERIFICATION.md]
started: 2026-10-26T23:12:00Z
updated: 2026-10-26T23:29:27.672Z
---

## Tests

### 1. Three-OS CGO-free CI matrix run
expected: Push this branch (or open the PR) and inspect the `Run tests` workflow run. All three legs
green; `Build (CGO-free)` log contains `CGO_ENABLED=0` and both pinned module lines; ubuntu + macOS
`SQLite race detector` green; windows skips the race step.
result: pass
evidence: |
  PR #52 → workflow run 40419738438 (success).

  - ubuntu-latest: `Build (CGO-free)` success — log shows `CGO_ENABLED=0`,
    `modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`; `SQLite two-process E2E` success;
    `SQLite race detector` success
  - macos-latest: `Build (CGO-free)` success; `SQLite two-process E2E` success;
    `SQLite race detector` success
  - windows-latest: `Build (CGO-free)` success — log shows `CGO_ENABLED=0` and
    `modernc.org/sqlite v1.60.1`; `SQLite two-process E2E` success;
    `SQLite race detector` skipped (expected — no C toolchain for `-race`)
  - `Go test coverage check` job: success
note: |
  First run (40419313142) failed on the ubuntu leg in 19s — `Build (CGO-free)` ran
  `go build ./...` before `make swapper`, and `release/embed_swapper.go` embeds the
  generated `swapper/swapper_*` binaries. Fixed in 7306e6c (see G-17-1) and re-run green.

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-17-1
  truth: "The three-OS CI matrix builds and tests CGO-free on Linux, macOS and Windows"
  status: resolved
  severity: blocker
  test: 1
  reason: "User reported: CI ubuntu leg failed — `go build ./...` in the new `Build (CGO-free)` step ran before `make swapper`, but release/embed_swapper.go //go:embed requires the generated swapper binaries."
  root_cause: "Step ordering: `Build swapper binaries` was placed after `Build (CGO-free)`. The dependency was invisible locally because the gitignored swapper artifacts already existed in the developer worktree."
  resolved_by: "commit 7306e6c — `fix(17): build swapper binaries before the CGO-free build gate` (moved `make swapper` ahead of the build step, still before `Run tests`)"
  resolved_at: 2026-10-26
  verification: "run 40419738438 — all four jobs green; CGO_ENABLED=0 and both pins visible on ubuntu + windows logs; race green on ubuntu/macOS, skipped on windows"
  artifacts: [".github/workflows/go.yml"]
  missing: []
