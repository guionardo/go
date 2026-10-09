---
phase: 16-batch-surface-multi-process-hardening
fixed_at: 2026-10-08T23:42:54Z
review_path: .planning/phases/16-batch-surface-multi-process-hardening/16-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 16: Code Review Fix Report

**Fixed at:** 2026-10-08T23:42:54Z
**Source review:** `.planning/phases/16-batch-surface-multi-process-hardening/16-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (fix_scope: `critical_warning` — warnings only; the 5 Info findings are out of scope)
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: `retryBusyWithin` can overrun its budget by a full extra `fn` call

**Files modified:** `cache/sqlite/retry.go`, `cache/sqlite/retry_internal_test.go`
**Commit:** `aa0f554`
**Status:** fixed: requires human verification
**Applied fix:** Hoisted the absolute-deadline check to the top of the loop and carried `lastErr` across iterations: once the budget is spent, no further `fn` attempt is started — exhaustion returns the last busy error instead of invoking `fn` once more past the deadline (which, for a `PingContext` blocking in the SQLite busy handler, could double the documented 5 s bound). The post-call combined condition still surfaces a busy error from an in-flight attempt that crosses the deadline. The doc comment now states the guarantee ("no further fn call is started once the budget is spent"). Added the regression subtest `budget_crossed_during_backoff_starts_no_extra_attempt` (backoff 100 ms > budget 10 ms, always-busy `fn`): it asserts exactly one `fn` call and a busy-classified exhaustion error — the pre-fix loop performed a second post-deadline call. The fix changes retry-loop semantics in a subtle timing path, so a human should confirm the behavior matches the D-06/doc.go intent before the phase proceeds to verification.

### WR-02: `chunksOf` hangs forever (infinite loop, unbounded allocation) on `size <= 0`

**Files modified:** `cache/sqlite/batch.go`, `cache/sqlite/batch_internal_test.go`
**Commit:** `829acb3`
**Status:** fixed
**Applied fix:** Added the `size <= 0` guard to `chunksOf` — a non-positive size now yields no chunks (`nil`), so callers issue no query instead of looping forever on `start += size` (which never advances). The guard mirrors the existing empty-input early return and matches the reviewer-suggested shape; both production call sites keep the fixed `chunkSize` constant, and the guard makes a future config-read refactor (D-01: "adjustable later") safe against a zero-value default. Doc comment updated. Extended `TestChunksOf` with `zero_size_no_chunks` and `negative_size_no_chunks` cases pinning the guard (both hang on the pre-fix loop).

## Verification

- Per-fix Tier 1 (re-read of every modified hunk) and Tier 2 (`gofmt -l` clean, `go build ./cache/sqlite/`, `go vet ./cache/sqlite/`, focused `go test ./cache/sqlite/`) passed for both fixes — 15 tests for the WR-01 change, 34 for the WR-02 change.
- Project gate `make coverage-quick` passed for both commits: total coverage 82.0% (WR-01 state) and 81.9% (WR-02 state), with file ≥70%, package ≥80%, and total ≥75% thresholds all satisfied.
- **Gate environment note:** the gates ran in the isolated review-fix worktree, not the main checkout. A fresh worktree lacks the gitignored generated swapper binaries that `release` embeds (`go:embed release/swapper/*`), so `make swapper` was run in the worktree first to produce those build artifacts (they remain gitignored — nothing extra was committed). With that prerequisite, the gate is reproducible from the worktree tree as committed. The coverage/verification numbers above reflect the worktree-env run.

---

_Fixed: 2026-10-08T23:42:54Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_