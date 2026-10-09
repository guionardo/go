---
phase: 12-toml-subset-python-rust-detectors
plan: 04
subsystem: testing
tags: [toml, skip-state, quote-aware, sc4, gap-closure, tdd]

# Dependency graph
requires:
  - phase: 12-toml-subset-python-rust-detectors
    provides: readTOMLSection with global skip states (12-01), detectPython (12-02), detectRust (12-03)
provides:
  - "Quote-aware skip-state scanning in readTOMLSection: closesMultiLine/clearsBracket/opensMultiLine helpers + cross-line pendingMLS state for multi-line strings opened inside bracket bodies"
  - "5 new matrix tests (13 subtests) pinning no-fabrication for all 8 CR-01 shapes plus the multi-line-string-opener-in-bracket-body family"
affects: [phase 13 xml-detectors, phase 14 semantics-hardening-fuzz, verify-work]

# Actuals (#2632) — pairs with the plan's `estimate` (26000 tokens).
actuals:
  tokens: 2745    # chars/4 over the realized diff (10979 chars / 4)
  tasks: 2
  commits: 1      # MEASURED: git rev-list --count 30b9afb..HEAD (fix commit; docs commit follows)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Quote-aware skip-state scanning: run-of-3 vs run-of->3 multi-line string close semantics, in-string bracket immunity honoring \\ escapes, cross-line pending multi-line-string state inside bracket bodies"

key-files:
  created: []
  modified:
    - project_probe/toml.go
    - project_probe/toml_test.go

key-decisions:
  - "Skip-state clearing is quote-aware: closesMultiLine treats a run of exactly 3 quotes as a close and a run > 3 as close+reopen (state persists); clearsBracket ignores ]/} inside \" or ' quoted strings (honoring \\ escapes) and tracks [/{ depth so nested constructs never clear the outer state"
  - "The skipClose loop carries a cross-line pendingMLS state (opensMultiLine seeds it, closesMultiLine net-closed clears it) so a \"\"\" or ''' opener inside a bracket body keeps every later ]/} inside the string — eliminating the premature-clear fabrication direction"
  - "enterSkip uses the same quote-aware scans: 4-quote opening lines and bracket-in-string opening lines now ENTER the skip state (previously missed entry was CR-01's second fabrication path)"
  - "Under-skip direction (extra line skipped when a construct closer shares a line with the string's close) is accepted: degrades to empty, the SAFE direction per the plan's flagged-assumptions table"

patterns-established:
  - "Red-Green-Refactor with Phase 10/11 TDD adaptation: RED verified-but-uncommitted (pre-commit go-test hook keeps CI green), RED evidence quoted in the fix commit body"
  - "Regression isolation in RED: new failing rows run with -run union of old+new skip tests to prove old matrix stays green"

requirements-completed: [ROBT-03, DETC-05, DETC-06]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Quote-aware skip-state scanning in readTOMLSection — CR-01 shapes (6-quote body lines for \"\"\" and ''', 4-quote opening lines, ]/} inside quoted strings in multi-line arrays/inline tables, fake header after delimiter-run body line) degrade strictly to empty fields, never fabricating name/version/description"
    requirement: ROBT-03
    verification:
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_SixQuoteBody"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_FourQuoteOpening"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_BracketInQuotedString"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_FakeHeaderAfterDelimiterRun"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_MultiLineStringInBracketBody"
        status: pass
    human_judgment: false
  - id: D2
    description: "Global skip states suppress header AND keyval parsing inside construct bodies in ANY section, including delimiter-run bodies — cross-section row pins the fake [project] header inside a [build-system] multi-line string never switches the section"
    requirement: ROBT-03
    verification:
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_FakeHeaderAfterDelimiterRun"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_CrossSectionSkip"
        status: pass
    human_judgment: false
  - id: D3
    description: "Malformed pyproject.toml/Cargo.toml degrade to empty/fallback fields — the reader is the single source of every detector field value, so the fixed reader restores 12-02 truth 5 (folder-base Name) and 12-03 (Rust) no-fabrication end-to-end"
    requirement: DETC-05
    verification:
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_SixQuoteBody"
        status: pass
      - kind: unit
        ref: "project_probe/toml_test.go#TestReadTOMLSection_QuoteAware_BracketInQuotedString"
        status: pass
    human_judgment: false

# Metrics
duration: 37min
completed: 2026-09-29
status: complete
---

# Phase 12 Plan 04: Quote-Aware Skip-State Scanning (CR-01 Gap Closure) Summary

**Quote-aware skip-state clearing in readTOMLSection (closesMultiLine/clearsBracket/opensMultiLine + cross-line pendingMLS), with a 13-subtest matrix pinning no-fabrication for every CR-01 shape — restores SC4, 12-01 truth 4, and 12-02 truth 5.**

## Performance

- **Duration:** 37 min
- **Started:** 2026-09-29T06:20:00Z
- **Completed:** 2026-09-29T06:57:43Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Replaced whole-line `strings.Contains(line, skipDelim)` (toml.go:26) and `strings.ContainsRune(line, skipClose)` (toml.go:31) with quote-aware scanning — the CR-01 blocker that fabricated `name="evil"` from inside multi-line construct bodies in 8/9 verifier probe rows.
- Implemented `closesMultiLine` (run of exactly 3 closes, run > 3 closes AND reopens → state persists, net across the line), `clearsBracket` (brackets inside `"`/`'`-quoted strings honoring `\` escapes never affect depth; `[`/`{` depth tracking so nested constructs never clear the outer state), and `opensMultiLine` + cross-line `pendingMLS` in the skipClose loop (a `"""`/`'''` opener inside a bracket body keeps every later `]`/`}` inside the string until a net-closed line).
- `enterSkip` now uses the same scans: a 4-quote opening line (`"""a""""` = close+reopen) ENTERS the multi-line-string skip state, and an opening line whose first `]`/`}` sits inside a quoted string (`["A ] B",`) ENTERS the bracket state — both were missed-entry fabrication paths.
- Added 5 matrix tests (13 subtests) pinning no-fabrication: SixQuoteBody (both `"""` and the previously-unexercised `'''` branch — IN-03), FourQuoteOpening, BracketInQuotedString (array_body, inline_table_body, array_opening_line), FakeHeaderAfterDelimiterRun (cross-section), MultiLineStringInBracketBody (body_line_opener, body_line_opener_literal, opening_line_opener).
- Full suite green: 185 project_probe tests (172 pre-existing + 13 new); `go build ./...` clean; `GOOS=windows go vet ./project_probe/...` clean; anti-feature greps print nothing.

## Task Commits

Each task was committed atomically (TDD RED is verified-but-uncommitted per the Phase 10/11 adaptation — the pre-commit hook runs `go test ./...` and a failing tree is never committed):

1. **Task 1: RED — pin the CR-01 skip-state fabrication shapes as failing matrix rows** - no commit (RED verified-but-uncommitted; RED evidence captured to `/var/folders/zt/1bwtr_wj1v16t_nwfkqk5q940000gn/T/opencode/toml-gap-red.txt`)
2. **Task 2: GREEN — quote-aware skip-state scanning in toml.go** - `596d58b` (fix(12), RED evidence in commit body)

**Plan metadata:** `docs(12-04): complete gap closure plan (quote-aware skip-state scanning)` (final metadata commit)

## Files Created/Modified
- `project_probe/toml.go` - quote-aware skip-state scanning: `closesMultiLine`, `clearsBracket`, `opensMultiLine` helpers; skipDelim case uses net-closed test; skipClose case carries cross-line `pendingMLS`; enterSkip uses the same scans; doc comment notes delimiter runs never clear skip states
- `project_probe/toml_test.go` - 5 new `TestReadTOMLSection_QuoteAware_*` tests (13 subtests), pure []byte + t.Parallel per file convention

## Decisions Made
- Skip-state clearing uses run-length semantics for multi-line strings (3 = close, >3 = close+reopen) and in-string immunity for brackets — the CR-01 fix direction from 12-REVIEW.md lines 67-90, implemented verbatim.
- Cross-line multi-line-string state (pendingMLS) is required inside bracket bodies — a per-line quote toggle alone would mask only the opener's own line; the line AFTER the opener would still clear prematurely (the plan's flagged-assumption resolution).
- Under-skip residual accepted: a line that nets the string closed skips bracket scanning, so a construct closer sharing that line is scanned on the next line — extra skipping degrades to empty, never fabricates (documented in the plan's flagged-assumptions table).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- None. The `make coverage-quick` run shows the pre-existing `release/update.go` 68.9% file-threshold failure (known-red, documented in 12-01-PLAN.md line 223) — untouched, exits 2 is not a failure of this plan. All project_probe rows (93.1% package, 87.7% toml.go file) and the total (79.3%) pass.

## TDD Gate Compliance

- **RED gate:** Satisfied per the Phase 10/11 TDD adaptation (12-01-PLAN.md lines 89-111): RED verified-but-uncommitted. The 5 QuoteAware tests failed against the pre-fix toml.go with fabricated `name="evil"` assertion diffs (10 diffs across 8 subtests), captured to `toml-gap-red.txt` and quoted in the fix commit body. No `test(...)` commit exists by design — a failing tree is never committed (pre-commit go-test hook).
- **GREEN gate:** Satisfied — `fix(12)` commit `596d58b` (the plan specifies `fix` scope, not `feat`, for this gap closure); all 185 project_probe tests pass after the fix.
- **REFACTOR gate:** Not applicable — no refactor commit was needed (the fix was minimal and fully pinned by the matrix).

## Next Phase Readiness
- SC4 restored for the reader: all 8 CR-01 probe shapes now degrade strictly to empty — the verifier's re-run of the 9-row probe should show 9/9 no-fabrication.
- 12-01 truth 4 restored (global skip states hold in ANY section, even delimiter-run bodies); 12-02 truth 5 restored (malformed pyproject degrades to folder-base Name).
- Ready for phase verification re-run; phase 13 (XML detectors) can proceed with the reader contract intact.

## Self-Check: PASSED
- SUMMARY.md exists: `.planning/phases/12-toml-subset-python-rust-detectors/12-04-SUMMARY.md` ✓
- Commit `596d58b` exists in git log ✓
- RED evidence file contains 10 `name":"evil"` diffs ✓
- `go test ./project_probe/... -run 'TestReadTOMLSection_QuoteAware'` → 13 passed ✓
- `go test ./project_probe/...` → 185 passed ✓

---
*Phase: 12-toml-subset-python-rust-detectors*
*Completed: 2026-09-29*