---
gsd_state_version: "1.0"
milestone: v1.7
milestone_name: Project Probe
current_phase: 12
current_phase_name: TOML Subset + Python/Rust Detectors
status: planning
stopped_at: Phase 11 complete, ready to plan Phase 12
last_updated: "2026-09-29T04:55:47.735Z"
last_activity: 2026-09-29
last_activity_desc: Phase 11 complete, transitioned to Phase 12
state_head: 3cacf1e819674bbb9e3b759b8b176dcfe0efe7a3
progress:
  total_phases: 5
  completed_phases: 9
  total_plans: 6
  completed_plans: 6
  percent: 75
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-29)

**Core value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.
**Current focus:** Phase 12 — TOML Subset + Python/Rust Detectors

## Current Position

Phase: 12 — TOML Subset + Python/Rust Detectors
Plan: Not started
Status: Ready to plan
Last activity: 2026-09-29 — Phase 11 complete, transitioned to Phase 12

Progress: [████████░░] 75%

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
| Phase 11 P02 | 8min | 2 tasks | 5 files |
| Phase 11-text-json-detectors-go-js-ts-php-readme-fallback P03 | 15min | 3 tasks | 8 files |

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

- [Phase 11]: Go detector matches on go.mod presence, not parse success (D-disc-1): a garbage go.mod still yields Language=Go with folder-base Name and empty Version
- [Phase 11]: Registry becomes a 7-position nil-slot literal (D-11): detectGo at index 0, nil at 1/2/3/4/5/6; nil entries skipped in runDetectors before the panic-recover path
- [Phase 11]: Version = raw go directive string, never normalized (D-02/DATA-03): 1.21rc1 stays 1.21rc1; toolchain lines never match the go directive (Pitfall 6)
- [Phase 11]: Rule 3 deviation: plan filenames detect_js.go/detect_js_test.go are excluded from every non-js build — "_js" is a legacy GOARCH in Go implicit file-constraint rules (verified via go/build.MatchFile + go list IgnoredGoFiles); renamed to detect_javascript.go/detect_javascript_test.go, identifier detectJS unchanged — Rule 3 deviation: plan filenames detect_js.go/detect_js_test.go are excluded from every non-js build — "_js" is a legacy GOARCH in Go implicit file-constraint rules (verified via go/build.MatchFile + go list IgnoredGoFiles); renamed to detect_javascript.go/detect_javascript_test.go, identifier detectJS unchanged
- [Phase 11]: D-06 executed as zero-value behavior: no Private field in the JS decode struct; private:true without version yields Version "" pinned by TestProbe_JSPrivateNoVersion — D-06 executed as zero-value behavior: no Private field in the JS decode struct; private:true without version yields Version "" pinned by TestProbe_JSPrivateNoVersion
- [Phase 11]: Parse-success match rule (D-04) proven by integration: broken package.json at position 3 falls through to a valid composer.json at position 6 (TestProbe_CascadePrecedence); Go@0 > JS@3 > PHP@6 first-match pinned — Parse-success match rule (D-04) proven by integration: broken package.json at position 3 falls through to a valid composer.json at position 6 (TestProbe_CascadePrecedence); Go@0 > JS@3 > PHP@6 first-match pinned
- [Phase 11]: Verified json semantics shipped as test rows (T-11-10 accept): duplicate keys last-wins, unknown fields ignored, type mismatch -> decode error -> false -> cascade; "version": null -> "" — Verified json semantics shipped as test rows (T-11-10 accept): duplicate keys last-wins, unknown fields ignored, type mismatch -> decode error -> false -> cascade; "version": null -> ""

### Pending Todos

None yet.

### Blockers/Concerns

- [Planning]: REQUIREMENTS.md claimed 18 v1 requirements; actual count is 21 (3+9+4+5) — traceability updated
- [Phase 10]: Code review CR-01 — D-04 errno mapping needs Windows verification (syscall.EACCES is an invented value on Windows; ENOTDIR aliases ERROR_PATH_NOT_FOUND; ERROR_DIRECTORY unmatched) — recorded advisory in 10-REVIEW-DISPOSITION.md, no failing test on darwin
- [Phase 10]: Pre-existing `make coverage-quick` failure — release/update.go 68.9% vs 70% file threshold (unrelated, logged to deferred-items.md)
- [Phase 11]: Code review WR-01 — isBadgeLine misses plain `![alt](url)` badge form (raw badge becomes Description on common READMEs) — open advisory in 11-REVIEW-DISPOSITION.md
- [Phase 11]: Code review WR-02 — multi-line HTML comment preambles leak into Description — open advisory in 11-REVIEW-DISPOSITION.md
- [Phase 11]: Verification debt — 11-03-SUMMARY.md references go/build.MatchFile (not a disk file) — tracked warning
- [Phase 12]: TOML strict-degrade-to-empty on legal-but-unsupported TOML (dotted keys, multiline strings, inline tables, workspace inheritance) — research flag, needs fixture-driven validation during planning
- [Phase 13]: .NET marker precedence and root-scoped vs 1-level subdir scan — resolved in favor of root-scoped (2-of-3 consensus); confirm during planning

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-09-29
Stopped at: Phase 11 complete, ready to plan Phase 12
Resume file: None
