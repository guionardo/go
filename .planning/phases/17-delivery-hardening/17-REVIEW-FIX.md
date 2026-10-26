---
phase: 17-delivery-hardening
fixed_at: 2026-10-26T23:40:00Z
review_path: .planning/phases/17-delivery-hardening/17-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 17: Code Review Fix Report

**Fixed at:** 2026-10-26T23:40:00Z
**Source review:** `.planning/phases/17-delivery-hardening/17-REVIEW.md`
**Iteration:** 1
**Scope:** CR-01 and WR-03 only. WR-01, WR-02, WR-04 and IN-01..IN-06 were
deliberately left `open` in the disposition ledger.

**Summary:**
- Findings in scope: 2
- Fixed: 2
- Skipped: 0

## Fixed Issues

### CR-01: Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers

**Files modified:** `cache/bench_test.go`
**Commit:** `55b0f72`
**Applied fix:** Added a one-shot `requireUsable` probe (Set + Get) that runs
before any timing starts and calls `b.Fatalf` if the provider is not usable —
this is what catches `sqlite.New`'s deferred `initErr`, since the constructor
never returns an error. Added `requireNoErr`, called after every `Get`, `Set`,
`Delete` and `GetOrSet` in the hot loops, so a provider that breaks mid-run
fails the benchmark instead of reporting an error return as a nanosecond
number. Sub-benchmark names (`Get`/`Set`/`Delete`/`GetOrSet`) and arm shapes are
unchanged, so all 12 `BenchmarkPrimitives/...` arms and all 48
`BenchmarkBatch/sqlite/...` arms are still produced.

**Verification (ran in the main checkout on `gsd/phase-17-delivery-hardening`; `workflow.use_worktrees` is `false` in this repo):**
- `go test -run XXX -bench 'BenchmarkPrimitives' -benchtime=1x ./cache/` → exit 0, 12 arms emitted.
- `go test -run XXX -bench 'BenchmarkBatch/sqlite/(Memory|File)' -benchtime=1x ./cache/` → 48 arms emitted (Memory and File families intact).
- Broken-path demonstration: a temporary benchmark wrapping
  `sqlite.New(sqlite.WithPath("/nonexistent-dir-xyz/cache.db"))` and calling
  `runPrimitiveBenchmarks` failed with
  `provider unusable: Set failed: cache/sqlite: mkdir /nonexistent-dir-xyz: read-only file system`
  (exit 1) instead of emitting ns/op. The temporary file was deleted immediately
  after the run; nothing from the demonstration is committed.

### WR-03: The CGO-free step prints its evidence but never asserts it

**Files modified:** `.github/workflows/go.yml`
**Commit:** `ad010d2`
**Applied fix:** The `Build (CGO-free)` step (`shell: bash`, name unchanged) now
captures `go env CGO_ENABLED` and exits non-zero with a `::error::` annotation
when the value is not `0`, then runs `go build ./...` and `go vet ./...` under
`set -euo pipefail`, and finally captures
`go list -m modernc.org/sqlite modernc.org/libc` and fails unless both modules
resolve to a pinned version. All four gate-grepped command substrings are
preserved verbatim.

**Verification (ran in the main checkout):**
- The workflow parses as YAML; the step is still named `Build (CGO-free)` and
  still uses `shell: bash`.
- The four literal substrings (`go env CGO_ENABLED`, `go build ./...`,
  `go vet ./...`, `go list -m modernc.org/sqlite modernc.org/libc`) are present.
- Extracted the step's `run` block and executed it locally:
  `CGO_ENABLED=1 bash …` → `::error::CGO_ENABLED=1, expected 0 …`, exit 1;
  `CGO_ENABLED=0 bash …` → pins printed (`modernc.org/sqlite v1.60.1`,
  `modernc.org/libc v1.77.1`), exit 0.
- `golangci-lint run ./cache/...` → `0 issues.`
- `make coverage-quick` → PASS (total 82.1%).

## Skipped Issues

None in scope. WR-01, WR-02, WR-04 and IN-01..IN-06 remain `open` in
`17-REVIEW-DISPOSITION.md` by design.

---

_Fixed: 2026-10-26T23:40:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
