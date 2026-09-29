---
phase: 10-package-foundation-api-contract-repo-cleanup
plan: 02
subsystem: api-contract
tags: [project-probe, probe, detector-registry, sentinel-errors, ignore-list, tdd, fnd-02, fnd-03]

# Dependency graph
requires:
  - phase: 10-package-foundation-api-contract-repo-cleanup
    provides: "10-01: project_detector/ deleted — go build ./... green module-wide"
provides:
  - "projectprobe public API contract: Probe(folder) (ProjectData, error) with the never-fail contract (FND-02/FND-03)"
  - "ProjectData{Folder, Language, Name, Version, Description} + Language type with 8 explicit-value constants (DATA-01, D-05..D-08)"
  - "Three sentinel errors ErrFolderNotFound/ErrNotDirectory/ErrPermissionDenied with syscall errno mapping incl. EPERM (D-01/D-04)"
  - "Ordered private detector registry with (ProjectData, bool) contract and panic-recovery dispatch (DETC-01, D-03)"
  - "13-entry exact-case ignore list + hasContent content gate (ROBT-01, D-10..D-12)"
affects: [10-03, phases 11-13 (detector bodies extend registry.go), CI, consumers of projectprobe]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens) to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 4317
  tasks: 3
  commits: 2

# Commit ledger (#3968) — measured, never narrated
plan_head_before: 0627291b12390af62e46841925db482d7c05f160
plan_head_after: e1eac0c2a1a23471b1f98fca4b6992540e79257b

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Never-fail API contract: errors reserved for hard folder-level I/O; content outcomes return LanguageUnknown + nil error (FND-02/D-03)"
    - "syscall errno mapping with errors.Is (ENOENT/ENOTDIR/EACCES+EPERM) instead of fs.ErrNotExist (D-04)"
    - "Named-return + defer recover() panic-recovery dispatch — a panicking detector is a non-match, never an error (cache/singleflight callSetter precedent)"
    - "Clean-once folder path: filepath.Clean(as-given) reused for stat/readdir/errors/Folder (D-09)"

key-files:
  created:
    - "project_probe/project.go — ProjectData struct + Language type + 8 explicit-value constants (D-05..D-08)"
    - "project_probe/errors.go — 3 sentinel errors + mapFolderError errno mapping (D-01/D-02/D-04)"
    - "project_probe/probe.go — Probe entry point: empty-guard, clean-once, stat, readdir, content gate, dispatch (FND-03/D-09/OQ-1)"
    - "project_probe/registry.go — detectorFunc type, empty ordered detectors slice, runDetectors + callDetector panic-recovery (DETC-01/D-03)"
    - "project_probe/ignore.go — 13-entry exact-case ignoreDirs map, isIgnoredDir, hasContent (D-10..D-12)"
    - "project_probe/doc.go — never-fail contract package doc with Usage + Sentinel errors sections (FND-02)"
    - "project_probe/probe_test.go — TestProbe (6 rows), TestProbe_Deterministic, TestIgnoreList"
    - "project_probe/errors_test.go — TestMapFolderError (synthetic errnos incl. EPERM), TestProbePermissionDenied (OS-gated)"
    - "project_probe/registry_test.go — empty/order/first-match/panic-recovery/merge-rule tests (no t.Parallel)"
    - "project_probe/example_test.go — ExampleProbe with deterministic Output"
  modified: []

key-decisions:
  - "RED evidence adapted to the repo's pre-commit hooks: tests written first and verified failing (compile-fail on the planned-but-absent API), then committed together with the implementation in the feat commit — RED commits cannot land because the go-test hook runs go test ./... and CI must stay green (plan 10-02 TDD adaptation note)"
  - "RED failure shape: expected 'no non-test Go files in' per plan; actual was 'undefined: ...' compile errors — an internal test package compiles its test files into the test binary, so the RED is compile-fail on the missing contract symbols; same RED intent (target tests cannot pass), documented in the feat commit message"
  - "GOTOOLCHAIN=go1.26.4 required for golangci-lint: the installed golangci-lint v2.12.2 is built with go1.26.3 and panics ('file requires newer Go version go1.27') under the go1.27.0 system toolchain — pre-existing environment issue affecting every package; commit environment pins the toolchain for the hook"
  - "make coverage-quick exit 2 is pre-existing and unrelated: release/update.go at 68.9% vs 70% file threshold (no change from this plan); project_probe itself is at 100% package/file coverage; logged to deferred-items.md"

patterns-established:
  - "Pattern: internal test package (package projectprobe) for unexported-surface tests (mapFolderError/hasContent/runDetectors/detectors) — config precedent"
  - "Pattern: registry tests save/restore the package-level detectors slice and never call t.Parallel() (AGENTS.md global-state rule)"

requirements-completed: [FND-02, FND-03, DETC-01, DETC-09, DATA-01, ROBT-01, ROBT-04]

coverage:
  - id: D1
    description: "Probe never-fail contract — sentinel errors (ErrFolderNotFound/ErrNotDirectory/ErrPermissionDenied, wrapped with the cleaned folder path) for hard folder-level I/O failures; LanguageUnknown + nil error for empty, ignored-only, and unrecognized-content folders; Probe(\"\") returns ErrFolderNotFound"
    requirement: FND-03
    verification:
      - kind: unit
        ref: "project_probe/probe_test.go#TestProbe"
        status: pass
      - kind: unit
        ref: "project_probe/errors_test.go#TestMapFolderError"
        status: pass
      - kind: unit
        ref: "project_probe/errors_test.go#TestProbePermissionDenied"
        status: pass
    human_judgment: false
  - id: D2
    description: "Ordered detector registry — (ProjectData, bool) never-fail contract, first match wins, panic-recovery dispatch treats a panicking detector as a non-match, empty registry default; detector merge rule overrides Unknown defaults while Folder stays probe-owned"
    requirement: DETC-01
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestRunDetectors_EmptyRegistry"
        status: pass
      - kind: unit
        ref: "project_probe/registry_test.go#TestRunDetectors_OrderAndFirstMatch"
        status: pass
      - kind: unit
        ref: "project_probe/registry_test.go#TestRunDetectors_PanicRecovery"
        status: pass
      - kind: unit
        ref: "project_probe/registry_test.go#TestProbe_MergeRule"
        status: pass
    human_judgment: false
  - id: D3
    description: "ProjectData model + Language constants with exact display values (incl. LanguageUnknown == \"unknown\", never the zero value); Folder = filepath.Clean(as-given), relative stays relative, identical repeated probes"
    requirement: DATA-01
    verification:
      - kind: unit
        ref: "project_probe/probe_test.go#TestProbe_Deterministic"
        status: pass
      - kind: unit
        ref: "project_probe/example_test.go#ExampleProbe"
        status: pass
    human_judgment: false
  - id: D4
    description: "Ignore-list content gate — all 13 D-10 names match exactly (case-sensitive), case variant Node_Modules does not; folder with only ignored dirs yields LanguageUnknown + nil error"
    requirement: ROBT-01
    verification:
      - kind: unit
        ref: "project_probe/probe_test.go#TestIgnoreList"
        status: pass
      - kind: unit
        ref: "project_probe/probe_test.go#TestProbe"
        status: pass
    human_judgment: false
  - id: D5
    description: "Never-fail contract documented in doc.go with Usage and Sentinel errors sections (FND-02); Windows-safe stdlib-only implementation (filepath not path, no os/exec/net/http/EvalSymlinks/WalkDir)"
    requirement: FND-02
    verification:
      - kind: other
        ref: "GOOS=windows go build + go vet ./project_probe/... (exit 0)"
        status: pass
      - kind: other
        ref: "grep for bare \"path\" import and ROBT-05 anti-features (no matches)"
        status: pass
      - kind: other
        ref: "head -1 project_probe/doc.go == '// Package projectprobe provides project detection for local folders.'"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-09-29
status: complete
---

# Phase 10 Plan 2: projectprobe API Contract Summary

**The `projectprobe` package's never-fail API contract shipped end-to-end under TDD: `Probe(folder) (ProjectData, error)` wired through clean-once path handling, syscall errno → sentinel mapping (incl. EPERM), the ordered panic-safe detector registry, and the 13-entry ignore-list content gate — with the empty registry delivering `LanguageUnknown` + nil error for every content-bearing folder, ready for detector bodies in phases 11-13.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-29T02:01:34Z
- **Completed:** 2026-09-29T02:16:11Z
- **Tasks:** 3 (RED → GREEN → REFACTOR)
- **Files modified:** 10 created (5 production + 4 test + 1 example)

## Accomplishments

- `Probe(folder) (ProjectData, error)` with the never-fail contract: sentinel errors (ErrFolderNotFound / ErrNotDirectory / ErrPermissionDenied, always `fmt.Errorf("probe %s: %w", folder, ...)` wrapped) reserved for hard folder-level I/O; every content outcome — empty, ignored-only, unrecognized — returns `LanguageUnknown` with nil error (D-01/D-02/D-03/DETC-09)
- `mapFolderError` errno mapping via `errors.Is` against `syscall.ENOENT/ENOTDIR/EACCES` + `EPERM` (D-04 as amended 2026-09-28) — never `fs.ErrNotExist` which conflates ENOENT+ENOTDIR
- Clean-once `filepath.Clean(as-given)` (D-09): one canonical path reused for stat, readdir, error wrapping, and the `Folder` field; relative input stays relative; `Probe("")` → `ErrFolderNotFound` (OQ-1 guard, never probes the cwd)
- Ordered detector registry: `detectorFunc(folder) (ProjectData, bool)` — never-fail by construction, first match wins, empty in Phase 10 (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP order contract commented in-file), `callDetector` panic-recovery dispatch mirroring cache `callSetter` (a panicking detector is a non-match, never an error, never a re-panic)
- Ignore-list gate: 13 exact-case names (node_modules, vendor, .git, dist, .idea, .venv, venv, __pycache__, target, build, bin, obj, .cache), `hasContent` true on first file or non-ignored dir (D-10..D-12)
- `ProjectData{Folder, Language, Name, Version, Description}` + `Language string` with 8 explicit-value constants (`LanguageUnknown = "unknown"`, never the zero value)
- doc.go documents the never-fail contract with Usage + Sentinel errors sections (FND-02); ExampleProbe prints only deterministic fields
- 100% statement coverage on project_probe (package ≥80% and file ≥70% satisfied without an override; total 77.6% ≥ 75% held)
- Windows-safe: `GOOS=windows go build` + `go vet` clean; `filepath` only, no bare `path` import, no os/exec/net/http/EvalSymlinks/WalkDir

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — write the failing Probe contract tests (end-to-end tracer)** — *no commit* (plan adaptation: RED commits cannot land — the pre-commit go-test hook runs `go test ./...` and CI must stay green; tests were written first, verified failing, and ship with the implementation in Task 2's commit)
2. **Task 2: GREEN — implement model, sentinels, Probe, registry, ignore gate** - `57cd875` (feat(10): implement projectprobe probe contract — message body records the RED evidence and the FND-01 untracked reality)
3. **Task 3: REFACTOR — document the never-fail contract and add the deterministic example** - `e1eac0c` (docs(10): document never-fail probe contract)

**Plan metadata:** `e1eac0c` + state files (docs commit at plan close)

_Note: TDD commits use phase-level scope `feat(10)`/`docs(10)` per the plan's explicit commit-message contract (plan 10-02 Task 2/3 `<action>`), not `feat(10-02)` — see TDD Gate Compliance below._

## Files Created/Modified

- `project_probe/project.go` - ProjectData struct + Language type + 8 explicit-value constants (D-05..D-08)
- `project_probe/errors.go` - 3 sentinel errors + mapFolderError (D-01/D-02/D-04)
- `project_probe/probe.go` - Probe entry point with empty-string guard, clean-once, stat/readdir mapping, content gate, registry dispatch
- `project_probe/registry.go` - detectorFunc, empty ordered detectors slice, runDetectors, callDetector (panic → non-match)
- `project_probe/ignore.go` - 13-entry exact-case ignoreDirs, isIgnoredDir, hasContent
- `project_probe/doc.go` - never-fail contract package doc (FND-02)
- `project_probe/probe_test.go` - TestProbe (6 rows), TestProbe_Deterministic, TestIgnoreList (t.Parallel-safe)
- `project_probe/errors_test.go` - TestMapFolderError (synthetic errnos + EPERM row), TestProbePermissionDenied (OS-gated skip)
- `project_probe/registry_test.go` - empty registry, order + first-match, panic recovery, merge rule (NO t.Parallel, save/restore detectors)
- `project_probe/example_test.go` - ExampleProbe with deterministic `// Output: language=unknown err=<nil>`

## Decisions Made

- **TDD RED adaptation (plan-sanctioned):** tests written first in Task 1, verified failing before any implementation, then committed together with the implementation — the repo's pre-commit hooks (go test ./...) make a RED-only commit impossible without breaking CI. RED evidence recorded in the feat commit message.
- **RED failure shape documented:** plan predicted "no non-test Go files in"; actual is "undefined: ..." compile errors because an internal test package's files compile into the test binary. Same RED intent — the target tests fail on the planned-but-absent API (strongest RED form: the contract symbols don't exist).
- **GOTOOLCHAIN=go1.26.4 for the lint hook:** pre-existing environment issue — golangci-lint v2.12.2 (built with go1.26.3) panics under the go1.27.0 system toolchain on any package in this repo; the commit environment pins the toolchain so the pre-commit golangci-lint hook runs. Noted in SUMMARY; the user may want to `make install-golangci` (installs latest).
- **make coverage-quick exit 2 is pre-existing:** `release/update.go` at 68.9% vs 70% file threshold — unrelated to this plan (project_probe at 100%); logged to deferred-items.md.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan-accuracy] RED failure message differs from plan prediction**
- **Found during:** Task 1 (RED verification)
- **Issue:** Plan's acceptance criteria expected `go test ./project_probe/...` to fail with "no non-test Go files in"; the actual failure is "undefined: ErrFolderNotFound / Probe / runDetectors ..." compile errors — internal test packages compile their test files into the test binary, so the package has "non-test" content to build and the missing-symbols error surfaces instead.
- **Fix:** None required — the RED state is achieved either way (exit 1, target tests cannot pass without the planned API). Documented the actual failure shape in the feat commit message and this SUMMARY.
- **Files modified:** none
- **Verification:** `go test ./project_probe/...` exits 1 pre-implementation (RED), then 0 post-implementation (GREEN)
- **Committed in:** 57cd875 (RED evidence in message body)

**2. [Rule 1 - Environment] golangci-lint v2.12.2 panics under the go1.27.0 system toolchain**
- **Found during:** Task 2 (GREEN pre-commit lint verification)
- **Issue:** `golangci-lint run ./project_probe/...` (and ANY package, e.g. br_docs) panics with "file requires newer Go version go1.27 (application built with go1.26)" — the installed binary is built with go1.26.3 and cannot type-check under the go1.27 toolchain driver. This would fail the pre-commit golangci-lint hook on every Go commit.
- **Fix:** Ran the lint and the commit with `GOTOOLCHAIN=go1.26.4`, which makes the go/packages driver use the go1.26.4 toolchain — golangci-lint then runs cleanly (0 issues on project_probe). No repo files changed.
- **Files modified:** none (environment workaround only)
- **Verification:** `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → 0 issues; commit landed with hooks green
- **Committed in:** 57cd875 (hook run within the commit)

**3. [Rule 2 - Missing critical] 63 lint issues fixed in the GREEN slice before the hook would pass**
- **Found during:** Task 2 (GREEN verification)
- **Issue:** The first golangci-lint run surfaced 63 issues across the 8 new files: goconst (7, e.g. repeated "stat" Op and ignore-name literals), gosec (8, file/dir permission modes 0o644/0o755), testifylint (2, assert.NotErrorIs → require.NotErrorIs), wsl_v5 (22, whitespace cuddling), and gofmt/goimports/golines formatting (24, incl. a 121-char table row).
- **Fix:** Deduplicated test literals (const opStat, ignoreNames fixture), lowered modes to 0o600/0o700, switched to require.NotErrorIs, restructured the D-10 map to a shared test fixture, applied wsl blank-line conventions, and ran `golangci-lint fmt` for the formatter set.
- **Files modified:** project_probe/*.go (all 8 then-current files)
- **Verification:** `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → 0 issues; all 20 tests still pass
- **Committed in:** 57cd875

**4. [Rule 3 - Blocking] errcheck on deferred os.RemoveAll in the example**
- **Found during:** Task 3 (REFACTOR verification)
- **Issue:** `defer os.RemoveAll(dir)` in ExampleProbe failed errcheck.
- **Fix:** `defer func() { _ = os.RemoveAll(dir) }()`.
- **Files modified:** project_probe/example_test.go
- **Verification:** golangci-lint run → 0 issues; 21 tests pass
- **Committed in:** e1eac0c

---

**Total deviations:** 4 auto-fixed (2 plan-accuracy/environment documentation, 1 missing-critical lint compliance, 1 blocking lint)
**Impact on plan:** All fixes were necessary for the commit gates (pre-commit hooks) to pass; no scope creep, no behavior changes to the D-01..D-12 contract.

## TDD Gate Compliance

Plan type: `tdd` — RED → GREEN → REFACTOR sequence executed with the plan's sanctioned adaptation:

| Gate | Expected commit | Actual | Status |
|------|-----------------|--------|--------|
| RED | `test(10-02): add failing test...` | no commit — RED verified-but-uncommitted (pre-commit go-test hook runs `go test ./...`; a failing tree cannot be committed; CI must stay green) | ADAPTED (plan 10-02 adaptation note; RED evidence recorded in the GREEN commit message) |
| GREEN | `feat(10-02): ...` | `57cd875 feat(10): implement projectprobe probe contract` | PASS — commit scope `(10)` per the plan's explicit commit-message contract, not `(10-02)` |
| REFACTOR | `refactor(10-02): ...` | `e1eac0c docs(10): document never-fail probe contract` | PASS — cleanup (docs + example) with tests still green |

- RED evidence: `go test ./project_probe/...` exited 1 pre-implementation with the target tests failing against the planned-but-absent API (compile-fail on undefined contract symbols) — verified before any production file existed.
- GREEN: implementation made all 20 Task-1 tests pass (verified exit 0), then coverage + Windows build/vet + lint all green.
- REFACTOR: doc.go + ExampleProbe added; all 21 tests pass, lint 0 issues.
- Commit-scope note: the plan's `<action>` blocks mandate `feat(10):` / `docs(10):` messages (matching the phase-10 commit history, e.g. `docs(10): create phase plan`); the gate-validation regex `^feat(10-02):` in tdd.md would not match these — flagging is expected and by design.

## Issues Encountered

- **Pre-existing coverage gate failure (out of scope):** `make coverage-quick` exits 2 on `release/update.go` (68.9% vs 70% file threshold) — unrelated to this plan, reproducible before any project_probe change (project_probe itself is at 100% package/file coverage and the total is 77.6% ≥ 75%). Per the scope boundary rule this was not fixed; logged to `deferred-items.md` in the phase directory.
- **golangci-lint toolchain panic (pre-existing environment):** see Deviation 2 — worked around with `GOTOOLCHAIN=go1.26.4`; a permanent fix is `make install-golangci` (installs latest).

## User Setup Required

None - no external service configuration required. (Optional: `make install-golangci` to upgrade the local golangci-lint past the go1.27 toolchain incompatibility.)

## Next Phase Readiness

- The full probe pipeline ships and is tested: clean-once → stat → readdir → content gate → registry dispatch. Plan 10-03 (readManifest) plugs into the same package — `maxManifestSize`, `utf8BOM`, and `readManifest` are design-provided in RESEARCH Pattern 4.
- Phases 11-13 extend `registry.go`'s `detectors` slice literal in cascade order (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) — the order contract comment is in-file; init-order across files is lexical, never relied on.
- Tracer verified end-to-end: every layer the phase touches (path, errors, content gate, dispatch, model) is exercised by the 21 passing tests at 100% package coverage.

---
*Phase: 10-package-foundation-api-contract-repo-cleanup*
*Completed: 2026-09-29*

## Self-Check: PASSED

- FOUND: all 10 project_probe files (5 production + 4 test + 1 example)
- FOUND: `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-02-SUMMARY.md`
- FOUND: commit `57cd875` (feat(10): implement projectprobe probe contract)
- FOUND: commit `e1eac0c` (docs(10): document never-fail probe contract)
- PASS: `go test ./project_probe/...` exits 0 (21 tests)
- PASS: `go build ./...` exits 0 (module-wide)
- PASS: `GOOS=windows go build` + `go vet ./project_probe/...` exit 0
- PASS: `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → 0 issues
- PASS: bare `path` import grep empty; ROBT-05 anti-feature grep empty
- PASS: no `t.Parallel()` in registry_test.go (count 0)
- PASS: project_probe statement coverage 100% (pkg ≥80%, file ≥70%, total 77.6% ≥ 75%)
- NOTE: `make coverage-quick` exits 2 on pre-existing `release/update.go` (68.9% < 70%) — out of scope, logged to deferred-items.md