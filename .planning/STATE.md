---
gsd_state_version: "1.0"
milestone: v1.8
milestone_name: SQLite Cache Backend
current_phase: "17.1"
current_phase_name: Fix error sentinel identity and MGet error propagation
status: Not planned (inserted gap-closure phase)
stopped_at: Phase 17.1 context gathered
last_updated: "2026-10-27T01:25:30.303Z"
last_activity: 2026-10-26
last_activity_desc: Phase 17 complete
state_head: fa6392c58f31bf231c545e5f6e54c3254f0a0a93
progress:
  total_phases: 4
  completed_phases: 15
  total_plans: 13
  completed_plans: 11
  percent: 85
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-10-26)

**Core value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.
**Current focus:** Phase 17 — Delivery Hardening

## Current Position

Phase: 17.1 (Fix error sentinel identity and MGet error propagation) — READY TO EXECUTE
Plan: Not started
Status: Not planned (inserted gap-closure phase)
Last activity: 2026-10-26 — Phase 17 complete

Progress: [█████████░] 85%

## Performance Metrics

**Velocity:**

- Total plans completed: 43 (27 through v1.6 + 16 in v1.7)
- Average duration: ~12 min (v1.7 phases 10-14)

**By Phase:** *(empty — no v1.8 plans completed yet)*

**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 10 P01 | 1 min | 1 tasks | 12 files |
| Phase 10 P02 | 15 min | 3 tasks | 10 files |
| Phase 10-package-foundation-api-contract-repo-cleanup P03 | 5min | 2 tasks | 2 files |
| Phase 11 P01 | 19min | 3 tasks | 6 files |
| Phase 11 P02 | 8min | 2 tasks | 5 files |
| Phase 11-text-json-detectors-go-js-ts-php-readme-fallback P03 | 15min | 3 tasks | 8 files |
| Phase 12-toml-subset-python-rust-detectors P01 | 7min | 2 tasks | 2 files |
| Phase 12-toml-subset-python-rust-detectors P02 | 6min | 2 tasks | 4 files |
| Phase 12-toml-subset-python-rust-detectors P03 | 8min | 2 tasks | 6 files |
| Phase 12-toml-subset-python-rust-detectors P04 | 37min | 2 tasks | 2 files |
| Phase 13 P01 | 7min | 2 tasks | 4 files |
| Phase 13 P02 | 9min | 2 tasks | 6 files |
| Phase 14 P01 | 6min | 2 tasks | 21 files |
| Phase 14 P02 | 6min | 3 tasks | 7 files |
| Phase 14-semantics-hardening-release-polish P03 | 8min | 2 tasks | 1 files |
| Phase 14 P04 | 34 | 3 tasks | 3 files |
| Phase 15 P01 | 14 min | 3 tasks | 11 files |
| Phase 15 P02 | 18 min | 3 tasks | 6 files |
| Phase 15 P03 | 9 min | 3 tasks | 10 files |
| Phase 16 P01 | 4min | 2 tasks | 2 files |
| Phase 16-batch-surface-multi-process-hardening P02 | 23 | 3 tasks | 6 files |
| Phase 16-batch-surface-multi-process-hardening P03 | 20 | 3 tasks | 5 files |
| Phase 16 P04 | 9min | 3 tasks | 3 files |
| Phase 16 P05 | 15min | 2 tasks | 2 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.8 Research]: Stack fixed — `modernc.org/sqlite` v1.60.1 (pure-Go, CGO-free, SQLite 3.53.4) + exact `modernc.org/libc` v1.77.1 pin; no other new runtime dependencies
- [v1.8 Research]: DSN uses validated mattn-compatible shorthand keys (`_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate`) with a mandatory pragma read-back test; never interpolate caller input into the DSN
- [v1.8 Research]: Pool pinned to one connection (MaxOpenConns=1, MaxIdleConns=1) for both file and `:memory:` modes — the single most important correctness decision
- [v1.8 Research]: Schema = fixed 3-column `cache_entries` (`cache_key TEXT PRIMARY KEY`, `value TEXT`, `expires_at INTEGER NULL`) + partial expiry index; `auto_vacuum=NONE` recommended; opt-in optimize helper instead of default VACUUM
- [Carried from v1.7]: Repo forbids empty commits; RED ships with its implementation in the feat commit; `make coverage-quick` green repo-wide (80.7%) must not regress
- [Phase 15]: cache/sqlite foundation ships DSN-carried pragmas, pinned 1-connection pool, BEGIN IMMEDIATE bootstrap, deferred initErr taxonomy (PROV-01..04, STOR-01..04, STOR-06)
- [Phase 15]: Per-commit coverage gate forced pure unit tests into the tracer commit (66.4% -> 84.9%); tasks 2-3 still own behavioral tests
- [Phase 15]: modernc.org/libc v1.77.1 transitively requires x/tools v0.50.0 — the wider x/* MVS bumps from go mod tidy are mandatory, not incidental
- [Phase 15]: Sweeper lifecycle: startSweeper() extraction keeps sweepLoop() ctx-free while satisfying contextcheck; stop/done channels exist only when the interval is positive (nil for d <= 0, D-08)
- [Phase 15]: CloseFunc order: CAS guard -> close(stop) + <-done -> db.Close(); sidecar removal only via the engine's last-connection checkpoint (no provider file deletion)
- [Phase 15]: TTL semantics proven filter-only: reads never delete rows (raw COUNT stays 1), reclamation only via open sweep + opt-in ticker; absolute UnixNano bound now in SQL
- [Phase 15]: Pre-existing concurrent first-open WAL conversion race deferred to Phase 16 (DI-15-01, deferred-items.md): journal_mode=WAL conversion can return immediate SQLITE_BUSY; retry policy belongs to CONC-01/02 per 15-RESEARCH
- [Phase 15]: Optimizable ships as an sqlite-side batchCache adapter returned by New (the frozen root cannot forward provider methods); separate type declarations with one justified //nolint:decorder honor the plan acceptance greps while keeping golangci-lint at 0 issues
- [Phase 15]: Blocked checkpoint shape empirically verified before tests — PRAGMA wal_checkpoint(TRUNCATE) returns err=nil with busy=1 after the 5s busy timeout; the three-column scan plus a busy!=0 descriptive error is the shipped implementation
- [Phase 15]: WithAutoCheckpoint pages>0 appends _pragma=wal_autocheckpoint(strconv.Itoa) in file mode only; read-back proves 200 configured / 1000 default / memory ignored; maintenance SQL lives only in optimize.go (VACUUM prohibition grep holds)
- [Phase 15]: STOR-07 closed — all Phase 15 requirement families (PROV/STOR/TTL) green repo-wide with no cache/sqlite coverage override; package at 92.5%, total 81.5%, Windows CGO-free build and golangci-lint clean
- [Phase 16]: 16-01: Two-process WAL spike confirmed the D-06 retry policy (5 ms backoff / 5 s budget / busy-only Code()&0xff==5 at the PingContext seam): raw arm failed 8/10 runs with immediate SQLITE_BUSY (DI-15-01 reproduced cross-process), policy arm ran 30/30 clean with journal_mode=wal, integrity_check=ok and exactly 301 rows per role
- [Phase 16]: 16-01: Harness children use raw database/sql handles with spike-local DSN/schema/SQL mirrors (plus a Rule 3 schema-bootstrap addition); the mirrors are eliminated in Plan 16-04 when children rewire to the shipped provider
- [Phase 16]: 16-01: Raw arm is observational only (SQLITE_E2E_RETRY=0, never a gate); the e2e build tag keeps the harness invisible to the unit lane and leaves coverage metrics unchanged (no cache/sqlite override)
- [Phase 16-batch-surface-multi-process-hardening]: TDD commit pattern under the repo constraint (no empty commits; per-commit coverage gate): in-session assertion-level RED evidence ships with its implementation in feat commits; test-only hardening commits as test(16-02)
- [Phase 16-batch-surface-multi-process-hardening]: journal_size_limit ships as a buildDSN parameter + compile-time constant 64 << 20, no new exported option (OQ1 resolution); non-positive appends nothing
- [Phase 16-batch-surface-multi-process-hardening]: Classifier uses named constants sqliteBusyCode/sqliteBusyMask over the plan's literal &0xff==5 shape — identical semantics, mnd-clean
- [Phase 16-batch-surface-multi-process-hardening]: TDD commit pattern per the repo constraint: in-session assertion-level RED evidence (stub-then-real, verified RED_EVIDENCE_OK) ships with the implementation in feat commits; test-only hardening commits as test(16-03)
- [Phase 16-batch-surface-multi-process-hardening]: TestBatchPlaceholderSmoke removed in Task 1 (MSet/MDel legs depended on then-stub methods — Rule 3); TestBatchMGet covers MGet there; TestBatchSemantics lands as the full replacement in Task 3
- [Phase 16-batch-surface-multi-process-hardening]: msetTx/mgetChunk helper extractions keep cyclop/sqlclosecheck clean while preserving the single-transaction MSet seam and close-before-next-chunk row handling
- [Phase 16]: 16-04: parent raw verification decodes the provider's JSON-encoded value column (json.Unmarshal); the old raw-handle spike compared plain strings and could never match
- [Phase 16]: 16-04: crash child blocks on a timer loop (not bare select {}) so the Go deadlock detector never terminates it on its own before the parent kills it
- [Phase 16]: 16-04: contention row-count contract pinned to 287 per role with the 14-distinct-deletion derivation in the constant's doc comment; deleted-key spot checks 0000/0007 (MDel) + 0050 (point)
- [Phase 16]: 16-04: CI race step gated to runner.os != 'Windows' (C toolchain); Windows -race broadening recorded as a reviewer-flagged follow-up in the workflow comment
- [Phase 16]: 16-05: The close-before-remove + errors.Join half of G-16-1 was already on the branch (a prior PR); this plan completed the removeFile = os.Remove test seam, routed the error-path removal through it, and normalized the message to 'removing partial file %s: %w' — the fs.ErrNotExist guard and io/fs import were dropped to match the plan's exact spec
- [Phase 16]: 16-05: The removeFile seam block sits after the type declarations but before func (f optionFunc) apply to satisfy decorder type->var->func order; the plan's 'keep it where it is' position (after the apply method) tripped dec-order the moment the line became new
- [Phase 16]: UAT 1/1 passed — the single checkpoint (CI two-process E2E matrix / G-16-1 re-test) was answered by PR #51 CI run 40340478439: windows 'Run tests' green (close-before-remove fixed DigestMismatch's NoFileExists), E2E executed on all three OSes, race green on ubuntu/macos and skipped on windows
- [Phase 16]: Security verified 13/13 threats closed at ASVS L1 (register authored at plan time → L1 short-circuit, no auditor spawn); SECURITY.md recorded retroactively (no prior file existed)

### Pending Todos

- [2026-10-08] [database] Spike modernc SQLite WAL with two concurrent processes — [todo file](.planning/todos/pending/2026-10-08-spike-modernc-sqlite-wal-with-two-concurrent-processes.md)

### Blockers/Concerns

- [Phase 16 — resolved]: Two-process `SQLITE_BUSY` behavior is now proven — 16-01 raw arm failed 8/10 runs (DI-15-01 reproduced cross-process) while the bounded busy-only policy arm ran 30/30 clean, and 16-04's CI E2E (contention + crash arms) is green on ubuntu/macos/windows. CONC-01/CONC-02 closed.
- [Phase 16 — reviewer-flagged follow-up]: Windows `-race` broadening — the race step is gated to `runner.os != 'Windows'` (no C toolchain); recorded as a follow-up in the `.github/workflows/go.yml` comment.
- [Phase 16 — pre-existing, tracked]: `release/update.go` joins untrusted `Asset.Name` into `os.Create` without sanitization (path traversal, defense-in-depth). Predates Phase 16, no requirement covers it; recorded `open` in `16-REVIEW-DISPOSITION.md`.
- [Deferred — tracked]: Phase 10 CR-01 — D-04 errno mapping needs Windows verification (syscall.EACCES invented on Windows) — deferred with rationale in 14-AUDIT.md (needs Windows CI evidence)
- [Deferred — tracked]: Phase 10 WR-02 — panic logging at registry dispatch — deferred with rationale in 14-AUDIT.md (dev-experience nicety, not correctness)
- [Non-blocking]: verification-debt SUMMARY metadata warnings (files not on disk) — 16-02/16-03 SUMMARYs cite a stale `internal_probe/two_process_test.go` path; tracked in /gsd-progress /gsd-audit-uat

### Roadmap Evolution

- Phase 17.1 inserted after Phase 17: Fix error sentinel identity and MGet error propagation (URGENT)

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-10-27T00:43:12.472Z
Stopped at: Phase 17.1 context gathered
Resume file: .planning/phases/17.1-fix-error-sentinel-identity-and-mget-error-propagation/17.1-CONTEXT.md

## Operator Next Steps

- Phase 16 complete (5/5 plans, UAT 1/1, security 13/13) — PR #51 open and green
- Discuss Phase 17: `/gsd-discuss-phase 17`
- Or plan it directly: `/gsd-plan-phase 17`
