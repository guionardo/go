---
phase: 10-package-foundation-api-contract-repo-cleanup
plan: 03
subsystem: api-contract
tags: [project-probe, read-manifest, size-cap, bom, tdd, rob-02]

# Dependency graph
requires:
  - phase: 10-package-foundation-api-contract-repo-cleanup
    provides: "10-01: project_detector/ deleted — go build ./... green module-wide"
  - phase: 10-package-foundation-api-contract-repo-cleanup
    provides: "10-02: projectprobe never-fail probe contract, registry (ProjectData, bool), ignore gate"
provides:
  - "Shared readManifest(folder, name) ([]byte, bool) helper — 1 MB size cap via the io.LimitReader(maxManifestSize+1) truncation probe and UTF-8 BOM (EF BB BF) strip (ROBT-02)"
  - "maxManifestSize = 1 << 20 as the single source of truth for the cap, consumed by the limit+1 truncation probe"
  - "The package's only file reader — never errors, never panics, never OOMs; callers degrade to ok=false (D-03)"
affects: [phases 11-13 (detector bodies consume readManifest), CI, consumers of projectprobe]

# Actuals (#2632) — pairs with the plan's estimate (10000 tokens) to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 1062
  tasks: 2
  commits: 1

# Commit ledger (#3968) — measured, never narrated
plan_head_before: 4d228534bc7fe2cdb879a362c08cf00eaf925beb
plan_head_after: 16becb8d0bc4590e85787c27d9fe98d63f88c20a

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Size-capped file read via the limit+1 truncation probe: io.LimitReader(f, cap+1) + ReadAll + len > cap → non-match — exact and race-free, never silently truncates (httptest_mock/request.go:213 LimitReader precedent)"
    - "UTF-8 BOM strip with bytes.TrimPrefix(content, utf8BOM) — three bytes, one call, no state machine"
    - "([]byte, bool) helper signature keeps the never-fail contract consistent with the registry's (ProjectData, bool) (D-03)"

key-files:
  created:
    - "project_probe/manifest.go — maxManifestSize const, utf8BOM var, readManifest (ROBT-02)"
    - "project_probe/manifest_test.go — TestReadManifest with 8 boundary rows (missing / exactly 1 MB / over 1 MB / BOM-stripped / no BOM / BOM-only / dir-at-path / empty-name)"
  modified: []

key-decisions:
  - "RED verified-but-uncommitted per the plan's TDD adaptation (10-02 precedent): the pre-commit go-test hook runs go test ./... and CI must stay green, so the failing boundary tests ship with the implementation; RED evidence (go test exit 1, undefined: readManifest) recorded in the feat commit message body"
  - "Boundary tests reference maxManifestSize (the single source of truth) instead of a literal 1<<20, and use literal EF BB BF bytes for BOM rows — tests stay independent of the implementation var while pinning the cap contract"
  - "Named returns (content []byte, ok bool) per RESEARCH Pattern 4 skeleton verbatim — semantically the plan's ([]byte, bool) signature"
  - "defer f.Close() //nolint: errcheck follows the config/profile/profile.go:51 precedent"

patterns-established:
  - "Pattern: limit+1 truncation probe (io.LimitReader) as the repo's capped-read idiom — exactly-at-cap accepted, cap+1 rejected as non-match"

requirements-completed: [ROBT-02]

coverage:
  - id: D1
    description: "readManifest 1 MB size cap — exactly 1 MB accepted (content intact), 1 MB + 1 byte rejected as (nil, false) via the limit+1 truncation probe; never silently truncated, never OOM (T-10-08)"
    requirement: ROBT-02
    verification:
      - kind: unit
        ref: "project_probe/manifest_test.go#TestReadManifest rows exactly_1mb, over_1mb"
        status: pass
    human_judgment: false
  - id: D2
    description: "UTF-8 BOM (EF BB BF) strip — BOM-prefixed content returned clean, no-BOM content unchanged, BOM-only file valid with empty content"
    requirement: ROBT-02
    verification:
      - kind: unit
        ref: "project_probe/manifest_test.go#TestReadManifest rows bom_stripped, no_bom, bom_only"
        status: pass
    human_judgment: false
  - id: D3
    description: "Never-fail contract — missing file, directory at the joined path, and empty name all return (nil, false); pathological input never panics; filepath.Join only (ROBT-04); Windows-safe"
    requirement: ROBT-02
    verification:
      - kind: unit
        ref: "project_probe/manifest_test.go#TestReadManifest rows missing_file, path_is_directory, empty_name"
        status: pass
      - kind: other
        ref: "GOOS=windows go vet ./project_probe/... (exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-09-29
status: complete
---

# Phase 10 Plan 3: Shared readManifest Helper Summary

**The shared safe-manifest helper `readManifest(folder, name) ([]byte, bool)` shipped under TDD — a 1 MB size cap enforced by the `io.LimitReader(maxManifestSize+1)` truncation probe (exactly 1 MB accepted, over-cap a non-match) and a UTF-8 BOM strip via `bytes.TrimPrefix`, with every failure mode degrading to `ok=false` so the package's never-fail contract (D-03) holds and phases 11-13 detectors get their file reader without a signature change.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-09-29T02:23:37Z
- **Completed:** 2026-09-29T02:28:28Z
- **Tasks:** 2 (RED → GREEN)
- **Files modified:** 2 created (1 production + 1 test)

## Accomplishments

- `readManifest(folder, name) (content []byte, ok bool)` — the package's only file reader, ready for detector consumption in phases 11-13 (ROBT-02)
- `const maxManifestSize = 1 << 20` — single source of truth for the cap, consumed by the `LimitReader(f, maxManifestSize+1)` truncation probe: exactly 1 MB accepted, over-cap rejected as a non-match, never silently truncated, never OOM (T-10-08 mitigation)
- `var utf8BOM = []byte{0xEF, 0xBB, 0xBF}` + `bytes.TrimPrefix` — three-byte, single-call BOM strip; BOM-only files are valid with empty content (A3)
- `([]byte, bool)` signature — no error channel anywhere; missing/unreadable/dir/over-cap all degrade to `ok=false` so callers fall to Unknown (D-03, never-fail contract)
- `filepath.Join` only, no bare `path` (ROBT-04); `// #nosec G304` rationale on the join — name is a compile-time constant at every call site, root-scoped by construction (DETC-01, T-10-09)
- TDD: 8 boundary rows (missing / exactly 1 MB / over 1 MB / BOM-stripped / no BOM / BOM-only / dir-at-path / empty-name) written first and verified failing (`undefined: readManifest`), then implemented — all 30 package tests green
- `readManifest` at 100% statement coverage; project_probe package total 100% (pkg ≥80%, file ≥70% satisfied)
- Windows-safe: `GOOS=windows go vet ./project_probe/...` clean; no `os/exec`/`net/http`/`EvalSymlinks`/`WalkDir` anti-features (ROBT-05)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — write the failing readManifest boundary tests** — *no commit* (plan adaptation: RED commits cannot land — the pre-commit go-test hook runs `go test ./...` and CI must stay green; tests were written first, verified failing with `undefined: readManifest`, and ship with the implementation in Task 2's commit)
2. **Task 2: GREEN — implement the size-capped, BOM-stripped readManifest** - `16becb8` (feat(10): add shared readManifest helper — message body records the RED evidence)

**Plan metadata:** pending docs commit at plan close (this SUMMARY + state files)

_Note: TDD commits use phase-level scope `feat(10)` per the plan's explicit commit-message contract (plan 10-03 Task 2 `<action>`), not `feat(10-03)` — see TDD Gate Compliance below._

## Files Created/Modified

- `project_probe/manifest.go` - `maxManifestSize = 1 << 20` const, `utf8BOM` var, `readManifest(folder, name) ([]byte, bool)` with LimitReader truncation probe + TrimPrefix BOM strip (ROBT-02)
- `project_probe/manifest_test.go` - `TestReadManifest` with 8 boundary rows: missing_file / exactly_1mb / over_1mb / bom_stripped / no_bom / bom_only / path_is_directory / empty_name (internal package, t.Parallel-safe — readManifest is stateless)

## Decisions Made

- **TDD RED adaptation (plan-sanctioned, 10-02 precedent):** tests written first in Task 1, verified failing before any implementation, then committed together with the implementation — the repo's pre-commit hooks (go test ./...) make a RED-only commit impossible without breaking CI. RED evidence recorded in the feat commit message body.
- **Test rows pin the cap via `maxManifestSize`** (single source of truth per plan key_links) rather than a duplicated literal, and use literal `\xEF\xBB\xBF` bytes for the BOM rows so the tests verify the behavior independently of the implementation's `utf8BOM` var.
- **Named returns `(content []byte, ok bool)`** — RESEARCH Pattern 4 skeleton copied verbatim; the plan's `([]byte, bool)` acceptance criterion is satisfied semantically (no `error` in the signature, bool present).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan-accuracy] RED failure output includes `undefined: maxManifestSize` alongside `undefined: readManifest`**
- **Found during:** Task 1 (RED verification)
- **Issue:** The plan's acceptance criteria expected `go test ./project_probe/...` to fail with `undefined: readManifest`; the actual compile error additionally reports `undefined: maxManifestSize` because the boundary rows reference the planned const (the single source of truth for the cap).
- **Fix:** None required — the RED state is achieved either way (exit 1, target tests cannot compile without the planned API); both symbols are absent pre-implementation, which is the strongest RED form. Documented the actual failure shape in the feat commit message.
- **Files modified:** none
- **Verification:** `go test ./project_probe/...` exits 1 pre-implementation (RED), then 0 post-implementation (GREEN, 30 tests)
- **Committed in:** 16becb8 (RED evidence in message body)

**2. [Rule 1 - Environment] golangci-lint v2.12.2 requires GOTOOLCHAIN=go1.26.4 (10-02-known toolchain panic)**
- **Found during:** Task 2 (GREEN pre-commit lint verification)
- **Issue:** The installed golangci-lint v2.12.2 is built with go1.26.3 and panics under the go1.27.0 system toolchain (pre-existing environment issue documented in 10-02).
- **Fix:** Ran lint and the commit with `GOTOOLCHAIN=go1.26.4` — golangci-lint runs cleanly (0 issues on project_probe). No repo files changed.
- **Files modified:** none (environment workaround only)
- **Verification:** `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → 0 issues; commit landed with hooks green
- **Committed in:** 16becb8

**3. [Rule 1 - Lint compliance] 8 lint issues in the two new files fixed before the hook would pass**
- **Found during:** Task 2 (GREEN verification)
- **Issue:** golangci-lint surfaced 8 issues: errcheck on `defer f.Close()` (manifest.go), wsl_v5 on the `if tt.wantOK` cuddle (manifest_test.go), plus the formatter set (gofmt/golines/goimports, 6×).
- **Fix:** `defer f.Close() //nolint: errcheck` (config/profile/profile.go:51 precedent), blank line before the if-block, and `golangci-lint fmt` for the formatter set.
- **Files modified:** project_probe/manifest.go, project_probe/manifest_test.go
- **Verification:** `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → "No issues found"; all 30 tests still pass
- **Committed in:** 16becb8

---

**Total deviations:** 3 auto-fixed (2 plan-accuracy/environment documentation, 1 lint compliance)
**Impact on plan:** All fixes were necessary for the commit gates (pre-commit hooks) to pass; no scope creep, no behavior changes to the ROBT-02 contract.

## TDD Gate Compliance

Plan type: `tdd` — RED → GREEN sequence executed with the plan's sanctioned adaptation:

| Gate | Expected commit | Actual | Status |
|------|-----------------|--------|--------|
| RED | `test(10-03): add failing test...` | no commit — RED verified-but-uncommitted (pre-commit go-test hook runs `go test ./...`; a failing tree cannot be committed; CI must stay green) | ADAPTED (plan Task 1 action; RED evidence recorded in the GREEN commit message) |
| GREEN | `feat(10-03): ...` | `16becb8 feat(10): add shared readManifest helper` | PASS — commit scope `(10)` per the plan's explicit commit-message contract, not `(10-03)` |
| REFACTOR | `refactor(10-03): ...` | — | N/A — the plan's 2-task structure has no refactor task |

- RED evidence: `go test ./project_probe/...` exited 1 pre-implementation with the target tests failing against the planned-but-absent API (compile-fail on `undefined: readManifest` and `undefined: maxManifestSize`) — verified before any production file existed. The `gsd_run check tdd-red-evidence` subcommand is not available in this gsd_run build (checked the available subcommand list); the plan-sanctioned record-in-commit-message path (10-02 precedent) was used instead, with a RED evidence JSON persisted for the audit trail.
- GREEN: implementation made all 8 Task-1 boundary tests (plus the 22-test 10-02 suite) pass (30 tests, exit 0), then coverage (project_probe 100%), Windows vet, and lint all green.
- Commit-scope note: the plan's `<action>` mandates `feat(10):` (matching the phase-10 commit history); the gate-validation regex `^feat(10-03):` in tdd.md would not match this — flagging is expected and by design.

## Issues Encountered

- **Pre-existing coverage gate failure (out of scope):** `make coverage-quick` exits 2 on `release/update.go` (68.9% vs 70% file threshold) — unrelated to this plan, reproducible before any project_probe change (project_probe itself is at 100% package/file coverage; package and total thresholds PASS at 77.7% total). Per the scope boundary rule this was not fixed; already logged to `deferred-items.md` in the phase directory (10-02).
- **golangci-lint toolchain panic (pre-existing environment):** see Deviation 2 — worked around with `GOTOOLCHAIN=go1.26.4`; permanent fix is `make install-golangci`.

## User Setup Required

None - no external service configuration required. (Optional: `make install-golangci` to upgrade the local golangci-lint past the go1.27 toolchain incompatibility.)

## Next Phase Readiness

- `readManifest(folder, name) ([]byte, bool)` is ready for detector consumption in phases 11-13 with no signature change: Go detector reads `go.mod`, Python reads `pyproject.toml`/`setup.py`, JS/TS reads `package.json`, .NET reads `.csproj`/`*.sln`, Rust reads `Cargo.toml`, Java reads `pom.xml`/`build.gradle`, PHP reads `composer.json` — each via the same capped, BOM-stripped, never-fail helper.
- Phases 11-13 extend `registry.go`'s `detectors` slice literal in cascade order (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP).
- ROBT-02 satisfied: size cap + BOM strip verified at the boundaries; the helper is the package's single file reader.

---
*Phase: 10-package-foundation-api-contract-repo-cleanup*
*Completed: 2026-09-29*

## Self-Check: PASSED

- FOUND: `project_probe/manifest.go` (35 lines: const, var, readManifest)
- FOUND: `project_probe/manifest_test.go` (TestReadManifest, 8 boundary rows)
- FOUND: `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-03-SUMMARY.md`
- FOUND: commit `16becb8` (feat(10): add shared readManifest helper)
- PASS: `go test ./project_probe/...` exits 0 (30 tests: 8 manifest boundary rows + 22 prior suite)
- PASS: `GOOS=windows go vet ./project_probe/...` exits 0
- PASS: `readManifest` at 100% coverage; project_probe package 100% (pkg ≥80%, file ≥70%)
- PASS: `grep -nE '^\s*"path"$' project_probe/manifest.go` empty (ROBT-04)
- PASS: anti-feature grep (os/exec|net/http|EvalSymlinks|WalkDir) empty (ROBT-05)
- PASS: no `error` in readManifest signature — `([]byte, bool)` never-fail contract (D-03)
- PASS: `GOTOOLCHAIN=go1.26.4 golangci-lint run ./project_probe/...` → "No issues found"
- NOTE: `make coverage-quick` exits 2 on pre-existing `release/update.go` (68.9% < 70%) — out of scope, already logged to deferred-items.md
