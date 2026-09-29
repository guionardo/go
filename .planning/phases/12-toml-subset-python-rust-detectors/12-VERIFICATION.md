---
phase: 12-toml-subset-python-rust-detectors
verified: 2026-09-29T22:45:00Z
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
covered_digest: "v2:sha256:4209675fb55052d4e6aecfefa5bb2413b99c94cadbc910102184c18f54e1df01"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/4
  gaps_closed:
    - "SC4 (roadmap SC 4): Malformed or unsupported TOML (dotted keys, multiline strings, inline tables) degrades strictly to empty fields — never partial data, nil error, no panic"
    - "12-01 must-have truth 4: Global skip states — multi-line strings/arrays/inline tables suppress header and keyval parsing until the closing delimiter — fake [project] or name = \"evil\" lines inside them are never parsed, in ANY section"
    - "12-02 must-have truth 5: Malformed pyproject.toml still matches on presence with folder-base Name and empty/fallback fields (D-09)"
  gaps_remaining: []
  regressions: []
advisory: # New-scope findings in re-verification mode — none; nothing unevidenced raised
  - finding: "None — the gap-closure round surfaced no new-scope concerns; all review carry-overs (WR-01, IN-01, IN-02, IN-04) are recorded in 12-REVIEW-DISPOSITION.md as advisory, not blockers"
    category: other
    reason: "Carry-overs documented and dispositioned in 12-REVIEW-DISPOSITION.md; none affect the 3 closed gaps"
    evidence_status: "none provided"
behavior_unverified_items: [] # 0 — every previously-failed truth now has behavioral evidence (pinned matrix subtests + production probe)
coincidental_reliance_items: [] # 0 — all verified truths hold on code-enforced state transitions, not fixtures/ordering
---

# Phase 12: TOML Subset + Python/Rust Detectors Verification Report

**Phase Goal:** Python and Rust projects are detected via the unexported section-aware TOML-subset reader.
**Verified:** 2026-09-29T22:45:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (plan 12-04, commit `596d58b`)

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (the contract):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Probe of a pyproject.toml folder with `[project]` returns Language=Python with PEP 621 name/version/description | ✓ VERIFIED | `detect_python.go` reads `readTOMLSection(content, "project")`; `TestProbe_PythonEndToEnd`, `TestReadTOMLSection_HappyPath` pass in the full suite (185 tests green) |
| 2 | SC2 — Poetry-managed folder (`[tool.poetry]`) returns Language=Python with poetry name/version/description | ✓ VERIFIED | `detect_python.go:21-23` len==0 whole-section fallback; `TestProbe_PythonPoetryLegacy`, `TestProbe_PythonEmptyProjectFallsToPoetry` pass |
| 3 | SC3 — Cargo.toml `[package]` returns Language=Rust with name/version/description; `version.workspace = true` → empty Version, never fabricated | ✓ VERIFIED | `detect_rust.go` single section read; D-07 via `isBareKey` rejecting `.` (toml.go:138-150); `TestProbe_RustEndToEnd`, `TestProbe_RustWorkspaceVersion`, `TestProbe_RustWorkspaceDescription` pass |
| 4 | SC4 — Malformed or unsupported TOML (dotted keys, multiline strings, inline tables) degrades strictly to empty fields — never partial data, nil error, no panic | ✓ VERIFIED (RESTORED) | CR-01 fixed in commit `596d58b`: `closesMultiLine` (run-of-3 closes, run>3 close+reopen, net across line) at toml.go:214-234, `clearsBracket` (in-string bracket immunity honoring `\` escapes, `[`/`{` depth tracking) at toml.go:247-282, `opensMultiLine` + cross-line `pendingMLS` at toml.go:294-349/36-50, `enterSkip` uses the same scans at toml.go:187-203. 13 pinned subtests (`TestReadTOMLSection_QuoteAware_*`) pass; verifier's 9-row production probe re-run: **9/9 no fabrication** (control + 8 CR-01 shapes, end-to-end through detectPython and detectRust into ProjectData) |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Plan-Level Must-Have Mapping (all restored)

| Plan | Must-have | Status |
|------|-----------|--------|
| 12-01 | T1 extract name/version/description from `[project]` | ✓ VERIFIED |
| 12-01 | T2 missing section → empty map, never error | ✓ VERIFIED |
| 12-01 | T3 legal-but-unsupported TOML never stores partial data | ✓ VERIFIED (RESTORED — CR-01 shapes now degrade; 6-quote/4-quote/bracket-in-string rows pin no-fabrication) |
| 12-01 | T4 global skip states — no parsing inside bodies, ANY section | ✓ VERIFIED (RESTORED — `FakeHeaderAfterDelimiterRun` + `CrossSectionSkip` rows: fake `[project]` header inside a `[build-system]` string body never switches the section) |
| 12-01 | T5 exact header equality, no prefix matching | ✓ VERIFIED |
| 12-01 | T6 CRLF/BOM, duplicate last-wins, garbage remainder | ✓ VERIFIED |
| 12-01 | T7 never panics on pathological input | ✓ VERIFIED (`TestReadTOMLSection_Adversarial` passes) |
| 12-02 | T1..T4, T6, T7 (Python PEP 621, poetry fallback, whole-section wins, dynamic → "", fallback chains, registry index 1) | ✓ VERIFIED |
| 12-02 | T5 malformed pyproject → folder-base Name + empty fields | ✓ VERIFIED (RESTORED — probe row: pyproject with only `name="evil"` inside a 6-quote body degrades to folder-base Name + empty Version; reader rows pin no-fabrication) |
| 12-03 | T1..T7 (Rust `[package]`, version.workspace → "", description.workspace → README fallback, array/boolean degrade, virtual manifest, index 4 + 5-of-7 + cascade, doc.go) | ✓ VERIFIED |

### Deferred Items

None. FRAM-02 (full TOML dependency) remains deferred to v2 by locked CONTEXT decisions — not a phase-12 gap. The 3 previously-failed must-haves were addressed by plan 12-04 within this phase; no gap is deferred to a later phase.

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | None — no new-scope concerns surfaced by the gap-closure round | other | Review carry-overs (WR-01, IN-01, IN-02, IN-04) are dispositioned as advisory in 12-REVIEW-DISPOSITION.md; none affect the closed gaps |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/toml.go` | readTOMLSection reader + strict degrade + global skip states, quote-aware (CR-01 fix) | ✓ VERIFIED | 349 lines, real state machine; `closesMultiLine`/`clearsBracket`/`opensMultiLine` helpers; `pendingMLS` cross-line state; stdlib-only (`bytes`/`strings`), no anti-features (`os/exec`, `net/http`, `EvalSymlinks`, `WalkDir`, `go-toml` greps clean) |
| `project_probe/toml_test.go` | Pure-content matrix incl. QuoteAware rows | ✓ VERIFIED | 468 lines, 34 tests/52 cases — 5 new `TestReadTOMLSection_QuoteAware_*` tests (13 subtests) pin all 8 CR-01 shapes + multi-line-string-opener-in-bracket-body family; all pass |
| `project_probe/detect_python.go` | detectPython | ✓ VERIFIED | 35 lines, readManifest → readTOMLSection, fallback chains (unchanged by 12-04) |
| `project_probe/detect_python_test.go` | Python fixture matrix | ✓ VERIFIED | 16 tests, all pass |
| `project_probe/detect_rust.go` | detectRust | ✓ VERIFIED | 35 lines, single `[package]` read (unchanged by 12-04) |
| `project_probe/detect_rust_test.go` | Rust fixture matrix | ✓ VERIFIED | 13 tests, all pass |
| `project_probe/registry.go` | Slots 1 and 4 live | ✓ VERIFIED | `detectPython`@1, `detectRust`@4; 5-of-7 live |
| `project_probe/registry_test.go` | TestDetectorPositions final pin | ✓ VERIFIED | NotNil 0/1/3/4/6, Nil 2/5 |
| `project_probe/doc.go` | Detector list refresh | ✓ VERIFIED | Names all five live detectors + Phase 13 slots |
| `project_probe/detect_php_test.go` | Cascade rows | ✓ VERIFIED | TestProbe_CascadePrecedence 5 rows |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| readTOMLSection skipDelim loop | closesMultiLine | `skipDelim != ""` case calls `closesMultiLine(line, skipDelim)`; clears only on net-closed (toml.go:30-34) | WIRED | 6-quote body lines (`""""""`, `''''''`) keep the skip state; plain `"""` clears it |
| readTOMLSection skipClose loop | clearsBracket + opensMultiLine + pendingMLS | `skipClose != 0` case: pendingMLS checked first (closesMultiLine net-closed clears), then opensMultiLine seeds pendingMLS, then clearsBracket at depth 0 clears (toml.go:35-54) | WIRED | `]`/`}` inside quoted strings never clear; `"""`/`'''` opener inside a bracket body keeps every later `]`/`}` inside the string |
| enterSkip | closesMultiLine/clearsBracket/opensMultiLine | same-line-close checks use the same scans (toml.go:187-203); 4-quote opening line enters the multi-line state; bracket-in-string opening line enters the bracket state; opening-line `"""` seeds pendingMLS (toml.go:64-70) | WIRED | Both CR-01 missed-entry paths eliminated |
| detectPython | readTOMLSection | `readManifest(folder,"pyproject.toml")` → `readTOMLSection(content,"project")` with len==0 fallback to `"tool.poetry"` | WIRED | detect_python.go:16-23; end-to-end probe row 8: `Name="acme"` preserved, no fabrication |
| detectRust | readTOMLSection | `readManifest(folder,"Cargo.toml")` → `readTOMLSection(content,"package")` | WIRED | detect_rust.go:19-23; end-to-end probe row 9: `Name="crate"` preserved, no fabrication |
| registry | detectors | `detectPython`@1, `detectRust`@4; runDetectors nil-skip + callDetector recover | WIRED | registry.go:14-22, 44-53 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detectPython | Name/Version/Description | readManifest bytes → readTOMLSection section map → ProjectData | Yes — real file content | ✓ FLOWING — no fabrication on any CR-01 shape (probe rows 1-8) |
| detectRust | Name/Version/Description | readManifest bytes → readTOMLSection `[package]` → ProjectData | Yes — real file content | ✓ FLOWING — no fabrication on any CR-01 shape (probe rows 1-7, 9) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| QuoteAware pinned matrix | `go test ./project_probe/... -run 'TestReadTOMLSection_QuoteAware'` | 13 passed | ✓ PASS |
| Full project_probe suite | `go test ./project_probe/...` | 185 passed (172 pre-existing + 13 new) | ✓ PASS |
| Repo-wide build | `go build ./...` | clean | ✓ PASS |
| Cross-platform vet | `GOOS=windows go vet ./project_probe/...` | clean | ✓ PASS |
| Coverage (project_probe pkg / toml.go file / total) | `make coverage-quick` | 93.1% / 87.7% / 79.3% — all thresholds pass; only known-red `release/update.go` 68.9% (documented, unrelated) | ✓ PASS |
| CR-01 probe row 1 (control) — simple multi-line string body | temp probe vs production `readTOMLSection` | `name="acme"` preserved | ✓ PASS |
| CR-01 probe row 2 — 6-quote line in `"""` body | temp probe | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 3 — 6-quote line in `'''` body | temp probe | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 4 — 4-quote opening line | temp probe | **no fabrication** — `map[]` | ✓ PASS |
| CR-01 probe row 5 — `]` inside quoted string in multi-line array | temp probe | **no fabrication** — `map[name:crate, version:1.2.3]` | ✓ PASS |
| CR-01 probe row 6 — `}` inside quoted string in inline table | temp probe | **no fabrication** — `map[name:acme]` | ✓ PASS |
| CR-01 probe row 7 — fake `[project]` header after 6-quote line in `[build-system]` | temp probe | **no section switch** — `map[]` | ✓ PASS |
| CR-01 probe row 8 — detectPython end-to-end (6-quote body line) | temp probe vs production `detectPython` | **`ProjectData.Name="acme"` — no fabrication** | ✓ PASS |
| CR-01 probe row 9 — detectRust end-to-end (`]` in quoted string) | temp probe vs production `detectRust` | **`ProjectData.Name="crate"` — no fabrication** | ✓ PASS |
| 12-02 truth 5 — malformed pyproject (only `name="evil"` inside body) | temp probe vs production `detectPython` | **folder-base Name + empty Version — never "evil"** | ✓ PASS |
| Anti-feature greps | `grep -rnE 'os/exec\|net/http\|EvalSymlinks\|WalkDir\|go-toml' project_probe/toml.go` | nothing | ✓ PASS |

The temporary probe file was deleted after the run; `git status --porcelain project_probe/` is clean (no residue).

### Probe Execution

No phase-declared probes (`scripts/*/tests/probe-*.sh` — none exist for this phase). The 9-row CR-01 probe was re-run by the verifier directly against the fixed production code per the re-verification mandate (Step 7b/7c behavior), mirroring the previous report's rows 129-137: **9/9 no fabrication**, plus the 12-02 truth 5 folder-base degrade row.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| ROBT-03 | 12-01, 12-04 | Unexported section-aware TOML-subset reader; strict degrade-to-empty | ✓ SATISFIED | Reader exists with exact-header matching and quote-aware global skip states; never-partial-data guarantee restored — 13 pinned subtests + 9-row probe show zero fabrication across all CR-01 shapes |
| DETC-05 | 12-02 | Python detector — pyproject `[project]` + legacy `[tool.poetry]`, PEP 621 fields | ✓ SATISFIED | Well-formed paths verified (SC1/SC2); malformed-input degrade now yields folder-base Name + empty/fallback fields, never fabricated (probe row 8 + truth-5 row) |
| DETC-06 | 12-03 | Rust detector — Cargo.toml `[package]` fields | ✓ SATISFIED | Well-formed paths verified (SC3); malformed arrays/inline tables degrade with no fabrication (probe row 9) |
| FRAM-02 | — | Full TOML dep deferred to v2 | — | Correctly out of scope (deferred in CONTEXT) |

No orphaned requirements: all Phase 12 IDs (DETC-05, DETC-06, ROBT-03) are claimed by plans (12-01/12-02/12-03/12-04) and marked Complete in REQUIREMENTS.md (lines 22-23, 39, 87-88, 98); no REQUIREMENTS.md ID mapped to Phase 12 is unclaimed.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | None | — | The CR-01 blocker (skip-state clearing not quote-aware, previous report lines 159-161) is FIXED in commit `596d58b`; the QuoteAware matrix pins the fix. No TBD/FIXME/XXX/TODO/HACK markers in phase-modified files; no stub patterns; no anti-feature imports |

### Human Verification Required

None. The 3 previously-failed behavior-dependent truths (SC4, truth 4, truth 5 — skip-state transitions/cancellation invariants) now have behavioral evidence: the 13 pinned matrix subtests exercise each transition against production code and pass, and the verifier's 9-row production probe (including end-to-end detectPython/detectRust into ProjectData) shows 9/9 no-fabrication. No visual, real-time, or external-service behavior is at stake.

### Gaps Summary

None. All 3 gaps from the previous verification are closed by plan 12-04 (commit `596d58b`):

1. **SC4** — quote-aware skip-state scanning restored the strict-degrade contract. `closesMultiLine` (run-of-3 closes; run>3 close+reopen) replaces `strings.Contains(line, skipDelim)`; `clearsBracket` (in-string bracket immunity honoring `\` escapes, `[`/`{` depth tracking) replaces `strings.ContainsRune(line, skipClose)`; `opensMultiLine` + cross-line `pendingMLS` keep a `"""`/`'''` opener inside a bracket body from letting a later `]`/`}` clear prematurely; `enterSkip` uses the same scans (4-quote opening lines and bracket-in-string opening lines now ENTER their states). The verifier's 9-row production probe: control + 8 CR-01 shapes → **9/9 no fabrication**, end-to-end into `ProjectData.Name` through detectPython AND detectRust.
2. **12-01 truth 4** — global skip states hold in ANY section even with delimiter-run bodies: fake `[project]` header after a 6-quote line in `[build-system]` never switches the section (pinned by `FakeHeaderAfterDelimiterRun`, `CrossSectionSkip`; probe row 7).
3. **12-02 truth 5** — malformed pyproject.toml degrades to folder-base Name + empty/fallback fields (probe: only-`evil`-inside-body pyproject → folder-base Name, empty Version).

No regressions: full suite 185/185 (172 pre-existing + 13 new), `go build ./...` clean, `GOOS=windows go vet` clean, coverage thresholds pass (project_probe 93.1%, toml.go 87.7%, total 79.3%; only known-red `release/update.go` 68.9% unrelated). The under-skip residual (a construct closer sharing a line with the string's close is scanned on the next line) is the documented safe direction — degrades to empty, never fabricates (12-04 flagged-assumptions).

---

_Verified: 2026-09-29T22:45:00Z_
_Verifier: the agent (gsd-verifier)_