---
phase: 13-xml-detectors-csharp-net-java-kotlin
plan: 01
subsystem: core-library
tags: [go, encoding/xml, csproj, xml, detector, project_probe, msbuild]

# Dependency graph
requires:
  - phase: 12-toml-subset-python-rust-detectors
    provides: readManifest (1 MB cap + BOM strip + WR-01 FIFO gate), readmeDescription, registry literal + nil-skip + callDetector, never-fail presence-match contract
provides:
  - detectCSharp presence-match C#/.NET detector (.csproj via readFirstManifest discovery + inline xml.Unmarshal decode)
  - readFirstManifest — single-level sorted os.ReadDir discovery for variable-named manifests
  - Registry slot 2 filled (6-of-7 live); TestDetectorPositions interim flip (detectors[2] NotNil, [5] Nil)
affects: [13-02 (Java/Kotlin — detectors[5] flip, doc.go refresh, cascade rows), verify-work, gsd-verify-work]

# Actuals (#2632) — pairs with the plan's estimate (34000 tokens)
actuals:
  tokens: 5318      # 21274 diff chars / 4 over the realized diff
  tasks: 2          # tasks completed
  commits: 2        # MEASURED: git rev-list --count caff0ac..HEAD (#3968)
  plan_head_before: caff0ac07252b8f324c1469dc2bc1c5874601b44
  plan_head_after: 59e5c67b17861bb0669f645fc71f353a4a91ce82

# Tech tracking
tech-stack:
  added: []         # stdlib only — encoding/xml, os, strings, path/filepath (zero installs, T-13-SC)
  patterns:
    - "readFirstManifest discovery: single-level os.ReadDir (sorted), exact-case suffix, e.IsDir skip, per-candidate readManifest WR-01 gate — first readable candidate wins (A3)"
    - "Inline decode-error-ignored xml.Unmarshal (_ = xml.Unmarshal(content, &proj)) — NO shared readXMLManifest/xml.go helper (0% file coverage would fail file:70 gate)"
    - "XMLName local-name matching (xml:\"Project\", plain child tags) — namespace-agnostic, one struct for SDK-style and old-style xmlns files (D-01)"
    - "Collect-first-then-chain across ALL PropertyGroups in document order (D-disc-6) — AssemblyName priority independent of group position (probe row J)"

key-files:
  created:
    - project_probe/detect_csharp.go
    - project_probe/detect_csharp_test.go
  modified:
    - project_probe/registry.go
    - project_probe/registry_test.go

key-decisions:
  - "XML decode is INLINE in detectCSharp (decode-error-ignored); xml.go/readXMLManifest deliberately NOT created — an uncalled unexported helper sits at 0% file coverage and fails the .testcoverage-quick.yml file:70 gate (no project_probe override)"
  - "readFirstManifest is the one new mechanism: .csproj is a variable filename, so discovery (sorted os.ReadDir, exact-case suffix, e.IsDir skip, per-candidate WR-01 gate) precedes the read (D-04, A3, A6, A9)"
  - "Version chain Version → VersionPrefix → '' executed with NEVER-fabricated empty (MSBuild-implicit 1.0.0 not reported, D-03/DATA-03); $(...) placeholders reported raw (A2)"
  - "A1 interpretation executed: Description chain is the FULL DATA-04 chain — csproj <Description> → readmeDescription → empty, matching every other manifest-bearing detector"
  - "TestDetectorPositions interim flip only (Pitfall 9): detectors[2] NotNil, detectors[5] stays Nil for 13-02's RED step; no parallel marker added (AGENTS.md global-state rule)"

patterns-established:
  - "Pattern: variable-named manifest discovery via readFirstManifest (os.ReadDir sorted → exact-case suffix → readManifest gate per candidate)"
  - "Pattern: inline _ = xml.Unmarshal decode-error-ignored with XMLName local-name structs (namespace-agnostic, D-01)"
  - "Pattern: collect-first-then-chain across repeated container elements (PropertyGroups)"

requirements-completed: [DETC-07]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: ".csproj folder probes as LanguageCSharp with AssemblyName/Version/Description verbatim — SDK-style, old-style namespaced (xmlns msbuild/2003), and BOM-prefixed (SC1, D-01/D-02/D-03)"
    requirement: DETC-07
    verification:
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpOldStyleNamespaced"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpBOM"
        status: pass
    human_judgment: false
  - id: D2
    description: "Chain and edge behaviors pinned: multi-PropertyGroup collection, VersionPrefix chain, RootNamespace chain, placeholder raw, description manifest-wins/README-fallback, root-scope hygiene, empty-file presence match, malformed presence match"
    requirement: DETC-07
    verification:
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpMultiGroup"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpVersionPrefixChain"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpRootNamespaceChain"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpPlaceholderRaw"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpDescriptionFromManifest"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpDescriptionReadmeFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpSubdirNamedCsproj"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpEmptyFile"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpMalformedPresence"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestProbe_CSharpRootScope"
        status: pass
    human_judgment: false
  - id: D3
    description: "Registry slot 2 live with interim position pin — detectors[2] NotNil, detectors[5] stays Nil for 13-02's RED (D-08, Pitfall 9)"
    requirement: DETC-07
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-09-29
status: complete
---

# Phase 13 Plan 1: C#/.NET Detector Summary

**C#/.NET detector end-to-end: readFirstManifest .csproj discovery + inline namespace-agnostic xml.Unmarshal decode (XMLName local-name), D-03 chains, registry slot 2 filled with interim TestDetectorPositions flip — 15 probe-level test rows pinning the DETC-07 contract**

## Performance

- **Duration:** 7 min
- **Started:** 2026-09-29T08:30:41Z
- **Completed:** 2026-09-29T08:37:41Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified; 540 insertions, 8 deletions)

## Accomplishments

- `detectCSharp(folder) (ProjectData, bool)` presence-match detector: `readFirstManifest(folder, ".csproj")` discovery (single-level sorted `os.ReadDir`, exact-case suffix, `e.IsDir()` skip, per-candidate `readManifest` WR-01 FIFO gate) → inline `_ = xml.Unmarshal(content, &proj)` decode-error-ignored → collect-first-then-chain across ALL PropertyGroups → D-03 chains (Name: AssemblyName → RootNamespace → folder base; Version: Version → VersionPrefix → empty, never the MSBuild-implicit 1.0.0; Description: manifest → `readmeDescription` → empty)
- Namespace-agnostic decode (D-01): `csprojManifest`/`propertyGroup` with `xml:"Project"` XMLName and plain child tags — one struct decodes SDK-style `<Project Sdk=...>`, old-style `<Project xmlns=...msbuild/2003>`, and BOM-prefixed files (D-02 at the readManifest boundary)
- NO `xml.go` — the decode is inline; a shared helper would sit at 0% file coverage and fail the `make coverage-quick` file:70 gate (Pitfall 10 rationale)
- Registry slot 2 filled in place (D-08): `detectCSharp` live at index 2, 6-of-7 slots now live; `TestDetectorPositions` interim flip pins `detectors[2]` NotNil while `detectors[5]` stays Nil for 13-02's RED (Pitfall 9)
- 15 new test rows (6 tracer + 9 expansion): end-to-end, old-style, BOM, malformed presence, root-scope, missing manifest, multi-group, VersionPrefix chain, RootNamespace chain, placeholder raw, description manifest-wins, README fallback, subdir-named-csproj skip, empty-file presence, direct-table edges — suite grew 185 → 206 tests, project_probe at 93.8% coverage

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — C#/.NET detector end-to-end through Probe, RED→GREEN** - `8b02ab2` (feat)
2. **Task 2: Expansion — pin C#/.NET detector edges** - `59e5c67` (test)

**Plan metadata:** `commits` counted 2 from ledger base `caff0ac` (#3968); metadata commit follows.

## Files Created/Modified

- `project_probe/detect_csharp.go` - `readFirstManifest` discovery + `csprojManifest`/`propertyGroup` decode structs + `detectCSharp` presence-match detector with inline decode-error-ignored xml.Unmarshal and collect-first-then-chain
- `project_probe/detect_csharp_test.go` - 15 probe-level rows (6 tracer + 9 expansion), all `t.Parallel()`-safe temp-dir fixtures
- `project_probe/registry.go` - slot 2 `nil` → `detectCSharp` (surgical, no reordering); nil-skip/callDetector unchanged
- `project_probe/registry_test.go` - `TestDetectorPositions` interim flip: `detectors[2]` NotNil, `detectors[5]` stays Nil (single assertion change; no parallel marker)

## TDD Gate Compliance

Plan type `tdd` — executed per the Phase 10/11 approved TDD adaptation (RED verified-but-uncommitted; RED evidence recorded in the feat commit body):

| Gate | Evidence | Status |
|------|----------|--------|
| RED | `go test ./project_probe/... -run 'TestProbe_CSharp\|TestDetectCSharp'` → exit 1, `detect_csharp_test.go:146:12: undefined: detectCSharp` (build failure — the target test failed on the planned behavior absence); test file written first, left uncommitted in the working tree | ✓ (verified-but-uncommitted per adaptation; evidence in `8b02ab2` body) |
| GREEN | `feat(13): implement C#/.NET detector` (`8b02ab2`) — implementation + registry edits + test file; 191/191 project_probe tests green | ✓ |
| REFACTOR | None needed — implementation shipped in final shape (no cleanup pass required) | — (n/a) |
| Expansion | `test(13): pin C#/.NET detector edges` (`59e5c67`) — 9 edge rows; 206/206 green | ✓ |

No `test(...)` RED commit exists by design: the plan's acceptance criteria (`git log --oneline -1` = `feat(13): implement C#/.NET detector` after Task 1) and the Phase 10 precedent lock the adaptation. Pre-commit hooks are not installed in this checkout (RESEARCH Pitfall 11); all verification ran manually before each commit.

## Decisions Made

- XML decode inline in `detectCSharp`; `xml.go`/`readXMLManifest` deliberately NOT created (coverage-gate rationale — dropped in plan revision)
- `readFirstManifest` as the one new mechanism (variable-named manifest discovery); first readable candidate in sorted ReadDir order (A3 stopgap; REFN-04 defers the real multi-csproj rule)
- A1 interpretation executed: Description = full DATA-04 chain (manifest `<Description>` → README → empty)
- A2 executed: `$(...)` placeholders reported raw, never resolved (no code branch)
- Interim registry flip only (Pitfall 9) — `detectors[5]` stays Nil for 13-02's RED

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `os.Open` token in doc comment tripped the direct-I/O acceptance grep**
- **Found during:** Task 1 (GREEN verification)
- **Issue:** `detect_csharp.go`'s `readFirstManifest` doc comment said "os.Open alone resolves case-insensitively..." — the acceptance grep `grep -nE 'os\.Open|os\.ReadFile|io\.ReadAll'` matched the prose token although no direct I/O exists (all reads route through `readManifest`)
- **Fix:** Reworded the comment to "a plain open alone resolves case-insensitively" — meaning preserved, literal token removed
- **Files modified:** project_probe/detect_csharp.go
- **Verification:** grep re-run prints nothing (exit 1); full suite re-run green
- **Committed in:** 8b02ab2 (part of Task 1 commit)

**2. [Rule 1 - Bug] `makeFIFO` token in test doc comment tripped the FIFO-helper acceptance grep**
- **Found during:** Task 2 (verification)
- **Issue:** `TestDetectCSharp`'s skip-rationale comment said "manifest_fifo_test.go carries the makeFIFO helper under a !windows build tag" — the acceptance grep `grep -nE 'syscall\.Mkfifo|makeFIFO'` matched the prose token (no FIFO fixture exists in the untagged file)
- **Fix:** Reworded to "carries the FIFO-creating helper" — skip rationale preserved
- **Files modified:** project_probe/detect_csharp_test.go
- **Verification:** grep re-run prints nothing (exit 1); 206/206 suite green
- **Committed in:** 59e5c67 (part of Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 — grep-token-in-comment, both cosmetic, no behavior change)
**Impact on plan:** None — both fixes were doc-comment rewordings required to satisfy the plan's literal acceptance greps; no scope creep, no behavior change.

## Issues Encountered

- **`git.base-branch --is-protected` verb mismatch:** `gsd_run query git.base-branch --is-protected <branch>` echoes the configured base branch name ("main") rather than a boolean; the executor protocol's strict non-"false" reading would false-FATAL every branch. Resolved by comparing the checked branch against the echoed base branch name (semantic intent: never commit onto the base). `gsd/v1.7-project-probe` is the AGENTS.md-mandated milestone working branch where all Phase 10-13 commits live — not protected.
- **`make coverage-quick` known-red (documented Pitfall 10):** the gate fails on `release/update.go` 68.9% vs 70% file threshold — pre-existing, unrelated, NOT fixed (scope boundary). Project_probe rows pass: package 93.8% (≥80%), `detect_csharp.go` 100% file (≥70%), total ≥75%.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for **plan 13-02** (Java/Kotlin detector): `detectors[5]` still Nil for its RED step (Pitfall 9); `doc.go` detector-list refresh and `TestProbe_CascadePrecedence` C#/Java rows land there
- C#@2 now live in the cascade — mixed `.csproj + package.json` folders resolve to LanguageCSharp (consequence pinned in 13-02's cascade rows)
- Phase success criterion 1 (C# part) and criterion 4 (C# part) satisfied by the 15 pinned rows

---
*Phase: 13-xml-detectors-csharp-net-java-kotlin*
*Completed: 2026-09-29*

## Self-Check: PASSED

- Files: `project_probe/detect_csharp.go`, `project_probe/detect_csharp_test.go`, `13-01-SUMMARY.md` all present
- Commits: `8b02ab2` (feat) and `59e5c67` (test) verified in git history
- Anti-feature greps (`os/exec|net/http|EvalSymlinks|WalkDir`, `Entity|CharsetReader`, `os.Open|os.ReadFile|io.ReadAll`) print nothing over `detect_csharp.go`
- `project_probe/xml.go` absent — decode is inline (file:70 coverage gate)