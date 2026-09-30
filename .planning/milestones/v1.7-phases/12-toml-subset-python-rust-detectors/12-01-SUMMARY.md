---
phase: 12-toml-subset-python-rust-detectors
plan: 01
subsystem: project-probe
tags: [toml, parser, strict-degrade, python, rust, pyproject.toml, cargo.toml]

# Dependency graph
requires:
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readManifest (1 MB cap + BOM strip + WR-01 FIFO gate), utf8BOM, never-fail helper shape, parseGoMod line-loop precedent, pure-content parallel test-table pattern
provides:
  - readTOMLSection(content []byte, section string) map[string]string — unexported section-aware TOML-subset reader (ROBT-03)
  - Global skip-state machine (multi-line string / array / inline table) — SC4 no-partial-data contract
  - 341-line pure-content fixture matrix pinning the strict-degrade classification (STATE.md blocker discharged)
affects: [12-02 (detectPython), 12-03 (detectRust) — both consume readTOMLSection for [project]/[tool.poetry]/[package]]

# Actuals (#2632) — pairs with the plan's estimate (35000 tokens, 2 tasks, low confidence)
actuals:
  tokens: 4977       # chars/4 over the realized diff (toml.go 6194 + toml_test.go 13715)
  tasks: 2           # tasks completed
  commits: 2         # MEASURED: git rev-list --count e928a98..HEAD (#3968)
  plan_head_before: e928a98cce6ea35edc1797358e0d1e8a9568bea3
  plan_head_after: f7b07b05cc0fd69afb2905791c169f871992ae3f

# Tech tracking
tech-stack:
  added: [none — stdlib strings/bytes only; testify existing test dep]
  patterns: [line-based state machine with global skip states, strict-degrade classification, quote-aware value extraction, key-validated-first parsing]

key-files:
  created: [project_probe/toml.go, project_probe/toml_test.go]
  modified: []

key-decisions:
  - "Global skip states entered from keyval lines in ANY section (D-disc-4): parseKeyValue runs for every keyval line, not only the target section — the RESEARCH skeleton's section-gated order is read as section-scoped storage, not section-scoped skip detection (flagged assumption; pinned by TestReadTOMLSection_CrossSectionSkip)"
  - "Skip-state entry folded into the caller via enterSkip(value) regardless of key target: authors = [ in [package] enters the array skip state even though authors is never stored (P4)"
  - "BOM tolerance implemented with bytes.TrimPrefix(content, utf8BOM) reusing the manifest.go constant (single source of truth, Phase 10 precedent) — required by the BOM test row"
  - "Reader lands at 172 lines total (128 non-comment code) — flagged soft-budget deviation documented (OQ-3/A7); correctness first"

requirements-completed: [ROBT-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "readTOMLSection — section-aware TOML-subset reader with exact header matching, strict degrade-to-empty, and three global skip states (ROBT-03)"
    requirement: ROBT-03
    verification:
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_HappyPath"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_MissingSection"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_CrossSectionSkip"
        status: pass
    human_judgment: false
  - id: D2
    description: "Full strict-degrade edge matrix — sub-table isolation, header forms, comment/escape/remainder rules, CRLF/BOM, duplicates, 200-key section, adversarial no-panic rows (SC4 fixture-driven validation)"
    requirement: ROBT-03
    verification:
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_Adversarial"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_SubtableIsolation"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-09-29
status: complete
---

# Phase 12 Plan 1: TOML-Subset Reader Summary

**Unexported section-aware TOML-subset reader `readTOMLSection` with exact-header matching, strict degrade-to-empty, and three global skip states — the fixture-validated SC4 contract that both Python/Rust detectors (plans 12-02/12-03) read through**

## Performance

- **Duration:** 7 min
- **Started:** 2026-09-29T05:37:21Z
- **Completed:** 2026-09-29T05:44:00Z
- **Tasks:** 2 (1 tracer RED→GREEN, 1 expansion)
- **Files modified:** 2 created (toml.go, toml_test.go)

## Accomplishments

- `readTOMLSection(content []byte, section string) map[string]string` — line-based state machine over `readManifest` output: `[name]` headers with **exact** string equality (never prefix matching, P2), bare/`"quoted"` keys with key-validated-first parsing, `=` separator, `"`/`'` quoted scalars (raw interior, escapes verbatim), `#` comments, blank lines, CRLF + BOM tolerant (D-01/D-03).
- Strict degrade-to-empty: multi-line strings, dotted keys (`version.workspace = true` → absent, D-07), arrays, inline tables, unquoted values, unspecified values → the key is **never stored**, no panic, no partial data (D-02). Missing section → empty map, never an error (D-03).
- **Three global skip states** (multi-line string `"""`/`'''`, array `]`, inline table `}`) entered from keyval lines in **any** section — string/array bodies can never fabricate fake headers or keyvals (D-disc-4, SC4). Pinned by `TestReadTOMLSection_MultiLineStringSkip` + `TestReadTOMLSection_CrossSectionSkip`.
- 341-line pure-content fixture matrix (29 test functions, 39 `TestReadTOMLSection` cases): happy path, section isolation both directions, sub-table leakage (project.optional-dependencies + tool.poetry.dependencies), array-of-tables non-match, header forms (trailing comment / padded / quoted-segment and inner-dot documented non-matches), hash/equals inside values, raw escapes, garbage remainder, quoted keys, non-target keys, unspecified values, duplicate last-wins, CRLF, BOM, 200-key section, cross-section skip, adversarial no-panic rows (10k bracket flood, unterminated quote, 1 MB jumble — T-12-01). **Discharges the STATE.md blocker "TOML strict-degrade needs fixture-driven validation".**
- Anti-features verified absent from toml.go: no `os/exec`, `net/http`, `EvalSymlinks`, `WalkDir`, no file I/O, no `go-toml` dependency (stdlib `strings`+`bytes` only).

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — readTOMLSection core (RED→GREEN)** - `c7e57d1` (feat)
2. **Task 2: Expansion — full edge matrix** - `f7b07b0` (test)

**Plan metadata:** `docs(12-01): complete TOML-subset reader plan` (final metadata commit)

_Note: TDD per the Phase 10/11 adaptation — RED verified-but-uncommitted (the tree is only ever committed green), RED evidence recorded in the feat commit body; see TDD Gate Compliance._

## Files Created/Modified

- `project_probe/toml.go` (172 lines) - `readTOMLSection` + helpers `headerName`, `parseKeyValue`, `isBareKey`, `quotedValue`, `enterSkip`
- `project_probe/toml_test.go` (341 lines) - pure-content parallel matrix (29 tests, 39 cases)

## Decisions Made

- **Global skip states are entered from keyval lines in ANY section** (D-disc-4): the RESEARCH skeleton's section-gated parse order applies to storage only, not skip detection — otherwise `TestReadTOMLSection_CrossSectionSkip` would fabricate `name = "evil"` from a `[build-system]` multi-line string body. This is the flagged assumption (12-01-PLAN.md) executed per A7.
- **Skip-state entry is independent of key target**: `authors = [` enters the array skip state even though `authors` is never stored (P4 — `[package]` arrays must not disturb state).
- **BOM tolerance reuses `utf8BOM` from manifest.go** (`bytes.TrimPrefix`) — single source of truth per the Phase 10 decision; the reader stays tolerant even though readManifest strips upstream (P5 BOM row).
- **Reader size: 172 lines total (128 non-comment code)** — beyond the ~80-line CONTEXT target and the ~100-110 flagged estimate; the flagged assumption's verification explicitly requires only documenting the final count. Correctness first (OQ-3/A7); the skip states and thorough doc comments (repo convention, json.go/parseGoMod precedent) account for the size.

## Deviations from Plan

None - plan executed exactly as written. The reader-size overrun is a flagged assumption documented in the plan's prohibitions, not an execution deviation.

## Issues Encountered

None. The only environment wrinkle: `go test -cover` prints no percentage line on this Go version's terminal output — the coverage figure was extracted from `-coverprofile` via `go tool cover -func` (toml.go 97.3% file, project_probe 96.3% package).

## TDD Gate Compliance

- **RED evidence:** `go test ./project_probe/... -run 'TestReadTOMLSection'` → exit 1, build failed with `project_probe/toml_test.go:25:24: undefined: readTOMLSection` (recorded in the `feat(12)` commit body). The TARGET test could not compile before toml.go existed — intentional RED on the missing function.
- **GREEN:** `feat(12): implement TOML-subset reader` (c7e57d1) — 111 tests pass after implementation.
- **Matrix pin:** `test(12): pin TOML reader edge matrix` (f7b07b0) — 135 tests pass.
- **Gate sequence note:** canonical TDD orders RED-commit → GREEN-commit; this repo's Phase 10-approved adaptation (STATE.md: "RED verified-but-uncommitted per plan TDD adaptation (10-02 precedent)") commits the tree only when green, shipping the RED test file with the feat commit and recording RED evidence in its body. Both gate commits exist (`test(12)` and `feat(12)`); order is flipped by the sanctioned adaptation. REFACTOR: none needed — no cleanup pass was warranted after GREEN.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Reader is the single source of every detector field value: plans 12-02 (detectPython — `[project]` with `[tool.poetry]` fallback) and 12-03 (detectRust — `[package]`) consume `readTOMLSection` unchanged.
- The SC4 "no partial data" contract is fixture-pinned (state blocker closed): detectors can rely on empty-field degradation for dotted keys (`version.workspace`), arrays (`dynamic`), inline tables, and multi-line constructs.
- No blockers. `release/update.go` 68.9% coverage known-red remains untouched (Pitfall 7 — out of scope).

---
*Phase: 12-toml-subset-python-rust-detectors*
*Completed: 2026-09-29*

## Self-Check: PASSED

- FOUND: project_probe/toml.go (172 lines, readTOMLSection + 5 helpers)
- FOUND: project_probe/toml_test.go (341 lines, 29 tests / 39 cases)
- FOUND: 12-01-SUMMARY.md
- FOUND: commit c7e57d1 (feat(12): implement TOML-subset reader)
- FOUND: commit f7b07b0 (test(12): pin TOML reader edge matrix)