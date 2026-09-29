---
gsd_state_version: 1.0
milestone: v1.7
milestone_name: Project Probe
current_phase: 11
current_phase_name: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback
status: executing
stopped_at: Completed 11-01-PLAN.md
last_updated: "2026-09-29T04:10:55.946Z"
last_activity: 2026-09-29
last_activity_desc: Phase 11 execution started
progress:
  total_phases: 5
  completed_phases: 1
  total_plans: 6
  completed_plans: 4
  percent: 20
state_head: e9a3369736e1ecffc42a289f64abc275d3649de3
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-29)

**Core value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.
**Current focus:** Phase 11 — Text/JSON Detectors — Go, JS/TS, PHP + README Fallback

## Current Position

Phase: 11 (Text/JSON Detectors — Go, JS/TS, PHP + README Fallback) — EXECUTING
Plan: 2 of 3
Status: Ready to execute
Last activity: 2026-09-29 — Phase 11 execution started

Progress: [███████░░░] 67%

## Performance Metrics

**Velocity:**

- Total plans completed: 27 (through v1.6)
- Average duration: ~10 min (v1.6 phases 8-9)

**By Phase:** *(empty — no v1.7 plans completed yet)*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 10 P01 | 1 min | 1 tasks | 12 files |
| Phase 10 P02 | 15 min | 3 tasks | 10 files |
| Phase 10-package-foundation-api-contract-repo-cleanup P03 | 5min | 2 tasks | 2 files |
| Phase 11 P01 | 19min | 3 tasks | 6 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Research]: Unknown-without-error contract — error reserved for hard I/O failures; detectors return `(ProjectData, bool)`, never error-as-control-flow
- [Research]: Manifest-first ordered cascade, first match wins, root-scoped only (no subdir probing)
- [Research]: Stdlib-only — unexported TOML-subset reader for pyproject/Cargo; full TOML dep deferred to framework milestone (v2)
- [Research]: go.mod `go` directive reported as Version with "toolchain floor, not release version" semantics — decision record lands in Phase 11
- [Research]: Detector order Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP — confirm during Phase 13 planning
- [Phase 10]: No empty git commit for FND-01: deleting untracked files produces no diff; the 'own commit' clause is satisfied by task isolation + go build verification (RESEARCH OQ-2); untracked reality documented for plan 10-02's first commit message — Repo forbids empty commits; git cannot represent deletion of untracked files
- [Phase 10]: RED verified-but-uncommitted per plan TDD adaptation (10-02 precedent): pre-commit go-test hook runs go test ./... and CI must stay green; failing boundary tests ship with the implementation; RED evidence recorded in the feat commit message
- [Phase 10]: readManifest boundary tests pin the cap via maxManifestSize (single source of truth) and use literal EF BB BF bytes for BOM rows — tests stay independent of the implementation var while pinning the ROBT-02 contract
- [Phase 10]: Probe("") returns ErrFolderNotFound (OQ-1 resolution) — never silently probes cwd
- [Phase 10]: syscall errno mapping with EACCES **or EPERM** → ErrPermissionDenied (D-04 amended 2026-09-28 after code review) — Unix permission failures report either errno
- [Phase 11]: WR-01 fix shape amended: plain os.Open + f.Stat is insufficient — open() itself blocks on a FIFO; O_NONBLOCK open required before the regular-file gate can run

D-08 exact-case enforced against real directory entries (os.ReadDir): os.Open alone resolves case-insensitively on macOS/Windows volumes
isBadgeLine strips innermost [..](..) segments via the first '](' + nearest prior '[' so wrapped [![..](..)](..) badges strip to empty
Commit scope (11) per Phase 10 precedent and plan acceptance criteria (git log --oneline -1 checks)

### Pending Todos

None yet.

### Blockers/Concerns

- [Planning]: REQUIREMENTS.md claimed 18 v1 requirements; actual count is 21 (3+9+4+5) — traceability updated
- [Phase 10]: Code review CR-01 — D-04 errno mapping needs Windows verification (syscall.EACCES is an invented value on Windows; ENOTDIR aliases ERROR_PATH_NOT_FOUND; ERROR_DIRECTORY unmatched) — recorded advisory in 10-REVIEW-DISPOSITION.md, no failing test on darwin
- [Phase 10]: Code review WR-01 — readManifest can block on a FIFO/special file; needs regular-file gate (phases 11-13 consume it)
- [Phase 10]: Pre-existing `make coverage-quick` failure — release/update.go 68.9% vs 70% file threshold (unrelated to this phase, logged to deferred-items.md)
- [Phase 12]: TOML strict-degrade-to-empty on legal-but-unsupported TOML (dotted keys, multiline strings, inline tables, workspace inheritance) — research flag, needs fixture-driven validation during planning
- [Phase 13]: .NET marker precedence and root-scoped vs 1-level subdir scan — resolved in favor of root-scoped (2-of-3 consensus); confirm during planning

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-09-29T04:10:40.608Z
Stopped at: Completed 11-01-PLAN.md
Resume file: None
