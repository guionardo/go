---
phase: 14
phase_name: "Semantics, Hardening, and Release Polish"
project: "go"
generated: "2026-09-29"
counts:
  decisions: 8
  lessons: 6
  patterns: 5
  surprises: 4
missing_artifacts:
  - "*-UAT.md"
---

# Phase 14 Learnings: Semantics, Hardening, and Release Polish

## Decisions

### Fuzz bodies call detectors directly, not Probe()
Fuzz targets exercise readManifest → decode → chains per manifest type, bypassing the registry cascade.

**Rationale:** detector-direct is ~4x faster and reaches per-manifest targets the cascade order hides (a malformed package.json would fall through to PHP before the fuzz body could exercise the JS path).
**Source:** 14-01-SUMMARY.md

### No detector-result assertions in fuzz bodies
Fuzz bodies never assert detector bools — no-panic is the only invariant.

**Rationale:** presence-match (TOML/XML) vs parse-success (JSON) semantics differ; a panic IS the failure signal. Asserting results would produce false crash reports.
**Source:** 14-01-SUMMARY.md

### Fuzz corpus mixes real-world and pinned-malformed shapes
Corpus seeds include composer.json without version, pyproject dynamic, Cargo version.workspace, old-style padded .csproj, pom parent version — plus truncated, BOM, whitespace-only, and the 12-CR-01 6-quote/bracket-in-string TOML shapes.

**Rationale:** not byte-copies of test fixtures — real-world variety keeps the corpus representative, pinned malformed shapes keep it adversarial.
**Source:** 14-01-SUMMARY.md

### README badge rule and HTML comment state machine
isBadgeLine uses TrimSpace-empty OR Trim(line,"!")-empty with the existing "!["-presence guard; multi-line comment state machine checks inComment FIRST so blank lines inside the block never return a partial paragraph.

**Rationale:** review-prescribed shapes (11 WR-01/02) — plain badges and multi-line comment preambles must not leak into Description.
**Source:** 14-02-SUMMARY.md

### XML trim is decode hygiene per field, not normalization
strings.TrimSpace applied per decoded field BEFORE chain resolution — whitespace-only elements no longer count as present, so Version→VersionPrefix, <version>→<parent><version>, and name→artifactId/folder-base chains fire (DATA-03).

**Rationale:** encoding/xml copies element text verbatim; trimming at decode time is hygiene on a verbatim value, never normalization.
**Source:** 14-02-SUMMARY.md

### Coverage closure uses module-path derivation + ENOTDIR triggers
TestCheckForUpdate_NoOptions derives owner/repo from debug.ReadBuildInfo Main.Path → url.Parse → words; error-path rows use ENOTDIR triggers and closed-server URLs instead of chmod.

**Rationale:** permission-based triggers break Windows CI; the no-options row covers the 13-statement module-derivation branch (68.9% → 82.4%).
**Source:** 14-03-SUMMARY.md

### Version semantics contract documented in doc.go
All six detector rules recorded: go.mod toolchain floor, pyproject dynamic → "", Cargo version.workspace → "", .csproj Version→VersionPrefix (never MSBuild 1.0.0), pom.xml parent inheritance (never Super POM 4.0.0), package.json/composer.json verbatim; placeholders raw.

**Rationale:** the single place downstream consumers read the DATA-03 contract; decode-hygiene named explicitly as NOT normalization.
**Source:** 14-04-SUMMARY.md

### Anti-feature audit recorded as evidence file with disclosed exceptions
14-AUDIT.md records five grep families + XML-entity family zero-match outputs, with the f.Stat() read-path nuance and TrimSpace decode-hygiene nuance stated explicitly.

**Rationale:** T-14-14 mitigation — the audit's evidence must be re-runnable; exceptions disclosed rather than hidden (badge-stripping strings.ReplaceAll disclosed after the WR-01 fix).
**Source:** 14-04-SUMMARY.md

---

## Lessons

### Fuzz corpus encoding is the #1 trap — raw manifests fail the build
`testdata/fuzz/FuzzX/` files must be `go test fuzz v1`-encoded (header + one Go-syntax value per line), NOT raw manifest bytes; backtick literals can't span lines, use \n escapes.

**Context:** verified experimentally — a single malformed corpus file fails the whole package build with MalformedCorpusError.
**Source:** 14-RESEARCH.md, 14-01-SUMMARY.md

### The coverage gate closed with ONE statement, not a rewrite
release/update.go at 68.9% (51/74 statements) needed just 52 — one no-options test covering the 13-statement module-derivation branch closed the gate, then seven error-path rows reached 95.9%.

**Context:** the known-red was pre-existing for five phases; a test-only fix (update.go untouched, byte-identical) resolved it.
**Source:** 14-03-SUMMARY.md

### Coverage arithmetic is not always reachable — document the residual
71/74 = 95.9%, not the research-predicted 74/74 = 100%: the three remaining blocks are module-derivation error sub-branches unreachable from a test binary (debug.ReadBuildInfo always returns the module's Main.Path).

**Context:** measured outcome differs from plan prediction; the gate (file:70) is green regardless — document the residual rather than forcing a production seam.
**Source:** 14-03-SUMMARY.md

### golangci-lint new-filter is inconsistent between dirty and clean trees
The `new`-filter's diff-base resolution behaves differently on a dirty working tree (~300 matches, whole v1.7 branch debt) vs clean committed state (0 issues).

**Context:** delta-zero must be measured on the clean committed state — what CI and the verifier consume — and the artifact documented explicitly.
**Source:** 14-04-SUMMARY.md, 14-04-SUMMARY.md Issues Encountered

### Multi-badge lines need ReplaceAll, not a single-strip
isBadgeLine's first fix only handled one plain badge per line; `![CI](a) ![Coverage](b)` left remainder "! !" and leaked raw markdown.

**Context:** found by code review post-fix (WR-01), verified empirically, fixed with strings.ReplaceAll + 2 matrix rows.
**Source:** 14-REVIEW.md

### require.Equal in httptest handler goroutines is forbidden
FailNow/Goexit on a non-test goroutine produces a confusing network error instead of the assertion message.

**Context:** pre-existing pattern at update_test.go:195 replicated by the new no-options row; fixed with t.Errorf in handler + non-2xx response.
**Source:** 14-REVIEW.md

---

## Patterns

### Review-prescribed fix shapes (fold-in discipline)
Each open review finding carries a prescribed fix shape from the ledger (badge Trim, comment block state, per-field TrimSpace, exact-name guard, empty-slice injection).

**When to use:** closing review dispositions in a hardening phase — use the review's prescribed shape, add pinning matrix rows, verify empirically.
**Source:** 14-02-PLAN.md, 14-02-SUMMARY.md

### Fuzz as test-mode hardening (no CI -fuzz flag)
Fuzz targets run their seed corpus under plain `go test` (CI-safe on all 3 OSes); t.TempDir works in fuzz bodies; -fuzz never goes in CI.

**When to use:** adding fuzz coverage to a library — seed-mode execution under normal test is the CI-safe pattern.
**Source:** 14-01-PLAN.md, 14-RESEARCH.md

### ENOTDIR triggers for cross-platform error-path tests
File-as-parent and slash-in-name ENOTDIR triggers plus closed-server URLs replace chmod/permission triggers that break Windows CI.

**When to use:** any error-path test that must run on Linux/macOS/Windows — never chmod.
**Source:** 14-03-PLAN.md, 14-03-SUMMARY.md

### Evidence-file audit with disclosed nuances (T-14-14 mitigation)
The anti-feature audit lives in 14-AUDIT.md as a re-runnable evidence file — commands verbatim, zero-match outputs recorded, nuances (f.Stat read path, TrimSpace decode hygiene) stated explicitly.

**When to use:** any "no X anywhere" guarantee — record the greps and their outputs so the audit is provable and exceptions are visible, not hidden.
**Source:** 14-04-SUMMARY.md

### Clean-committed-state measurement for lint gates
Delta-zero lint gates must run on the clean committed state (what CI/verifier consume), not the dirty working tree where diff-base resolution skews results.

**When to use:** any delta-based lint or diff gate — pin the measurement state explicitly.
**Source:** 14-04-SUMMARY.md

---

## Surprises

### The coverage-gate known-red was the ONLY repo-wide failure — and closed in one plan
The 68.9% release/update.go row was pre-existing since Phase 10 and the sole gate failure; a test-only plan closed it in minutes.

**Impact:** the long-standing "known-red exception" in every verification section disappeared — make coverage-quick now passes outright.
**Source:** 14-03-SUMMARY.md

### The audit's own evidence went stale within the same phase
The WR-01 fix (strings.ReplaceAll badge stripping) landed AFTER the audit commit, making the family-5a "zero matches" row non-reproducible — the verifier caught it as a documentation-evidence gap.

**Impact:** audits must be re-run or their exceptions disclosed before verification; this cost a re-verification cycle and a user-accepted override.
**Source:** 14-VERIFICATION.md, 14-REVIEW.md

### Lint baseline of 28 pre-existing issues never blocked anything
The v1.7 branch carried 28 lint issues from phases 10-13 (plus ~300 in dirty-tree new-filter artifacts), yet delta-zero on phase edits was the operative contract.

**Impact:** "lint clean" needed a defined measurement (delta-zero on touched files, clean state) rather than an absolute zero.
**Source:** 14-04-SUMMARY.md

### Fuzz bodies calling detectors directly was ~4x faster than Probe()
The intended Probe()-based fuzz path was slower and hid per-manifest targets behind the cascade; detector-direct calls are the efficient shape.

**Impact:** fuzz targets reach each manifest type's parse path directly, making the no-panic invariant more targeted.
**Source:** 14-01-SUMMARY.md