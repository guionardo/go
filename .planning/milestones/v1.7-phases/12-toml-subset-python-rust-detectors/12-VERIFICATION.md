---
phase: 12-toml-subset-python-rust-detectors
verified: 2026-09-29T13:46:32Z
status: passed
score: 4/4 must-haves verified
covered_files:
  - .planning/phases/12-toml-subset-python-rust-detectors/12-01-PLAN.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-01-SUMMARY.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-02-PLAN.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-02-SUMMARY.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-03-PLAN.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-03-SUMMARY.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-04-PLAN.md
  - .planning/phases/12-toml-subset-python-rust-detectors/12-04-SUMMARY.md
  - project_probe/toml.go
  - project_probe/toml_test.go
  - project_probe/detect_python.go
  - project_probe/detect_python_test.go
  - project_probe/detect_rust.go
  - project_probe/detect_rust_test.go
  - project_probe/registry.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - project_probe/detect_php_test.go
covered_digest: "v2:sha256:11619045a8521807c98f2956a5afaed13f29cae28445d06f37c119605199e756"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 4/4
  gaps_closed: []
  gaps_remaining: []
  regressions:
    - "None — phase-12 core files (toml.go, toml_test.go, detect_python.go, detect_python_test.go, detect_rust.go, detect_rust_test.go) are byte-identical to commit 596d58b (git diff empty); the registry slots Python@1 and Rust@4 hold their positions with all 7 slots now live (phase 13)"
advisory: # New-scope findings in re-verification mode — none; digest refresh only (#4682)
  - finding: "None — this regeneration exists because phases 13/14 modified phase-12 covered files (doc.go, registry.go, registry_test.go, detect_php_test.go — all additive: doc refresh, all-7-slots-live registry, new cascade rows). No phase-12 must-have regressed"
    category: other
    reason: "Covered-file digest was stale (#4682); re-verified all must-haves against the current tree with fresh behavioral evidence"
    evidence_status: "none provided"
behavior_unverified_items: [] # 0 — SC4 skip-state transitions exercised by 13 pinned matrix subtests + verifier's 12-subtest production probe
coincidental_reliance_items: [] # 0 — all verified truths hold on code-enforced state transitions (quote-aware scans in toml.go), not fixtures/ordering
---

# Phase 12: TOML Subset + Python/Rust Detectors Verification Report

**Phase Goal:** Python and Rust projects are detected via the unexported section-aware TOML-subset reader.
**Verified:** 2026-09-29T13:46:32Z
**Status:** passed
**Re-verification:** Yes — digest refresh (#4682): later phases (13/14) modified phase-12 covered files after the prior verifier ran; all must-haves re-verified against the current tree.

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (the contract):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Probe of a pyproject.toml folder with `[project]` returns Language=Python with PEP 621 name/version/description | ✓ VERIFIED | `detect_python.go:20` `readTOMLSection(content, "project")`; file byte-identical to the phase-12 close commit `596d58b` (git diff empty); `TestProbe_PythonEndToEnd` passes (named run + full suite) |
| 2 | SC2 — Poetry-managed folder (`[tool.poetry]`) returns Language=Python with poetry name/version/description | ✓ VERIFIED | `detect_python.go:21-23` len==0 whole-section fallback; `TestProbe_PythonPoetryLegacy`, `TestProbe_PythonEmptyProjectFallsToPoetry` pass |
| 3 | SC3 — Cargo.toml `[package]` returns Language=Rust with name/version/description; `version.workspace = true` → empty Version, never fabricated | ✓ VERIFIED | `detect_rust.go:23` single section read; dotted keys rejected by `isBareKey` (toml.go:138-150); `TestProbe_RustEndToEnd`, `TestProbe_RustWorkspaceVersion`, `TestProbe_RustWorkspaceDescription` pass |
| 4 | SC4 — Malformed or unsupported TOML (dotted keys, multiline strings, inline tables) degrades strictly to empty fields — never partial data, nil error, no panic | ✓ VERIFIED (STILL HOLDS) | CR-01 fix intact in `toml.go` (byte-identical to `596d58b`): `closesMultiLine` (toml.go:214-234), `clearsBracket` (toml.go:247-282), `opensMultiLine` + cross-line `pendingMLS` (toml.go:294-349/36-50), `enterSkip` same scans (toml.go:187-203). 13 pinned QuoteAware subtests pass; verifier's production probe re-run: **12/12 subtests green — 9/9 CR-01 shapes produce no fabrication** (control, 6-quote `"""`/`'''` body lines, 4-quote opening lines, `]`/`}` inside quoted strings in arrays/inline tables, fake `[project]` header in a `[build-system]` string body, multi-line-string opener in bracket body, bracket-in-string on opening line), plus 3 end-to-end rows through `detectPython`/`detectRust` into `ProjectData` |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Plan-Level Must-Have Mapping (all still hold)

| Plan | Must-have | Status |
|------|-----------|--------|
| 12-01 | T1 extract name/version/description from `[project]` | ✓ VERIFIED |
| 12-01 | T2 missing section → empty map, never error | ✓ VERIFIED |
| 12-01 | T3 legal-but-unsupported TOML never stores partial data | ✓ VERIFIED (STILL HOLDS — 6-quote/4-quote/bracket-in-string probe rows pin no-fabrication) |
| 12-01 | T4 global skip states — no parsing inside bodies, ANY section | ✓ VERIFIED (STILL HOLDS — `FakeHeaderAfterDelimiterRun` + `CrossSectionSkip` pass; probe: fake `[project]` header after 6-quote line in `[build-system]` body → `map[]`) |
| 12-01 | T5 exact header equality, no prefix matching | ✓ VERIFIED |
| 12-01 | T6 CRLF/BOM, duplicate last-wins, garbage remainder | ✓ VERIFIED |
| 12-01 | T7 never panics on pathological input | ✓ VERIFIED (`TestReadTOMLSection_Adversarial` passes) |
| 12-02 | T1..T4, T6, T7 (Python PEP 621, poetry fallback, whole-section wins, dynamic → "", fallback chains, registry index 1) | ✓ VERIFIED — registry still has `detectPython`@1 (`registry.go:12`) |
| 12-02 | T5 malformed pyproject → folder-base Name + empty fields | ✓ VERIFIED (STILL HOLDS — probe: pyproject with only `name="evil"` inside a 6-quote body → `Name="evilproj"` folder-base, empty Version, never "evil") |
| 12-03 | T1..T7 (Rust `[package]`, version.workspace → "", description.workspace → README fallback, array/boolean degrade, virtual manifest, index 4 + 5-of-7 + cascade, doc.go) | ✓ VERIFIED — registry has `detectRust`@4 (`registry.go:15`); doc.go documents both detectors (lines 44-46) |
| 12-04 | SC4 / truth 4 / truth 5 restored (quote-aware skip-state scanning) | ✓ VERIFIED (STILL HOLDS) — 5 QuoteAware tests (13 subtests) pass; probe re-run 12/12 green |

### Deferred Items

None. FRAM-02 (full TOML dependency) remains deferred to v2 by locked CONTEXT decisions — not a phase-12 gap.

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | None — digest-refresh regeneration only; no new-scope concerns | other | Phase 13/14 file changes (doc.go refresh, all-7-slots-live registry, new cascade rows, fuzz corpus) are additive and verified by their own phases |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/toml.go` | readTOMLSection reader + strict degrade + global skip states, quote-aware (CR-01 fix) | ✓ VERIFIED | 349 lines, byte-identical to `596d58b`; `closesMultiLine`/`clearsBracket`/`opensMultiLine`/`pendingMLS` all present; stdlib-only; no anti-features (greps clean) |
| `project_probe/toml_test.go` | Pure-content matrix incl. QuoteAware rows | ✓ VERIFIED | 468 lines, 34 tests; 5 `TestReadTOMLSection_QuoteAware_*` tests (13 subtests) pin all CR-01 shapes; all pass |
| `project_probe/detect_python.go` | detectPython | ✓ VERIFIED | 35 lines, unchanged since `596d58b`; readManifest → readTOMLSection, fallback chains |
| `project_probe/detect_python_test.go` | Python fixture matrix | ✓ VERIFIED | All tests pass (named run + full suite) |
| `project_probe/detect_rust.go` | detectRust | ✓ VERIFIED | 35 lines, unchanged since `596d58b`; single `[package]` read |
| `project_probe/detect_rust_test.go` | Rust fixture matrix | ✓ VERIFIED | All tests pass (named run + full suite) |
| `project_probe/registry.go` | Slots 1 and 4 live | ✓ VERIFIED | `detectPython`@1, `detectRust`@4 — positions preserved; all 7 slots now live (phase 13 additive change) |
| `project_probe/registry_test.go` | TestDetectorPositions pin | ✓ VERIFIED | NotNil 0-6; no `t.Parallel` (global-state rule); all pass |
| `project_probe/doc.go` | Detector list refresh | ✓ VERIFIED | Python/Rust documented (lines 44-46); version semantics contract added by phase 14 — additive |
| `project_probe/detect_php_test.go` | Cascade rows | ✓ VERIFIED | Phase-12 rows (Python@1 beats Rust@4, broken-JS@3) intact; phase 13 added C#/Java rows — additive; `TestProbe_CascadePrecedence` passes |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| readTOMLSection skipDelim loop | closesMultiLine | `skipDelim != ""` case calls `closesMultiLine(line, skipDelim)`; clears only on net-closed (toml.go:30-34) | WIRED | 6-quote body lines (`""""""`, `''''''`) keep the skip state; plain `"""` clears it — probe rows 2/3 confirm |
| readTOMLSection skipClose loop | clearsBracket + opensMultiLine + pendingMLS | `skipClose != 0` case: pendingMLS checked first, then opensMultiLine seeds pendingMLS, then clearsBracket at depth 0 clears (toml.go:35-54) | WIRED | `]`/`}` inside quoted strings never clear; `"""`/`'''` opener inside a bracket body keeps later `]`/`}` inside the string — probe rows 5/6/8 confirm |
| enterSkip | closesMultiLine/clearsBracket/opensMultiLine | same-line-close checks use the same scans (toml.go:187-203); 4-quote opening line enters multi-line state; bracket-in-string opening line enters bracket state (toml.go:64-70) | WIRED | Probe rows 4/9 confirm both missed-entry paths remain closed |
| detectPython | readTOMLSection | `readManifest(folder,"pyproject.toml")` → `readTOMLSection(content,"project")` with len==0 fallback to `"tool.poetry"` | WIRED | detect_python.go:16-23; end-to-end probe: 6-quote-body pyproject → `Name="acme"`, no fabrication |
| detectRust | readTOMLSection | `readManifest(folder,"Cargo.toml")` → `readTOMLSection(content,"package")` | WIRED | detect_rust.go:19-23; end-to-end probe: `]`-in-string Cargo.toml → `Name="crate"`, `Version="1.2.3"`, no fabrication |
| registry | detectors | `detectPython`@1, `detectRust`@4; runDetectors nil-skip + callDetector recover | WIRED | registry.go:12-15, 31-41; positions unchanged by the phase-13 all-slots-live update |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detectPython | Name/Version/Description | readManifest bytes → readTOMLSection section map → ProjectData | Yes — real file content | ✓ FLOWING — no fabrication on any CR-01 shape (probe rows 1-4, 6, 9; e2e rows 1-2) |
| detectRust | Name/Version/Description | readManifest bytes → readTOMLSection `[package]` → ProjectData | Yes — real file content | ✓ FLOWING — no fabrication on any CR-01 shape (probe rows 5, 8; e2e row 3) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| QuoteAware pinned matrix | `go test ./project_probe/... -run 'TestReadTOMLSection_QuoteAware'` | 13 passed | ✓ PASS |
| SC1/SC2/SC3 named tests | `go test ./project_probe/... -run 'TestProbe_Python\|TestProbe_Rust\|TestDetectorPositions'` | 26 passed | ✓ PASS |
| SC1/SC2/SC3 end-to-end + cascade | `go test ./project_probe/... -run 'TestProbe_RustWorkspaceVersion\|TestProbe_RustWorkspaceDescription\|TestProbe_PythonPoetryLegacy\|TestProbe_PythonEmptyProjectFallsToPoetry\|TestProbe_PythonEndToEnd\|TestProbe_RustEndToEnd\|TestProbe_CascadePrecedence'` | 17 passed | ✓ PASS |
| Full project_probe suite | `go test ./project_probe/...` | 284 passed (phase-12 tests all green; 185 at phase-12 close + phase 13/14 additions) | ✓ PASS |
| Repo-wide build | `go build ./...` | clean | ✓ PASS |
| Cross-platform vet | `go vet ./project_probe/...` + `GOOS=windows go vet ./project_probe/...` | both clean | ✓ PASS |
| Coverage gate | `make coverage-quick` | file 70% PASS / package 80% PASS / total 80.7% (≥75%) PASS; toml.go file 87.7% (≥70%), project_probe pkg 93.9% (≥80%) | ✓ PASS |
| CR-01 probe row 1 (control) — simple multi-line string body | in-package verifier test vs production `readTOMLSection` | `name="acme"` preserved | ✓ PASS |
| CR-01 probe row 2 — 6-quote line in `"""` body | verifier test | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 3 — 6-quote line in `'''` body | verifier test | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 4 — 4-quote opening line | verifier test | **no fabrication** — `map[]` | ✓ PASS |
| CR-01 probe row 5 — `]` inside quoted string in multi-line array | verifier test | **no fabrication** — `map[name:crate, version:1.2.3]` | ✓ PASS |
| CR-01 probe row 6 — `}` inside quoted string in inline table | verifier test | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 7 — fake `[project]` header after 6-quote line in `[build-system]` | verifier test | **no section switch** — `map[]` | ✓ PASS |
| CR-01 probe row 8 — multi-line-string opener inside bracket body | verifier test | **no fabrication** — `map[name:crate]` | ✓ PASS |
| CR-01 probe row 9 — `]` inside quoted string ON array opening line | verifier test | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe e2e — detectPython (6-quote body line) | verifier test vs production `detectPython` | **`ProjectData.Name="acme"`** — no fabrication | ✓ PASS |
| CR-01 probe e2e — detectPython (fake header only) | verifier test vs production `detectPython` | **folder-base `Name="evilproj"`, empty Version — never "evil"** | ✓ PASS |
| CR-01 probe e2e — detectRust (`]` in quoted string) | verifier test vs production `detectRust` | **`Name="crate"`, `Version="1.2.3"`** — no fabrication | ✓ PASS |
| 12-02 truth 5 — malformed pyproject (only `name="evil"` inside body) | verifier test vs production `detectPython` | **folder-base Name + empty Version — never "evil"** | ✓ PASS |
| Anti-feature greps | `grep -rnE 'os/exec\|net/http\|EvalSymlinks\|WalkDir\|go-toml' project_probe/toml.go` + detectors | nothing (exit 1) | ✓ PASS |
| Direct-IO greps | `grep -nE 'os\.Open\|os\.ReadFile\|io\.ReadAll' project_probe/toml.go` + detectors | nothing (exit 1) | ✓ PASS |
| Debt-marker greps | `grep -nE 'TBD\|FIXME\|XXX'` on all phase-12 covered source files | nothing (exit 1) | ✓ PASS |

The temporary verifier probe file (`project_probe/zz_verify_probe_test.go`) was deleted after the run; `git status --porcelain project_probe/` is clean (no residue).

### Probe Execution

No phase-declared probes (`scripts/*/tests/probe-*.sh` — none exist for this phase). The CR-01 fabrication-shape probe was re-run by the verifier as an in-package test against current production code (Step 7b/7c behavior), covering the 9 reader-level shapes + 3 end-to-end rows: **12/12 green, zero fabrication** — plus the 12-02 truth 5 folder-base degrade row (included in the e2e fake-header row).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| ROBT-03 | 12-01, 12-04 | Unexported section-aware TOML-subset reader; strict degrade-to-empty | ✓ SATISFIED | Reader exists with exact-header matching and quote-aware global skip states; never-partial-data contract holds — 13 pinned subtests + 12-subtest probe show zero fabrication across all CR-01 shapes |
| DETC-05 | 12-02 | Python detector — pyproject `[project]` + legacy `[tool.poetry]`, PEP 621 fields | ✓ SATISFIED | Well-formed paths verified (SC1/SC2); malformed-input degrade yields folder-base Name + empty/fallback fields, never fabricated (probe e2e rows) |
| DETC-06 | 12-03 | Rust detector — Cargo.toml `[package]` fields | ✓ SATISFIED | Well-formed paths verified (SC3); malformed arrays/inline tables degrade with no fabrication (probe rows 5/8) |
| FRAM-02 | — | Full TOML dep deferred to v2 | — | Correctly out of scope (deferred in CONTEXT) |

No orphaned requirements: all Phase 12 IDs (DETC-05, DETC-06, ROBT-03) are claimed by plans (12-01/12-02/12-03/12-04), marked Complete in REQUIREMENTS.md (lines 22-23, 39, 87-88, 98), and re-verified against the current tree.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | None | — | No TBD/FIXME/XXX markers in phase-12 covered source files; no stub patterns; no anti-feature imports; no direct file I/O in toml.go/detectors; no `t.Parallel` in registry_test.go (global-state rule) |

### Human Verification Required

None. The behavior-dependent truths (SC4, 12-01 truth 4, 12-02 truth 5 — skip-state transitions/invariants) have behavioral evidence: the 13 pinned matrix subtests exercise each transition against production code and pass, and the verifier's 12-subtest production probe (including end-to-end detectPython/detectRust into ProjectData) shows zero fabrication. No visual, real-time, or external-service behavior is at stake.

### Gaps Summary

None. This is a digest-refresh regeneration (#4682), not a gap-closure round:

1. **Why regenerated:** Phases 13/14 modified phase-12 covered files after the prior verifier ran — `doc.go` (detector-list + version-semantics docs), `registry.go` (all 7 slots live; Python@1 and Rust@4 positions preserved), `registry_test.go` (all-slot pin), `detect_php_test.go` (additive C#/Java cascade rows), plus new phase-13 files and fuzz corpus outside this phase's scope.
2. **What was re-verified:** All 4 roadmap SCs and all plan-level must-haves against the current tree. Phase-12 core files (`toml.go`, `toml_test.go`, `detect_python.go`, `detect_python_test.go`, `detect_rust.go`, `detect_rust_test.go`) are byte-identical to the gap-closure commit `596d58b` — the SC4 no-fabrication contract holds unchanged: **the fabrication shapes (6-quote body lines, 4-quote opening lines, brackets-in-quotes, fake headers in bodies) produce no data**, confirmed by the re-run 12-subtest probe.
3. **Regressions:** None — full project_probe suite 284/284, `go build ./...` clean, `go vet` + `GOOS=windows go vet` clean, coverage thresholds pass (toml.go 87.7%, project_probe 93.9%, total 80.7%). The under-skip residual (a construct closer sharing a line with the string's close is scanned on the next line) remains the documented safe direction — degrades to empty, never fabricates (12-04 flagged-assumptions).

---

_Verified: 2026-09-29T13:46:32Z_
_Verifier: the agent (gsd-verifier)_