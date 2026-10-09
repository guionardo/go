---
phase: 14-semantics-hardening-release-polish
plan: 04
subsystem: docs
tags: [version-semantics, anti-feature-audit, lint-gate, coverage-gate, milestone-close]

# Dependency graph
requires:
  - phase: 14-semantics-hardening-release-polish
    provides: 14-01 fuzz targets + corpus, 14-02 correctness fold-ins (XML trim, exact-name guard, registry hygiene), 14-03 coverage closure (release/update.go tests)
provides:
  - "doc.go Version-semantics contract section (D-02) — the single downstream-readable place for DATA-03"
  - "README package index row + per-package section for project_probe (D-10)"
  - "14-AUDIT.md anti-feature audit evidence (D-03), deferred dispositions ledger (D-08), lint baseline + delta-zero verification, final gate outputs"
affects: [14-VERIFICATION.md, milestone v1.7 completion, future milestones]

actuals:
  tokens: 2659        # chars/4 over the realized diff (10634 chars added)
  tasks: 3
  commits: 3          # MEASURED: git rev-list --count ${PLAN_HEAD_BEFORE}..HEAD (#3968)
plan_head_before: 4402e03e1b0b63c9d738bd5effeea9c4eb8ab3b8
plan_head_after: e4aff8310878ff81315c8785e8bb720015625aab

tech-stack:
  added: []
  patterns:
    - "doc.go contract section as the single downstream-readable spec for version semantics"
    - "grep-verified anti-feature audit with verbatim commands recorded as evidence"
    - "delta-zero lint verification measured on the clean committed state (new-filter diff base)"

key-files:
  created:
    - .planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md
  modified:
    - project_probe/doc.go
    - README.md

key-decisions:
  - "Version semantics contract documented in doc.go with all six detector rules: go.mod toolchain floor, pyproject dynamic → '', Cargo version.workspace → '', .csproj Version→VersionPrefix (never MSBuild 1.0.0), pom.xml parent inheritance (never Super POM 4.0.0), package.json/composer.json verbatim; placeholders raw; XML TrimSpace named as decode hygiene, not normalization (D-02/DATA-03)"
  - "Anti-feature audit recorded as evidence file (14-AUDIT.md): five grep families + XML-entity family all zero matches; f.Stat() read-path nuance and TrimSpace decode-hygiene nuance stated (D-03/ROBT-05)"
  - "Deferred dispositions ledger recorded with rationale: Phase 10 CR-01 (Windows errno — needs Windows CI evidence, tracked) and WR-02 (panic logging — future milestone); residual INFO findings (11 IN-01/IN-02, 13 IN-02) accepted not-in-scope, recorded not dropped (D-08)"
  - "Lint delta-zero verified on the clean committed state: full run 0 issues, Phase-14 delta (8461196..HEAD) 0 issues; the dirty working tree's ~300 pre-existing issues documented as a new-filter diff-base artifact, advisory per A3 (D-10)"

patterns-established:
  - "Pattern: audit evidence files record verbatim commands + outputs so the verifier can re-run and confirm (T-14-14 mitigation)"
  - "Pattern: lint delta-zero measured against the pre-plan commit with --new-from-rev on a clean committed state"

requirements-completed: [DATA-03, ROBT-05]

coverage:
  - id: D1
    description: "doc.go Version-semantics contract section covering all seven detectors' raw-string rules (D-02/DATA-03)"
    requirement: DATA-03
    verification:
      - kind: other
        ref: "grep -c 'Version semantics' project_probe/doc.go → 1; go build ./...; go vet ./project_probe/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "README package index row (#package-project_probe, alphabetical) + ### Package project_probe section with Probe snippet (D-10)"
    verification:
      - kind: other
        ref: "grep -n 'projectprobe' README.md → row at :32 between pathtools/reflecttools + section at :312; go build ./... unaffected"
        status: pass
    human_judgment: false
  - id: D3
    description: "14-AUDIT.md anti-feature audit — five grep families + XML-entity family zero matches, two nuances stated (D-03/ROBT-05)"
    requirement: ROBT-05
    verification:
      - kind: other
        ref: "grep -c 'zero matches' 14-AUDIT.md → 10; all seven grep commands re-run in this plan print nothing"
        status: pass
    human_judgment: false
  - id: D4
    description: "Deferred dispositions ledger with rationale (D-08): CR-01, panic logging, residual INFO findings accepted not-in-scope"
    verification:
      - kind: other
        ref: "grep -c 'CR-01' 14-AUDIT.md → 1; grep -c 'accepted not-in-scope' → 4"
        status: pass
    human_judgment: false
  - id: D5
    description: "Final gates green: make coverage-quick, go vet, GOOS=windows vet, lint delta-zero on Phase-14 touched files"
    verification:
      - kind: other
        ref: "make coverage-quick (PASS 80.7%); go vet ./... (clean); GOOS=windows go vet ./project_probe/... (clean); GOTOOLCHAIN=go1.26.4 golangci-lint run on clean committed state → 0 issues"
        status: pass
    human_judgment: false

duration: 34min
completed: 2026-09-29
status: complete
---

# Phase 14 Plan 4: Documentation, Audit, and Milestone Gates Summary

**Version-semantics contract in doc.go (all 7 detectors' raw-string rules), README project_probe package index row + section, recorded anti-feature audit with deferred-disposition ledger, and the final green gates closing milestone v1.7**

## Performance

- **Duration:** 34 min
- **Started:** 2026-09-29T10:43:37Z
- **Completed:** 2026-09-29T11:17:44Z
- **Tasks:** 3
- **Files modified:** 3 (doc.go, README.md, 14-AUDIT.md)

## Accomplishments

- doc.go carries the `# Version semantics (DATA-03)` contract section — the single downstream-readable place for DATA-03: all six detector rules (go.mod toolchain floor, pyproject `dynamic` → `""`, Cargo `version.workspace` → `""`, .csproj Version→VersionPrefix never the MSBuild 1.0.0, pom.xml parent inheritance never the Super POM 4.0.0, package.json/composer.json verbatim), placeholders reported raw, and the decode-hygiene note naming XML whitespace trimming as NOT normalization (Pitfall 2)
- README.md package index gains the projectprobe row between pathtools and reflecttools (locked anchor `#package-project_probe`) plus a `### Package project_probe` section with import line, never-fail contract + 7-detector cascade description, and a Probe usage snippet (D-10)
- 14-AUDIT.md records the D-03 anti-feature audit (five grep families + XML-entity family, all zero matches, commands verbatim, with the f.Stat() read-path and TrimSpace decode-hygiene nuances), the D-08 deferred dispositions ledger (CR-01 Windows errno, panic logging, residual INFO findings), the lint baseline (28 advisory, A3) with delta-zero verification, and the final green gate outputs
- Final gates all green: make coverage-quick (80.7%), go vet ./..., GOOS=windows go vet ./project_probe/..., go build, 817 tests in 25 packages, go.mod/go.sum unchanged — milestone v1.7 complete

## Task Commits

Each task was committed atomically:

1. **Task 1: doc.go Version-semantics contract section** - `f23fe27` (docs)
2. **Task 2: README package index row + section** - `5a5aaa9` (docs)
3. **Task 3: 14-AUDIT.md audit + gates** - `e4aff83` (docs)

**Plan metadata:** pending docs commit

## Files Created/Modified

- `project_probe/doc.go` - Added `# Version semantics (DATA-03)` contract section (23 lines) after the detector-list paragraph: raw-string rules for all 7 detectors, placeholder boundary, decode-hygiene note; never-fail contract intact
- `README.md` - Added projectprobe package index row (line 32, alphabetical) and `### Package project_probe` section with import line, never-fail + cascade description, Probe snippet
- `.planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md` - New audit evidence file: anti-feature audit (7 grep commands, zero matches), two nuances, D-08 deferred ledger, lint baseline + delta-zero verification, final gate outputs

## Decisions Made

- Version semantics documented as the raw-string contract in doc.go (D-02): every Version assignment is a verbatim manifest value or `""`; toolchain-floor semantics for go.mod; structural degrades (dynamic/version.workspace) → empty; XML TrimSpace explicitly named decode hygiene, not normalization
- Anti-feature audit recorded as a durable evidence file rather than a prose claim (D-03) — the verifier re-runs the recorded greps
- Lint delta-zero (D-10/A3) verified on the clean committed state: full run 0 issues, Phase-14 delta 0 issues; the dirty working tree's ~300 pre-existing issues documented as a golangci new-filter diff-base artifact (advisory 28-baseline grown by branch size) — no Phase-14-introduced issues exist

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 'decode hygiene' phrase split across doc comment lines failed the plan's own grep gate**
- **Found during:** Task 1 (doc.go Version-semantics contract)
- **Issue:** The contract wording recommended in 14-RESEARCH Pattern 7 broke the phrase "decode hygiene" across two comment lines ("(decode\n// hygiene —"), so the plan's verify `grep -c 'decode hygiene' project_probe/doc.go` printed 0 (required ≥1)
- **Fix:** Reworded the closing boundary statement to keep the phrase on one line: "whitespace-trimmed at decode time; that trim is decode hygiene — the value itself is still never normalized" — same semantics, gate-compliant placement
- **Files modified:** project_probe/doc.go
- **Verification:** grep -c 'decode hygiene' → 1; all other Task 1 greps pass; go build + go vet clean
- **Committed in:** f23fe27 (Task 1 commit)

**2. [Rule 3 - Blocking] Lint delta-zero gate needed clean-committed-state measurement (dirty-tree artifact)**
- **Found during:** Task 3 (14-AUDIT.md lint baseline section)
- **Issue:** The plan's literal gate (`GOTOOLCHAIN=go1.26.4 golangci-lint run | grep touched-files`) prints matches in the dirty main working tree (~300 issues). Investigation showed golangci's `new`-filter mis-resolves its diff base in a dirty tree (reports the entire v1.7 branch as "new" — the advisory 28-baseline grown with branch size, phases 10-13 debt). The `--new-from-rev` CLI flag is ineffective in the dirty tree (config `new-from-rev: HEAD` + `new-from-merge-base: main` both set)
- **Fix:** Measured delta-zero on the clean committed state (fresh clone of HEAD with local main ref at merge-base + swapper binaries built): full `golangci-lint run` → 0 issues; `--new-from-rev=8461196` (Phase-14 delta) → 0 issues. Documented the measurement methodology + verifier instructions in 14-AUDIT.md Section 3 so the verifier reproduces delta-zero correctly
- **Files modified:** .planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md (documentation)
- **Verification:** clean clone at HEAD: plain run 0 issues, delta run 0 issues; all other gates green
- **Committed in:** e4aff83 (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 3 blocking issues; no Rule 1/2/4)
**Impact on plan:** Both fixes were documentation/measurement adjustments — no production code changed beyond the planned doc.go/README edits; no scope creep; milestone gates verified green.

## Issues Encountered

- golangci-lint 2.12.2 `new`-filter behaves inconsistently between dirty and clean working trees (diff-base resolution). Resolved by measuring delta-zero on the clean committed state — the state CI and the verifier consume — and documenting the artifact explicitly in 14-AUDIT.md (T-14-14 mitigation: evidence is re-runnable).
- LSP diagnostics flagged js/wasm-target errors on unrelated pre-existing files (path_tools, manifest.go, mid, swapper) — platform-constraint noise, out of scope, not introduced by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Milestone v1.7 complete: all four Phase 14 success criteria hold (raw-version semantics proven + documented, anti-features proven absent, fuzz seeds running under go test, package shipping green — coverage, vet, lint delta-zero)
- All 21 v1.7 requirements mapped and satisfied; DATA-03 and ROBT-05 documented with grep-verified evidence in 14-AUDIT.md
- Phase 14 is the final v1.7 phase — ready for milestone close (/gsd-complete-milestone) and the phase verifier (/gsd-verify-work), which consumes 14-AUDIT.md + 14-VERIFICATION.md
- Deferred to future milestones: Phase 10 CR-01 (Windows errno mapping — needs Windows CI), Phase 10 WR-02 (panic logging), v2 backlog (FRAM-01/02, REFN-01..04)

---
*Phase: 14-semantics-hardening-release-polish*
*Completed: 2026-09-29*