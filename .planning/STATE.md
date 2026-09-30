---
gsd_state_version: "1.0"
milestone: v1.7
milestone_name: Project Probe
status: Awaiting next milestone
stopped_at: Phase 14 complete — all phases complete
last_updated: "2026-09-30T23:19:07.437Z"
last_activity: 2026-09-30
last_activity_desc: Milestone v1.7 completed and archived
state_head: 2b8b56b1dd9efa9a858cb0ded60d9a1773d994f2
progress:
  total_phases: 5
  completed_phases: 12
  total_plans: 16
  completed_plans: 16
  percent: 100
current_phase: 14
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-30)

**Core value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.
**Current focus:** Awaiting next milestone

## Current Position

Phase: Milestone v1.7 complete
Plan: —
Status: Awaiting next milestone
Last activity: 2026-09-30 — Milestone v1.7 completed and archived

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
| Phase 12-toml-subset-python-rust-detectors P01 | 7min | 2 tasks | 2 files |
| Phase 12-toml-subset-python-rust-detectors P02 | 6min | 2 tasks | 4 files |
| Phase 12-toml-subset-python-rust-detectors P03 | 8min | 2 tasks | 6 files |
| Phase 12-toml-subset-python-rust-detectors P04 | 37min | 2 tasks | 2 files |
| Phase 13 P01 | 7min | 2 tasks | 4 files |
| Phase 13 P02 | 9min | 2 tasks | 6 files |
| Phase 14 P01 | 6min | 2 tasks | 21 files |
| Phase 14 P02 | 6min | 3 tasks | 7 files |
| Phase 14-semantics-hardening-release-polish P03 | 8min | 2 tasks | 1 files |
| Phase 14 P04 | 34 | 3 tasks | 3 files |

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
- [Phase 12-toml-subset-python-rust-detectors]: Global skip states (multi-line string/array/inline table) are entered from keyval lines in ANY section, not just the target section — the RESEARCH skeleton's section-gated parse order applies to storage only, never to skip detection (D-disc-4/A7; pinned by TestReadTOMLSection_CrossSectionSkip)
- [Phase 12-toml-subset-python-rust-detectors]: Skip-state entry is independent of the key being a target name: authors = [ enters the array skip state even though authors is never stored (P4)
- [Phase 12-toml-subset-python-rust-detectors]: BOM tolerance in the reader reuses manifest.go's utf8BOM via bytes.TrimPrefix — single source of truth, Phase 10 precedent; reader stays tolerant even though readManifest strips upstream
- [Phase 12-toml-subset-python-rust-detectors]: Whole-section precedence executed per D-disc-3 (flagged): [project] non-empty map wins; [tool.poetry] read ONLY when len(fields)==0 — no per-field mixing across sections; pinned by TestProbe_PythonProjectWins + TestProbe_PythonEmptyProjectFallsToPoetry — D-04's 'when [project] absent' is the contract; per-field mixing would fabricate hybrid metadata in migrated Poetry 2.x files
- [Phase 12-toml-subset-python-rust-detectors]: Presence-match asymmetry (D-09) shipped: garbage pyproject.toml claims Python at index 1 while JS needs parse success at index 3; cascade consequence pinned in plan 12-03's TestProbe_CascadePrecedence — The manifest file IS the ecosystem marker; fields degrade to empty with fallbacks, never fabricated (T-12-04 accept)
- [Phase 12-toml-subset-python-rust-detectors]: dynamic = ["version"] has NO code branch in the detector — the reader degrades the array to an absent key, Version reads "" (DATA-03 never-fabricated); pinned by TestProbe_PythonDynamicVersion — A dynamic-field special case is the anti-pattern RESEARCH calls out; strict degrade is structural
- [Phase 12-toml-subset-python-rust-detectors]: Interim registry flip only (Pitfall 9): TestDetectorPositions changes exactly one assertion (detectors[1] NotNil); detectors[4] stays Nil for plan 12-03's RED step; the flip is the ONLY registry_test.go change per the plan prohibition — Flipping both in 12-02 breaks plan 12-03's RED step; the 7-position order contract is guarded at every commit
- [Phase 12-toml-subset-python-rust-detectors]: D-07 executed structurally: version.workspace = true degrades to Version "" with ZERO detector code branches — the reader's dotted-key classification (isBareKey rejects '.') is the only mechanism; no workspace-root resolution, no special-casing; pinned by TestProbe_RustWorkspaceVersion
- [Phase 12-toml-subset-python-rust-detectors]: Virtual manifest (OQ-2 flagged): [workspace]-only Cargo.toml matches on presence (D-09) with folder-base Name and "" Version — natural consequence of the locked decisions; pinned by TestProbe_RustVirtualManifest
- [Phase 12-toml-subset-python-rust-detectors]: Presence-match asymmetry shipped at cascade level: a garbage pyproject.toml claims Python at index 1 over a valid Cargo.toml at index 4 (T-12-08 accept — the manifest file IS the ecosystem marker); pinned by the pyproject_toml_and_cargo_toml cascade row
- [Phase 12-toml-subset-python-rust-detectors]: registry_test.go final flip ONLY (plan prohibition): exactly one assertion changed (detectors[4] Nil→NotNil) plus gofmt alignment of the 7-slot literal in registry.go (the literal was already non-gofmt at HEAD from prior edits; formatting the file I modified is the clean end state); no parallel marker added
- [Phase 12-toml-subset-python-rust-detectors]: Skip-state clearing is quote-aware (CR-01 fix): closesMultiLine (run of 3 closes, run > 3 closes+reopens → state persists), clearsBracket (]/} inside quoted strings honoring escapes never clear; [/{ depth tracking), opensMultiLine + cross-line pendingMLS for multi-line strings opened inside bracket bodies
- [Phase 12-toml-subset-python-rust-detectors]: enterSkip uses the same quote-aware scans: 4-quote opening lines ("""a"""") and bracket-in-string opening lines (["A ] B",) now ENTER the skip state — closing both CR-01 missed-entry fabrication paths
- [Phase 12-toml-subset-python-rust-detectors]: CR-01 fabrication gap closed by plan 12-04 (gap closure): verifier's 9-row adversarial probe re-run shows 9/9 no-fabrication; 5 quote-aware matrix functions (13 subtests) pin the fix
- [Phase 14]: Fuzz bodies call detectors directly (D-05) — readManifest → decode → chains — not Probe(): detector-direct is ~4x faster and reaches per-manifest targets the cascade order hides
- [Phase 14]: No detector-result assertions in f.Fuzz bodies (Pitfall 3): presence-match (TOML/XML) vs parse-success (JSON) semantics differ; a panic IS the failure signal
- [Phase 14]: Corpus mixes real-world shapes (composer.json without version, pyproject dynamic, Cargo version.workspace, old-style padded .csproj, pom parent version) with pinned malformed shapes (truncated, BOM, whitespace-only, 12 CR-01 6-quote/bracket-in-string) — not byte-copies of test fixtures (Pitfall 6)
- [Phase 14]: FuzzXMLManifest writes 'MyApp.csproj', never bare '.csproj' — the 13 IN-03 exact-name guard makes a bare suffix file non-matchable
- [Phase 14]: readme.go badge rule: TrimSpace-empty OR Trim(line, "!")-empty with the existing '!['-presence guard; multi-line comment state machine with inComment FIRST (blank lines inside the block never return a partial paragraph)
- [Phase 14]: XML trim is decode hygiene per field BEFORE chain resolution — whitespace-only elements no longer count as present, so Version→VersionPrefix, <version>→<parent><version>, name→artifactId/folder-base chains fire (DATA-03)
- [Phase 14]: exact-name guard len(e.Name()) <= len(suffix) precedes HasSuffix — a file literally named '.csproj' can never match (13 IN-03)
- [Phase 14]: TestRunDetectors_EmptyRegistry injects []detectorFunc{} and probes the non-colliding folder 'x' (12/13 WR-02)
- [Phase 14]: The mandatory no-options row derives owner/repo from the module path: CheckForUpdate(ctx, "v1.0.0") with NO options enters getCurrentModule → debug.ReadBuildInfo Main.Path "github.com/guionardo/go" → url.Parse (scheme-less → Path) → words[1]=guionardo, words[2]=go; the mock handler's require.Equal on /repos/guionardo/go/releases/latest pins the derivation (T-14-12 mitigation)
- [Phase 14]: Seven error-path rows pin error-not-panic on every failure branch of CheckForUpdate (91-92 request creation, 102-103 network, 112-113 decode, 117-118 version) and DownloadUpdate (130-131 MkdirAll, 136-137 Create, 141-144 digest mismatch with os.Remove cleanup)
- [Phase 14]: ENOTDIR triggers (file-as-parent, slash-in-matched-name) + closed-server URLs replace permission-based triggers — cross-platform on all three CI OSes, no chmod anywhere (T-14-11 mitigation)
- [Phase 14]: Coverage arithmetic adjusted from research: 71/74 = 95.9% not 74/74 = 100% — the three remaining blocks (62-63, 67-68, 72-73) are module-derivation error sub-branches unreachable from a test binary (debug.ReadBuildInfo always returns Main.Path); gate (file:70) green regardless
- [Phase 14]: Version semantics contract documented in doc.go with all six detector rules: go.mod toolchain floor, pyproject dynamic → '', Cargo version.workspace → '', .csproj Version→VersionPrefix (never MSBuild 1.0.0), pom.xml parent inheritance (never Super POM 4.0.0), package.json/composer.json verbatim; placeholders raw; XML TrimSpace named as decode hygiene, not normalization (D-02/DATA-03)
- [Phase 14]: Anti-feature audit recorded as evidence file (14-AUDIT.md): five grep families + XML-entity family all zero matches; f.Stat() read-path nuance and TrimSpace decode-hygiene nuance stated (D-03/ROBT-05)
- [Phase 14]: Deferred dispositions ledger recorded with rationale: Phase 10 CR-01 (Windows errno — needs Windows CI evidence, tracked) and WR-02 (panic logging — future milestone); residual INFO findings (11 IN-01/IN-02, 13 IN-02) accepted not-in-scope, recorded not dropped (D-08)
- [Phase 14]: Lint delta-zero verified on the clean committed state: full run 0 issues, Phase-14 delta (8461196..HEAD) 0 issues; the dirty working tree's ~300 pre-existing issues documented as a new-filter diff-base artifact, advisory per A3 (D-10)

### Pending Todos

None yet.

### Blockers/Concerns

- [Planning]: REQUIREMENTS.md claimed 18 v1 requirements; actual count is 21 (3+9+4+5) — traceability updated
- [Deferred — tracked]: Phase 10 CR-01 — D-04 errno mapping needs Windows verification (syscall.EACCES invented on Windows) — deferred with rationale in 14-AUDIT.md (needs Windows CI evidence)
- [Deferred — tracked]: Phase 10 WR-02 — panic logging at registry dispatch — deferred with rationale in 14-AUDIT.md (dev-experience nicety, not correctness)
- [Resolved]: Phase 10/11/13 code review advisories — ALL fold-in fixed in Phase 14 (badge multi-line, HTML comments, XML trim, exact-name guard, registry test hygiene) or disclosed with override (badge-stripping grep exception in 14-AUDIT.md)
- [Resolved]: `make coverage-quick` pre-existing failure — CLOSED 2026-09-29 by Phase 14 (release/update.go 95.9%, total 80.7% PASS)
- [Resolved]: Phase 12 TOML strict-degrade — DISCHARGED by plan 12-01 matrix
- [Resolved]: Phase 13 root-scoped marker decision — CONFIRMED at Phase 13 planning
- [Non-blocking]: verification-debt SUMMARY metadata warnings (files not on disk) — tracked in /gsd-progress /gsd-audit-uat

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-09-29T11:19:32.049Z
Stopped at: Phase 14 complete — all phases complete
Resume file: None

## Operator Next Steps

- Start the next milestone with /gsd-new-milestone
