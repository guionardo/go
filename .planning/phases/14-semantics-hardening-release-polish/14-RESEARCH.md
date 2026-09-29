# Phase 14: Semantics, Hardening, and Release Polish - Research

**Researched:** 2026-09-29
**Domain:** Go fuzz testing (seed-corpus mechanics), coverage-gate closure, version-semantics documentation, anti-feature audit, README/doc.go polish
**Confidence:** HIGH

## Summary

Phase 14 is the final v1.7 phase: it proves and documents the version-semantics contract (DATA-03), proves the anti-feature absence (ROBT-05), adds fuzz targets that never panic on malformed manifests (SC3), folds in the open review dispositions, and closes the last coverage-gate failure so the milestone ships green. Nothing in this phase needs a new dependency, a new runtime, or new external services — every requirement is met with stdlib, one new test file, seed-corpus fixture files, and surgical edits to existing files.

The phase's one genuinely new mechanism is **Go fuzzing**, and the single most dangerous trap is the seed-corpus file format: `testdata/fuzz/FuzzX/` files are **not** raw manifest bytes — they are encoded in `go test fuzz v1` format (a header line plus one Go-syntax value per line). This was verified live this session on the local toolchain (go1.27.0): raw bytes in a corpus file fail the whole package build with `unmarshal: must include version and at least one value`; correctly encoded files run under plain `go test` (seed mode), under `-run=FuzzX`, and under `-fuzz=FuzzX` (5800 execs in 3s with a disk-writing fuzz body — `t.TempDir()` works in both modes). CI runs `go test -v ./...` on three OSes, so fuzz seeds execute on every CI run — exactly SC3's "running under normal go test".

The coverage gate is provably closeable with a tiny test addition: `release/update.go` has **74 statements, 51 covered = 68.9%**; 70% needs only **52 covered statements** (verified from the coverage profile). A single new test — `CheckForUpdate` called with **no options** — enters the module-path derivation branch and covers 13 statements (86.5% file), because Go's cover instrumentation counts a block as covered when its entry executes, not per-branch. A full 7-row set covers 100% of the file; the no-options row alone closes the gate. The anti-feature audit is already provably clean: greps for `os/exec`, `net/http`, `EvalSymlinks`/`Readlink`, `WalkDir`, and version-normalization calls (`ParseVersion`, `semver`, `regexp`) return **zero matches** across `project_probe/` (verified this session). Version semantics are likewise raw-string across all 7 detectors by construction — every Version assignment is a verbatim manifest value or `""`, and the two structural degrades (Cargo `version.workspace`, pyproject `dynamic`) have no code branches at all (Phase 12 pinned both by tests).

**Primary recommendation:** four plans mirroring prior-phase granularity — **P01 = fuzz targets + encoded seed corpus** (new `fuzz_test.go` with `FuzzJSONManifest`/`FuzzTOMLManifest`/`FuzzXMLManifest` calling the detectors directly, plus `testdata/fuzz/Fuzz<Name>/` encoded real-manifest seeds); **P02 = correctness fold-in** (readme.go WR-01 plain badge + WR-02 multi-line HTML comments with matrix rows, XML `strings.TrimSpace` per field in both XML detectors with rows, `.csproj` exact-name guard 13 IN-03, decode-comment rewording 13 IN-01, registry test hygiene 12/13 WR-02); **P03 = release/update.go coverage closure** (no-options test mandatory, six error-path rows recommended); **P04 = docs + audit + gate** (doc.go Version-semantics contract section D-02, README row + section D-10, anti-feature grep audit recorded in the verification report D-03, `GOTOOLCHAIN=go1.26.4 golangci-lint run` delta-zero, `make coverage-quick` green, milestone complete).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Version Semantics (DATA-03)
- **D-01:** Package-wide audit confirms Version is always the raw manifest string across all 7 detectors: go.mod `go` directive (toolchain floor, documented), Maven `<parent><version>` inheritance, Cargo `version.workspace` → empty, pyproject `dynamic` → empty. Never normalized, never fabricated.
- **D-02:** doc.go gains a final Version-semantics contract section documenting these rules (the single place downstream consumers read them).

#### Anti-Feature Audit (ROBT-05)
- **D-03:** Systematic grep-verified audit across all 7 detectors + reader helpers: no os/exec (build-tool execution), no net/http (network), no EvalSymlinks/WalkDir (symlink following), no version normalization in any code path. Results documented in the verification report.

#### Fuzz Targets (SC3)
- **D-04:** 3 Fuzz functions — FuzzJSONManifest (package.json/composer.json), FuzzTOMLManifest (pyproject/Cargo), FuzzXMLManifest (.csproj/pom.xml) — seeded with real manifest files as corpus, running under normal `go test` (fuzz functions execute seed corpus in test mode). Malformed inputs never panic, never crash the probe.
- **D-05:** Fuzz bodies exercise the detectors' parse paths (readManifest → decode → chains), not just the raw readers.

#### Review Dispositions Fold-In
- **D-06:** Fix the open advisory findings that affect correctness: readme.go plain-badge form `![alt](url)` (11 WR-01), multi-line HTML comment preambles (11 WR-02), XML element text trimming (13 WR-01 — strings.TrimSpace per field before chain resolution).
- **D-07:** Fix test hygiene: TestRunDetectors_EmptyRegistry injects an explicit empty slice (12/13 WR-02), stale comments (IN-01), `.csproj` exact-name guard (13 IN-03).
- **D-08:** Deferred dispositions recorded explicitly with rationale: Phase 10 CR-01 (Windows errno mapping — requires Windows CI evidence, tracked), panic logging (10 WR-02) — noted for future milestone, not silently dropped.

#### Coverage Gate (SC4)
- **D-09:** `make coverage-quick` must pass — add missing `release/update.go` tests (the pre-existing 68.9% < 70% file-threshold known-red is the ONLY repo-wide gate failure; SC4 explicitly requires the gate to pass; project_probe rows already pass). Small test addition in the release package closes it.

#### Release Polish
- **D-10:** README.md gains the project_probe package index row (structure precedent: table of packages); doc.go never-fail contract finalized; go vet + golangci-lint clean (GOTOOLCHAIN=go1.26.4 workaround); milestone v1.7 complete.

### the agent's Discretion
- Fuzz corpus layout (testdata/fuzz dirs), exact test additions for release/update.go, README row wording. Follow repo conventions (CONVENTIONS.md) and prior-phase patterns.

### Deferred Ideas (OUT OF SCOPE)
- Framework detection (FRAM-01), full TOML dep (FRAM-02), typescript value (REFN-01), kotlin value (REFN-02), lockfile confirmation (REFN-03), multi-csproj rule (REFN-04) — v2 backlog
- Phase 10 CR-01 (Windows errno verification) + WR-02 (panic logging) — deferred with rationale (D-08), tracked
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DATA-03 | Version: raw manifest string, empty when absent/dynamic, never fabricated | All 7 Version assignments read this session and quoted verbatim in Pattern 7 — raw values only [VERIFIED: project_probe/detect_go.go:26,63; detect_javascript.go:34; detect_php.go:33; detect_python.go:29; detect_rust.go:29; detect_csharp.go:103-106; detect_java_kotlin.go:104-107]; structural degrades (`version.workspace`, `dynamic`) have zero code branches [VERIFIED: project_probe/toml.go reader classification + Phase 12 STATE.md pins]; no `ParseVersion`/`semver`/`regexp` anywhere in project_probe [VERIFIED: grep this session — zero matches]; doc.go contract section shape in Pattern 7 (D-02) |
| ROBT-05 | No build-tool execution, no network, no symlink following, no version normalization (anti-features enforced) | Grep-verified audit commands in Pattern 8 — all five greps return zero matches across project_probe/ this session [VERIFIED: `grep -rnE "os/exec"` → NONE, `net/http` → NONE, `EvalSymlinks` → NONE, `WalkDir` → NONE, `ParseVersion|semver|regexp` → NONE]; the XML `strings.TrimSpace` fold-in (13 WR-01) is decode hygiene, NOT normalization — the doc.go contract must say so explicitly (Pitfall 2); audit results land in the phase verification report (D-03) |

## Project Constraints (from AGENTS.md)

Directives extracted from `./AGENTS.md` — the planner must verify every plan complies:

- **Milestone branches:** every milestone gets its own git branch for the PR flow; branch `gsd/v{VERSION}-{slug}`, PR to `main` when complete (current branch: `gsd/v1.7-project-probe`).
- **Before every commit:** invoke `Skill("spike-findings-go")`; run `make coverage-quick` and do not commit if it fails (thresholds: packages ≥80%, files ≥70%, total ≥75%; cache providers have overrides). **This gate is the D-09 target — it fails today ONLY on `release/update.go` 68.9%.**
- **Before pushing a tag (not every commit):** `make quality-report` + `git add quality-report.md`.
- **Cross-platform CI (Linux/macOS/Windows):** `go test -v ./...` runs on all three — the fuzz seed corpus runs there (SC3). Windows: `http.DefaultClient` header canonicalization (existing release tests already handle); `os.LookupEnv` case-insensitive; file permissions (never write chmod-dependent tests — the recommended release error-path tests use ENOTDIR triggers, not chmod); tests mutating global state (registry `detectors`, release `githubAPIBase`) must not use `t.Parallel()`.
- **Code style:** standard Go conventions, focused functions, `testify` assertions, doc comments on all exported symbols, Conventional Commits, minimal external dependencies (stdlib preferred — this phase adds zero new deps).

## Architectural Responsibility Map

Single-package stdlib library — no browser/SSR/CDN/database tiers exist. Every capability belongs to the package core or the repo tooling layer; the map prevents the planner from pushing fuzz targets, audits, or docs into the wrong layer.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Fuzz targets (SC3) | API/Backend (package core) | — | `project_probe/fuzz_test.go` — `package projectprobe` (internal) so unexported detectors are reachable; fuzz bodies call `detectJS`/`detectPHP`/`detectPython`/`detectRust`/`detectCSharp`/`detectJavaKotlin` directly (D-05: readManifest → decode → chains) |
| Fuzz seed corpus | Repo fixtures (testdata) | — | `project_probe/testdata/fuzz/FuzzJSONManifest|FuzzTOMLManifest|FuzzXMLManifest/` — encoded `go test fuzz v1` files, real manifests + malformed shapes; NOT raw bytes (Pitfall 1 — verified) |
| Version-semantics audit (DATA-03) | API/Backend (package core) | Docs | Code inspection of the 7 detectors (Pattern 7) + grep evidence; the contract lands in `doc.go` (D-02) — the single downstream-readable place |
| Anti-feature audit (ROBT-05) | Package core (code paths) | Verification report | Grep commands in Pattern 8; results documented in the phase verification report (D-03), not in code |
| Coverage-gate closure (D-09) | Repo tooling | — | `release/update_test.go` — new tests for the 23 uncovered statements of `update.go`; no production-code change to `update.go` is needed or wanted |
| readme.go fold-in (11 WR-01/WR-02) | Package core | — | `project_probe/readme.go` `firstRealParagraph` — badge remainder rule + comment-block state (Pattern 4) |
| XML trim + guards (13 WR-01/IN-03) | Package core | — | `detect_csharp.go`/`detect_java_kotlin.go` — `strings.TrimSpace` per decoded field before chain resolution; `readFirstManifest` exact-name guard (Pattern 5) |
| Registry test hygiene (12/13 WR-02) | Tests | — | `registry_test.go` `TestRunDetectors_EmptyRegistry` — inject an explicit empty slice (Pattern 6) |
| Release docs (D-10) | Docs | — | `README.md` package index row + `### Package project_probe` section (Pattern 9); `doc.go` Version-semantics contract (D-02) |
| Lint/vet cleanliness (D-10) | Repo tooling | — | `go vet` green today [VERIFIED]; `GOTOOLCHAIN=go1.26.4 golangci-lint run` — 28 pre-existing new-vs-main issues in project_probe/release (advisory baseline, phases 10-13); Phase 14's edits must add ZERO new issues |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `testing` fuzz support | 1.26.4 (go.mod; local 1.27.0) | `FuzzJSONManifest`/`FuzzTOMLManifest`/`FuzzXMLManifest` + seed-corpus execution under `go test` | D-04 locked; behavior verified live this session (seed mode, `-run=FuzzX`, `-fuzz=FuzzX`, TempDir-in-body, corpus format) |
| Go stdlib `strings`/`bytes` | 1.26.4 | TrimSpace fold-in (13 WR-01), readme.go badge/comment fixes | Already imported in the touched files |
| Go stdlib `encoding/xml`, `encoding/json`, `os`, `path/filepath` | 1.26.4 | Unchanged detector paths the fuzz bodies exercise | Existing (phases 11-13), no changes |
| `github.com/stretchr/testify` | v1.11.1 (test-only, existing) | `assert`/`require` in all new tests | AGENTS.md mandate; already in go.mod [VERIFIED: go.mod] |
| `github.com/hashicorp/go-version` | v1.9.0 (existing) | `ParseVersion` inside `CheckForUpdate` — exercised by the new release tests | Existing dep [VERIFIED: go.mod]; the phase adds no new code using it |
| `github.com/opencontainers/go-digest` | v1.0.0 (existing) | `Asset.Download` digest verification — exercised by the digest-mismatch row | Existing dep [VERIFIED: go.mod] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go-test-coverage` v2 | installed | `make coverage-quick` gate — the D-09 verification command | Local + the phase gate; installed via `make install-go-test-coverage` [VERIFIED: runs on this machine] |
| `golangci-lint` | installed | D-10 lint check — MUST run with `GOTOOLCHAIN=go1.26.4` | Verification only; 28 pre-existing issues are advisory baseline (see Security/Assumptions) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Detector-direct fuzz bodies (D-05) | `Probe(folder)` in the fuzz body | Probe runs the full cascade (up to 7 reads/iteration) and the cascade order makes per-manifest targets unreachable (package.json at index 3 always beats composer.json at 6; garbage package.json needed for PHP to fire) — detector-direct calls exercise exactly readManifest → decode → chains per D-05 and are ~4× faster (measured 1933 execs/s with disk writes) |
| Encoded seed corpus (`go test fuzz v1` files) | Raw manifest bytes in `testdata/fuzz/` | Raw bytes FAIL the build — the format is mandated by `internal/fuzz` (verified; the whole package fails with `must include version and at least one value`) |
| `f.Add()` seeds only | `f.Add()` + corpus files | D-04 mandates "seeded with real manifest files as corpus" — corpus files are the durable, reviewable fixture; `f.Add` seeds are the inline complement (both run under `go test`) |
| New helper `readXMLManifest` for the trim fix | Per-field `strings.TrimSpace` in the two detectors | Phase 13 D-disc-8 precedent: an uncalled unexported helper fails the file:70 coverage gate; the trim is 5 assignments, not a helper |
| Fixing all 28 pre-existing lint issues | Delta-zero on lint | The 28 are the whole v1.7 branch's new-vs-main debt (phases 10-13), unrelated to Phase 14; fixing them is scope creep. D-10 "lint clean" = Phase 14's own edits add zero issues (pre-commit `gofmt`/`goimports` already enforced on touched files) |

**Installation:** none — stdlib only; testify/go-version/go-digest already in go.mod [VERIFIED: go.mod]. The planner must not add any `go get` task.
**Version verification:** `go.mod` declares `go 1.26.4` [VERIFIED: go.mod]; local toolchain go1.27.0 (darwin/arm64); `GOTOOLCHAIN=go1.26.4 go version` → go1.26.4 [VERIFIED this session]; CI pins via `go-version-file: go.mod` [VERIFIED: .github/workflows/go.yml:24].

## Package Legitimacy Audit

> Gate outcome: **N/A — no external packages installed by this phase.** All runtime and test code is Go stdlib; `testify`, `hashicorp/go-version`, and `opencontainers/go-digest` are established module dependencies used test-only or pre-existing [VERIFIED: go.mod]. The planner must not add any `go get` task — same disposition as Phases 11/12/13 (go ecosystem not covered by the package-legitimacy seam; gate satisfied by zero installs).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|----------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/stretchr/testify` v1.11.1 | Go modules | ~10 yrs | widely used | stretchr/testify | OK (existing dep) | Approved — no install action |
| `github.com/hashicorp/go-version` v1.9.0 | Go modules | ~10 yrs | widely used | hashicorp/go-version | OK (existing dep) | Approved — no install action |
| `github.com/opencontainers/go-digest` v1.0.0 | Go modules | ~8 yrs | widely used | opencontainers/go-digest | OK (existing dep) | Approved — no install action |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────────────────────────────────────────┐
                    │           Phase 14 deliverables                   │
                    └─────────────────────────┬────────────────────────┘
                                              │
        ┌─────────────────┬───────────────────┼───────────────────┬──────────────────┐
        ▼                 ▼                   ▼                   ▼                  ▼
  ┌────────────┐   ┌──────────────┐   ┌───────────────┐   ┌──────────────┐   ┌──────────────┐
  │ Fuzz targets│   │ Fold-in fixes │   │ Coverage gate │   │ docs.go +     │   │ Anti-feature │
  │ (D-04/D-05) │   │ (D-06/D-07)  │   │ (D-09)        │   │ README (D-10) │   │ audit (D-03) │
  └──────┬─────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘
         │                │                  │                  │                  │
         ▼                ▼                  ▼                  ▼                  ▼
  fuzz_test.go      readme.go (WR-01/02)  update_test.go    doc.go §Version    grep -rnE
  ┌────────────┐    detect_csharp.go       ┌──────────┐     semantics (D-02)  "os/exec|net/http|
  │ FuzzJSON   │    detect_java_kotlin.go  │ no-options│    README.md row     EvalSymlinks|WalkDir|
  │ FuzzTOML   │    registry_test.go       │ + 6 error │    + §Package        ParseVersion|semver|
  │ FuzzXML    │    (13 WR-01/IN-03,       │  rows     │    project_probe      regexp" → 0 hits
  └──────┬─────┘    12/13 WR-02, IN-01)    └──────────┘    └──────┬───────┘   └──────┬───────┘
         │                                                        │                  │
         ▼                                                        ▼                  ▼
  testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/         make coverage-quick       verification
  └─ go test fuzz v1 encoded seeds ─┐                (74 stmts; +1 covered      report
                                    │                 closes 70% file gate)     (evidence)
                                    ▼
                    CI: go test -v ./... (3 OSes) — seeds run, no -fuzz
```

Entry point: none new — this phase adds no production API. Processing: fuzz seeds → `Fuzz<X>` bodies → detector functions (`detectJS` etc. → `readManifest` → decode → chains) with no-panic as the invariant; the coverage row closes `CheckForUpdate`/`DownloadUpdate` error paths; docs/audit are read-only evidence. External dependencies: filesystem only (TempDir fixtures); no network, no services.

### Recommended Project Structure

```
project_probe/
├── fuzz_test.go                 # NEW: FuzzJSONManifest + FuzzTOMLManifest + FuzzXMLManifest
│                                #   (package projectprobe — unexported detectors reachable)
├── testdata/fuzz/FuzzJSONManifest/   # NEW: encoded seeds — real package.json, composer.json,
│                                #   malformed/BOM/truncated/duplicate-key variants
├── testdata/fuzz/FuzzTOMLManifest/   # NEW: encoded seeds — real pyproject.toml, Cargo.toml,
│                                #   CR-01 adversarial shapes (6-quote, 4-quote, bracket-in-string),
│                                #   dynamic/version.workspace variants
├── testdata/fuzz/FuzzXMLManifest/    # NEW: encoded seeds — real .csproj (SDK + old-style),
│                                #   pom.xml (parent-version), pretty-printed, whitespace-only,
│                                #   truncated-after-complete-group (13 IN-01 shape)
├── readme.go                    # MOD: isBadgeLine plain-badge rule (11 WR-01); firstRealParagraph
│                                #   comment-block state (11 WR-02); test rows in readme_test.go
├── detect_csharp.go             # MOD: strings.TrimSpace per field in the collect loop (13 WR-01);
│                                #   readFirstManifest exact-name guard (13 IN-03); comment reword (13 IN-01)
├── detect_java_kotlin.go        # MOD: strings.TrimSpace on pom fields + parent version (13 WR-01);
│                                #   comment reword (13 IN-01)
├── registry_test.go             # MOD: TestRunDetectors_EmptyRegistry injects []detectorFunc{} (12/13 WR-02)
│                                #   + comment refresh (12 IN-01 already correct — verify only)
├── detect_csharp_test.go        # MOD: padded/whitespace-only rows (13 WR-01); exact-name guard rows (13 IN-03)
├── detect_java_kotlin_test.go   # MOD: padded/whitespace-only rows (13 WR-01)
├── doc.go                       # MOD: Version-semantics contract section (D-02) + never-fail refresh (D-10)
└── (everything else)            # UNCHANGED
release/
├── update_test.go               # MOD: TestCheckForUpdate_NoOptions (mandatory — closes gate) +
│                                #   request-creation/network/decode/release-version error rows +
│                                #   MkdirAll/Create/Digest-mismatch rows for DownloadUpdate
└── (everything else)            # UNCHANGED — update.go is NOT modified (no production change needed)
README.md                        # MOD: package index row + ### Package project_probe section (D-10)
```

### Pattern 1: Go fuzz targets — seed corpus runs under normal `go test` (D-04/D-05)

**What:** Fuzz functions are ordinary `testing.F`-based tests: under plain `go test` (and CI's `go test -v ./...`), each fuzz target executes its **seed corpus** (inputs from `f.Add` plus files in `testdata/fuzz/<FuzzName>/`) as normal test cases — a panic or failure in a seed fails the build. The unbounded fuzzing loop runs only under `go test -fuzz=<FuzzName>` and must NEVER be used in CI. All of this was **verified live this session on go1.27.0**: seed mode ran `f.Add` seeds (`seed#0`, `seed#1`, …) and corpus files (named by filename), `-run=FuzzJSONManifest` selects seed mode, and a 3-second `-fuzz` run completed 5800 execs (1933/s) with a disk-writing body — so `t.TempDir()` works inside fuzz targets in both modes. Fuzz-mode findings (new interesting inputs, crashes) are written back to `testdata/fuzz/<FuzzName>/` as `<16-hex-sha256>` files [VERIFIED: internal/fuzz/fuzz.go:1047-1057].

**Fuzz-target skeleton for the three targets** (internal package — unexported detectors reachable):

```go
package projectprobe

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzJSONManifest exercises the JS/PHP detector parse paths (D-05):
// readManifest → json.Unmarshal → chains. The fuzz body only writes the
// input and calls both detectors; the invariant is no-panic on any input
// (SC3). Parse-success match (D-04): garbage JSON yields ok=false — never
// asserted, never a crash.
func FuzzJSONManifest(f *testing.F) {
	f.Add([]byte(`{"name":"acme","version":"1.2.3","description":"A pkg"}`))
	f.Add([]byte(`{"name":"vendor/pkg","version":"2.0.0"}`))
	f.Add([]byte(`{"name":`)) // truncated
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectJS(dir) // no-panic is the contract; result ignored
		if err := os.WriteFile(filepath.Join(dir, "composer.json"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectPHP(dir)
	})
}
```

**Body rules (executor contract):**
- Same shape for `FuzzTOMLManifest` (writes `pyproject.toml` → `detectPython`; `Cargo.toml` → `detectRust`) and `FuzzXMLManifest` (writes `MyApp.csproj` → `detectCSharp`; `pom.xml` → `detectJavaKotlin`; use the exact-name guard so `.csproj` alone can never be the file name — 13 IN-03).
- **Never `t.Fatal`/`require` in the fuzz body on detector results** — presence-match detectors return `ok=true` on garbage (TOML/XML), parse-success detectors return `false` (JSON); both are correct. Any assertion on the bool would turn correct behavior into a fuzz "crash". The no-panic invariant needs no assertion: a panic IS the failure.
- `t.Parallel()` is forbidden in fuzz targets.
- Keep the seed `f.Add` calls (3-5 per target) AND the corpus files — both run under `go test`.

**When to use:** D-04/D-05 only. No `-fuzz` flag anywhere in the plan, Makefile, or CI.

### Pattern 2: Seed-corpus file format — `go test fuzz v1` encoding (the phase's #1 trap)

**What:** Files in `testdata/fuzz/<FuzzName>/` are **encoded**, not raw. The format [VERIFIED: internal/fuzz/encoding.go:19-28,105-114 + live experiment]: line 1 is `go test fuzz v1`; each following line is one Go-syntax value (`[]byte("...")` for a `[]byte` parameter). A raw manifest file fails the ENTIRE package build with `unmarshal: must include version and at least one value` [VERIFIED: experiment — the failure lists every bad file]. Two working encodings, both verified:

```
go test fuzz v1
[]byte(`{"name":"acme","version":"1.2.3"}`)
```

```text
go test fuzz v1
[]byte("<Project>\n  <Version> 1.2.3 </Version>\n</Project>")
```

Encoding rules (verified):
- **One value per line** — a backtick raw literal CANNOT span lines ("raw string literal not terminated", verified). Multi-line content uses `\n` escapes in a quoted literal (verified working).
- Backtick literals are fine for single-line content; JSON/TOML/XML never contain backticks (Phase 13 D-disc-7 precedent holds).
- Non-UTF8 bytes (e.g., the BOM `\xef\xbb\xbf`) need escaped literals: `[]byte("\xef\xbb\xbf{\"name\":\"bom\"}")` (verified).
- **File names are free-form** — `ReadCorpus` reads every file regardless of name [VERIFIED: internal/fuzz/fuzz.go:941-1000; the TODO at :941 confirms names are not yet validated]. Use readable names (`package.json`, `malformed-truncated.json`); the fuzzer itself writes `<16-hex-sha256>` when it records new interesting inputs [VERIFIED: fuzz.go:1052].
- Malformed corpus files fail the test run (collected in `MalformedCorpusError`) [VERIFIED: experiment] — so the corpus must be generated correctly once, by the executor, and verified by `go test` before commit.

**When to use:** every seed corpus file. Recommended seed inventory (D-04 "real manifests"): real-world-shaped `package.json` (name/version/description), `composer.json` (vendor/package, no version — the norm), `pyproject.toml` (`[project]` PEP 621 + `[tool.poetry]` legacy + `dynamic = ["version"]`), `Cargo.toml` (`[package]` + `version.workspace = true`), `.csproj` (SDK-style + old-style namespaced + multi-PropertyGroup), `pom.xml` (child with `<parent><version>`, no own version); plus the malformed shapes the phase tests already pin: truncated JSON/XML, BOM-prefixed, whitespace-only XML elements (13 WR-01 shape), 6-quote close+reopen TOML lines (12 CR-01 shape), bracket-in-string TOML lines.

### Pattern 3: release/update.go coverage closure (D-09)

**What:** `release/update.go` is the ONLY file below the 70% file threshold (all others ≥73%; verified via `go-test-coverage --config=.testcoverage-quick.yml` this session — exit failure lists exactly `release/update.go 68.9%` with lines `60-63 65-68 70-73 75-80 91-92 102-103 112-113 117-118 130-131 136-137 141-144`). The arithmetic, verified from the coverage profile: **74 statements, 51 covered; 70% of 74 = 51.8 → 52 needed → ONE more covered statement closes the gate** (52/74 = 70.3%). Go's cover instrumentation counts a block as covered when its entry executes — branch outcomes are not tracked — so error-branch bodies count once their guard is entered.

**The mandatory row — `TestCheckForUpdate_NoOptions`:** every existing test passes `WithOwner`/`WithRepo`, so the whole module-path derivation branch (lines 59-81, 13 statements) is dead. Calling `CheckForUpdate(ctx, "v1.0.0")` with NO options enters it: `getCurrentModule()` reads `debug.ReadBuildInfo()` of the test binary — verified to succeed with `Main.Path` set in test binaries (live experiment) — `url.Parse("github.com/guionardo/go")` yields `Path="github.com/guionardo/go"` → `words = [github.com, guionardo, go]` → `owner="guionardo"`, `repo="go"`. The mock server (existing `httptest` + `githubAPIBase` override + `mu` lock pattern from update_test.go:87-132) must serve `/repos/guionardo/go/releases/latest` — assert the path in the handler to pin the derivation. This row alone: 51 + 13 = 64/74 = **86.5%**, gate closed.

**Six recommended error-path rows** (each ~10 lines, mirroring existing shapes; together 74/74 = 100%):

| Row | Trigger (cross-platform — NO chmod tricks, AGENTS.md Windows rule) | Covers |
|-----|------------------------------------------------------------------------|--------|
| `TestCheckForUpdate_RequestCreationError` | `githubAPIBase = "http://exa mple.com"` (space in host → `http.NewRequestWithContext` fails) | 91-92 |
| `TestCheckForUpdate_NetworkError` | `httptest.NewServer` then `server.Close()`; use its URL → connection refused → `githubClient.Do` error | 102-103 |
| `TestCheckForUpdate_InvalidReleaseJSON` | server returns `not-json` with 200 → decode error | 112-113 |
| `TestCheckForUpdate_InvalidReleaseVersion` | `tag_name: "not-a-version"` → `ParseVersion` error | 117-118 |
| `TestDownloadUpdate_MkdirAllError` | `targetDir = filepath.Join(existingFile, "sub")` → ENOTDIR (file-as-parent) | 130-131 |
| `TestDownloadUpdate_CreateError` | `Asset{Name: "sub/" + goos + "_" + goarch + ".tar.gz"}` (slash inside the matched name) → `os.Create` ENOTDIR | 136-137 |
| `TestDownloadUpdate_DigestMismatch` | server returns content whose digest ≠ `asset.Digest` → `asset.Download` error → `os.Remove` cleanup | 141-144 |

All rows follow the existing `mu.Lock()` + `githubAPIBase` save/restore discipline (update_test.go:87-124). The no-options row also needs `mu` (it mutates `githubAPIBase`). `release/update.go` itself is NOT modified — this is test-only (D-09: "add missing tests").

**When to use:** D-09. Verification: `make coverage-quick` green (or `go-test-coverage --config=./.testcoverage-quick.yml`).

### Pattern 4: readme.go fold-in — plain badge + multi-line HTML comments (11 WR-01/WR-02)

**What:** Two ~3-line fixes in `firstRealParagraph`/`isBadgeLine`, with matrix rows in `readme_test.go`. Current state verified: `isBadgeLine` (readme.go:113-135) strips `[..](..)` segments but leaves a bare `!` behind for the unwrapped form — `![logo](x)` → remainder `"!"` → not a badge (11-REVIEW WR-01, replicated in the review); `firstRealParagraph` (readme.go:73-74) skips only single-line `<!-- ... -->` — a multi-line comment preamble leaks `<!-- Banner -->` into the Description (11-REVIEW WR-02).

**Fix 1 — plain-badge remainder** (11-REVIEW suggested shape; current line 134 is `return strings.TrimSpace(line) == ""`):

```go
return strings.TrimSpace(line) == "" || strings.Trim(line, "!") == ""
```

The `![`-presence guard at readme.go:115 keeps prose like `!!!` or `!important` out (they lack `![`); a stripped `![logo](x)` leaves exactly `"!"` → badge. Add matrix rows: `{"plain_image_badge", "![logo](x)\npara\n", "para"}` and a shields.io-style URL row.

**Fix 2 — comment-block state** (contract from 11-REVIEW: no `<!--`-block content ever enters `para`):

```go
// in the loop, before the existing cases:
case inComment:
    if strings.Contains(trimmed, "-->") {
        inComment = false
    }
    prev = ""
case strings.HasPrefix(trimmed, "<!--") && !strings.HasSuffix(trimmed, "-->"):
    inComment = true // multi-line comment opens — consume until "-->"
    prev = ""
```

Ordering matters: `inComment` must be the FIRST case so blank lines and comment bodies inside the block are consumed (a blank line must not return a partial paragraph — comments precede the first paragraph in every realistic preamble). The existing single-line case (`HasPrefix && HasSuffix`, readme.go:73-74) stays untouched. Add row `{"multi_line_html_comment", "<!--\nBanner\n-->\npara\n", "para"}`. Optional adjacent cleanup: 11-REVIEW IN-01's dead condition at readme.go:69 (`para[len(para)-1] == prev` is always true when `prev != ""`) — simplify only if the edit lands in the same switch (not mandated by D-06/D-07).

**When to use:** D-06. Both fixes are pure-function changes — the `TestFirstRealParagraph` matrix (readme_test.go:127-158) is the home for the rows.

### Pattern 5: XML element trim + exact-name guard (13 WR-01 / 13 IN-03)

**What:** `encoding/xml` copies element text verbatim — verified in 13-REVIEW (replicated probe): `<Version> 1.2.3 </Version>` → `" 1.2.3 "`, `<name> </name>` → `" "` which **blocks the artifactId/folder-base fallback** (whitespace-only counts as present). The fix is `strings.TrimSpace` per decoded field **before** chain resolution.

```go
// detect_csharp.go — inside the collect loop (current lines 76-93):
for _, pg := range proj.PropertyGroups {
    if assemblyName == "" {
        assemblyName = strings.TrimSpace(pg.AssemblyName)
    }
    // ... same for rootNamespace, version, versionPrefix, description
}

// detect_java_kotlin.go — before the chains (current lines 97-111):
data.Name = strings.TrimSpace(pom.Name)
if data.Name == "" {
    data.Name = strings.TrimSpace(pom.ArtifactID)
}
// ...
data.Version = strings.TrimSpace(pom.Version)
if data.Version == "" && pom.Parent != nil {
    data.Version = strings.TrimSpace(pom.Parent.Version) // D-05 single-level
}
data.Description = strings.TrimSpace(pom.Description)
```

Test rows (one per shape, per 13-REVIEW WR-01): padded values → trimmed output; whitespace-only `<Version> </Version>` with `<parent><version>` → parent version; whitespace-only `<name> </name>` → artifactId/folder base. **This TrimSpace is decode hygiene, NOT version normalization** — the D-03 audit and doc.go contract must state it explicitly (Pitfall 2).

**13 IN-03 exact-name guard** in `readFirstManifest` (current line 25) — a file literally named `.csproj` passes `HasSuffix`:

```go
if e.IsDir() || len(e.Name()) <= len(suffix) || !strings.HasSuffix(e.Name(), suffix) {
    continue
}
```

Add a fixture row: a file named exactly `.csproj` → non-match (folder falls through to the next detector). **13 IN-01 comment reword** (both detectors): replace "decode error → zero struct" with "decode error → whatever decoded before the error point survives; the rest degrades to empty" — verified behavior (13-REVIEW IN-01); optionally add the truncated-after-complete-group row (it doubles as a fuzz seed).

**When to use:** D-06/D-07. `strings` is already imported in both files.

### Pattern 6: Registry test hygiene (12/13 WR-02)

**What:** `TestRunDetectors_EmptyRegistry` (registry_test.go:21-28) saves/restores `detectors` but **never mutates it** — it runs the production 7-detector registry against `runDetectors("")`, resolving relative manifest paths in the package dir. It passes only because `project_probe/` holds no manifest names, and Phase 13 made it MORE fragile (ReadDir-based `.csproj` discovery matches any package-root fixture). Fix per 13-REVIEW — inject the empty slice and use a folder that cannot collide:

```go
func TestRunDetectors_EmptyRegistry(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	detectors = []detectorFunc{} // explicit: no live detectors
	defer func() { detectors = original }()

	pd, ok := runDetectors("x")
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}
```

Also verify (do not re-edit): 12-REVIEW IN-01's stale `TestDetectorPositions` comment was already refreshed in Phase 13 — the current comment (registry_test.go:65-69) correctly says "ALL SEVEN are live" [VERIFIED]. Keep the file's no-`t.Parallel()` discipline.

**When to use:** D-07.

### Pattern 7: Version-semantics inventory + doc.go contract section (D-01/D-02)

**What:** The audit is a code read plus grep evidence; the deliverable is the doc.go contract. Inventory verified this session, verbatim from source (all 7 detectors + the two structural degrades):

| Detector | Version source (verbatim from source) | Semantics |
|----------|----------------------------------------|-----------|
| Go — `detect_go.go:26,63` | `data.Version = version // raw go directive or "" (D-02 — toolchain floor, never normalized)`; `goVersion = fields[1] // raw string, never normalized (D-02/DATA-03)` | `go` directive verbatim = toolchain floor; `toolchain` lines never match (first-token equality) |
| JS/TS — `detect_javascript.go:34` | `data.Version = m.Version // raw; "" when absent (D-06 — zero value, no code branch)` | raw JSON value; `"version": null` → `""` |
| PHP — `detect_php.go:33` | `data.Version = m.Version // raw; "" when absent — the composer norm, never fabricated` | raw JSON value; Packagist infers from tags — `""` is correct |
| Python — `detect_python.go:29` | `data.Version = fields["version"] // raw string or "" (DATA-03 — dynamic/unsupported → "", never fabricated)` | `dynamic = ["version"]` degrades in the READER (array → absent key), zero detector branches (Phase 12 pin) |
| Rust — `detect_rust.go:29` | `data.Version = fields["version"] // "" for version.workspace = true (D-07 — never resolved)` | dotted-key classification (`isBareKey` rejects `.`) degrades in the reader, zero branches (Phase 12 pin) |
| C#/.NET — `detect_csharp.go:103-106` | `data.Version = version`; `if data.Version == "" { data.Version = versionPrefix // D-03 chain; "" when both absent (never 1.0.0 fabrication) }` | `Version` → `VersionPrefix` → `""`; MSBuild's implicit `1.0.0` never fabricated; `$(...)` placeholders raw (A2) |
| Java/Kotlin — `detect_java_kotlin.go:104-107` | `data.Version = pom.Version`; `if data.Version == "" && pom.Parent != nil { data.Version = pom.Parent.Version // D-05: single-level parent inheritance (same file) }` | child `<version>` → `<parent><version>` (single level, same file); Maven Super POM `4.0.0` default never fabricated; `${revision}` raw |

**doc.go contract section (D-02)** — a new paragraph after the detector-list paragraph (doc.go:35-53), wording recommendation:

```go
// # Version semantics (DATA-03)
//
// Version is always the raw manifest string — verbatim, never normalized,
// never fabricated. Absent or unsupported values yield "". The rules:
//
//   - go.mod: the "go" directive, reported as the toolchain floor — the
//     minimum Go version the module was written for, NOT the release
//     version; "toolchain" lines are never the directive.
//   - pyproject.toml: the [project] version; a "dynamic" version array
//     degrades to "" — never resolved, never guessed.
//   - Cargo.toml: the [package] version; version.workspace = true degrades
//     to "" — never resolved from a workspace root.
//   - .csproj: <Version>, falling back to <VersionPrefix>; the MSBuild
//     implicit 1.0.0 default is never reported.
//   - pom.xml: <version>, inheriting <parent><version> when absent (single
//     level, from the same file); the Maven Super POM 4.0.0 default is
//     never reported.
//   - package.json / composer.json: the "version" field verbatim.
//
// Placeholders ($(...) in MSBuild, ${...} in Maven) are reported raw, never
// resolved. XML element text is whitespace-trimmed at decode time (decode
// hygiene — the value itself is still never normalized).
```

**When to use:** D-01/D-02. The doc.go paragraph is the single downstream-readable place; the anti-feature audit (Pattern 8) is the proof it's true.

### Pattern 8: Anti-feature audit grep set (D-03 / ROBT-05)

**What:** The exact grep commands, run this session against the current tree — **all five return zero matches** [VERIFIED]. The verification report records the commands + empty output + the code-read evidence from Pattern 7.

```bash
# 1. Build-tool execution — no os/exec anywhere
grep -rnE "os/exec|exec\.Command" project_probe/            # → NONE
# 2. Network — no net/http anywhere
grep -rnE "net/http|http\." project_probe/                  # → NONE
# 3. Symlink following — no EvalSymlinks/Readlink/Lstat/Symlink
grep -rnE "EvalSymlinks|os\.Readlink|os\.Lstat|os\.Symlink" project_probe/   # → NONE
# 4. Recursive discovery — no WalkDir/filepath.Walk (root-scoped only)
grep -rnE "WalkDir|filepath\.Walk" project_probe/           # → NONE
# 5. Version normalization — no semver parsing, v-stripping, replace, regexp
grep -rnE "ParseVersion|semver|version\.Compare|strings\.(Replace|TrimPrefix|TrimSuffix)" project_probe/   # → NONE
grep -rn "regexp" project_probe/                            # → NONE
```

Nuances the report must state: `readManifest`'s `f.Stat()` follows a symlink to check `IsRegular()` — that is the read path, not discovery (documented at manifest.go:35-39, "ROBT-05 bans symlink *discovery*, not the read path"); the sanctioned `strings.TrimSpace` on XML fields (Pattern 5) is decode hygiene, explicitly NOT normalization (the audit's grep #5 does not match `TrimSpace`, and the doc.go contract says why).

**When to use:** D-03 — a verification-report section (or dedicated audit section in 14-VERIFICATION.md), not a code change.

### Pattern 9: README package index row (D-10)

**What:** The README table (README.md:13-36) is a sorted list of `| [name](#package-<dir>) | \`import\` | description |` rows, and every package has a `### Package <name>` section with import, description, and a usage snippet. The project_probe row slots alphabetically between `pathtools` and `reflecttools` (or per repo preference — rows are currently sorted by package name):

```markdown
| [projectprobe](#package-project_probe) | `project_probe` | Best-effort project detection across 7 languages (manifest-first, stdlib-only, never fails) |
```

plus a `### Package project_probe` section following the per-package section precedent (import line, one-paragraph description of the never-fail contract + the 7-detector cascade, a `Probe` usage snippet). Section body wording is the agent's discretion; the ROW is the lock.

**When to use:** D-10, final plan.

### Anti-Patterns to Avoid

- **Raw bytes in `testdata/fuzz/`:** corpus files must be `go test fuzz v1`-encoded; raw manifests fail the whole package build (verified). Generate the encoded form, run `go test` once, then commit.
- **Multi-line backtick literals in corpus files:** one value per line; use `\n` escapes in quoted literals (verified failure + fix).
- **`t.Fatal`/assertions on detector bools in fuzz bodies:** presence-match (TOML/XML) vs parse-success (JSON) semantics differ; asserting either crashes the fuzzer on correct behavior. No-panic needs no assertion.
- **`-fuzz` in CI or in the plan:** the unbounded loop never runs in CI; seeds run under plain `go test` — that is SC3's contract.
- **`t.Parallel()` in fuzz targets or registry/release global-state tests:** fuzz targets can't be parallel; registry tests mutate `detectors`; release tests mutate `githubAPIBase` (existing `mu` discipline).
- **chmod-based error triggers in release tests:** Windows permission semantics differ (AGENTS.md); use ENOTDIR triggers (file-as-parent, slash-in-name) — cross-platform.
- **Fixing all 28 pre-existing lint issues:** the whole v1.7 branch's new-vs-main debt; out of scope. Delta-zero on Phase 14's own edits is the D-10 contract.
- **A shared `readXMLManifest` helper for the trim fix:** uncalled helper → 0% file coverage → file:70 gate failure (Phase 13 D-disc-8 lesson).
- **Version normalization creeping into the audit narrative:** `strings.TrimSpace` on XML fields is mandated decode hygiene (13 WR-01); the doc.go contract must name it as such so the D-03 audit and D-01 claim don't contradict each other.
- **Reordering the README table or renaming rows:** add one row; keep the sorted order and anchor format.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fuzz harness + corpus management | A custom property/random-input generator | Go's native `testing.F` fuzz targets + `testdata/fuzz/` seed corpus | Native fuzzing runs under plain `go test` (SC3's exact contract), writes crash/interesting inputs to the corpus automatically, and is maintained by the toolchain — verified live this session |
| Seed-corpus encoding | Hand-rolled "write the manifest" files | The `go test fuzz v1` header format (Pattern 2) | The format is what `internal/fuzz` reads; anything else fails the build (verified) — encoding is 2 lines per file |
| Coverage-gate measurement | A custom percentage script | `make coverage-quick` → `go-test-coverage --config=./.testcoverage-quick.yml` | The repo's established gate; already installed and verified this session |
| Version-semantics proof | Rewriting detectors to "be safe" | The audit (Patterns 7-8) — read + grep | The detectors are already raw-by-construction; the phase's job is to PROVE and DOCUMENT, not to add defensive code (new code = new coverage risk) |
| README extraction heuristics | Rewriting `firstRealParagraph` | The two-line fixes of Pattern 4 | The heuristics were built in Phase 11; WR-01/WR-02 are edge-case gaps with review-prescribed shapes |
| Test triggers for I/O errors | Platform-specific hacks (chmod, syscalls) | ENOTDIR triggers (file-as-parent, slash-in-filename), closed-server URLs | Cross-platform on all three CI OSes (AGENTS.md Windows rules); mirrors existing test shapes |
| Fuzz seed generation | A generator script committed to the repo | Hand-written encoded files (real manifests + pinned malformed shapes) | Seeds are static fixtures; a generator adds a second source of truth for zero benefit |

**Key insight:** every "hard" problem in this phase is already solved — the detectors are raw-by-construction, the anti-features are absent, the parsers don't panic (pinned by phases 10-13's 236 tests). Phase 14's risk is entirely in **proving** these facts (audit, fuzz, doc.go) and **closing** two small gaps (readme.go edge cases, one coverage test row). The one genuinely new failure mode is the corpus-file format — a 2-line-per-file encoding detail that fails the build if wrong (Pattern 2).

## Common Pitfalls

### Pitfall 1: Raw manifests placed in `testdata/fuzz/` fail the whole build
**What goes wrong:** `go test ./...` fails with `"testdata/fuzz/FuzzJSONManifest/package.json": unmarshal: must include version and at least one value` — every corpus file is listed, and the package fails.
**Why it happens:** `internal/fuzz.ReadCorpus` decodes each corpus file as a `go test fuzz v1`-encoded entry (header + Go-syntax values); raw bytes have no header (verified in source and by experiment).
**How to avoid:** Encode every seed file with the header + `[]byte(...)` line (Pattern 2); run `go test ./project_probe/...` once before committing — green means the corpus is well-formed.
**Warning signs:** A fresh `testdata/fuzz/` commit that has never been through `go test`; corpus files that look like the source manifests.

### Pitfall 2: The XML TrimSpace fold-in tripping the "no normalization" audit
**What goes wrong:** The D-03 audit greps for version normalization, and the D-06 fold-in adds `strings.TrimSpace` to version fields — a naive reviewer reads the diff as normalization.
**Why it happens:** 13 WR-01 mandates trimming padded/whitespace-only XML element text BEFORE chain resolution; it is decode hygiene on a verbatim value, not semantic transformation.
**How to avoid:** The doc.go contract section (Pattern 7) explicitly names "XML element text is whitespace-trimmed at decode time (decode hygiene — the value itself is still never normalized)"; the audit report states the same distinction.
**Warning signs:** Audit findings that quote `strings.TrimSpace` as a violation; a planner task that "removes trimming to pass the audit".

### Pitfall 3: Fuzz bodies asserting detector results
**What goes wrong:** `if !ok { t.Fatal(...) }` in a fuzz body — the fuzzer reports a "crash" on every malformed input, and seed mode fails the build.
**Why it happens:** Presence-match detectors (TOML/XML) return `ok=true` on garbage; parse-success detectors (JSON) return `false`; both are correct, and the never-fail contract never promises `ok=true`.
**How to avoid:** Fuzz bodies call the detectors and ignore results — no-panic is the only invariant, and a panic IS the failure signal.
**Warning signs:** `require`/`assert`/`t.Fatal` inside `f.Fuzz` bodies.

### Pitfall 4: The coverage gate "needs" production changes
**What goes wrong:** A planner task modifies `release/update.go` to make it testable (extracting a helper, adding a branch) — new production code changes the denominator and risks new uncovered lines.
**Why it happens:** 68.9% looks like a big gap; the profile shows 23 uncovered statements across two functions.
**How to avoid:** The math says ONE covered statement closes the gate (74 stmts, 52 needed); the no-options test covers 13. Test-only changes (Pattern 3).
**Warning signs:** A plan diff touching `release/update.go` outside `_test.go`.

### Pitfall 5: Lint "cleanliness" interpreted as fixing the 28 pre-existing issues
**What goes wrong:** The executor runs `GOTOOLCHAIN=go1.26.4 golangci-lint run` and starts fixing cyclop/decorder/gosmopolitan issues across the whole package — massive scope creep in the final phase.
**Why it happens:** `new-from-merge-base: main` reports the entire v1.7 branch as new (all project_probe code is new vs main); 28 issues pre-exist (verified this session).
**How to avoid:** Record the baseline count (28) at phase start; D-10's contract is delta-zero — Phase 14's edits add no issues. Pre-commit `gofmt`/`goimports` handle formatting on touched files.
**Warning signs:** Lint-fix tasks targeting files Phase 14 doesn't otherwise touch.

### Pitfall 6: Fuzz seeds that duplicate test fixtures instead of real manifests
**What goes wrong:** The corpus is a copy of the inline test fixtures — D-04's "seeded with real manifest files" is unmet, and the fuzz value (real-world shapes) is lost.
**Why it happens:** The repo convention is inline fixtures; reaching for the same strings is natural.
**How to avoid:** Include realistic real-world shapes (shields-style README-adjacent JSON, IDE-reformatted pretty-printed XML, migrated Poetry 2.x pyproject with `dynamic`, workspace-inherited Cargo) alongside the pinned malformed shapes (Pattern 2 inventory).
**Warning signs:** Corpus files byte-identical to `detect_*_test.go` fixtures with no real-world shapes.

## Code Examples

Verified patterns from official sources and this session's experiments:

### Fuzz target + encoded corpus (verified on go1.27.0 this session)
```go
// fuzz_test.go (package projectprobe) — one file, three targets
func FuzzTOMLManifest(f *testing.F) {
	f.Add([]byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir() // works in seed AND fuzz mode (verified)
		if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectPython(dir) // readManifest → readTOMLSection → chains (D-05)
		if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectRust(dir)
	})
}
```
```text
# testdata/fuzz/FuzzTOMLManifest/pyproject-real.toml — ENCODED, not raw
go test fuzz v1
[]byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A library.\"\n")
```
```text
# testdata/fuzz/FuzzXMLManifest/csproj-oldstyle.xml — escaped multi-line literal
go test fuzz v1
[]byte("<Project ToolsVersion=\"15.0\" xmlns=\"http://schemas.microsoft.com/developer/msbuild/2003\">\n  <PropertyGroup>\n    <Version> 1.2.3 </Version>\n  </PropertyGroup>\n</Project>")
```
Run: `go test ./project_probe/...` (seed mode; CI-safe). Fuzz: `go test -fuzz=FuzzTOMLManifest -fuzztime=30s ./project_probe/` (developer-local only).

### release/update.go coverage rows (Pattern 3) — the no-options row
```go
// update_test.go — follows the existing mu + githubAPIBase discipline
func TestCheckForUpdate_NoOptionsDerivesOwnerRepo(t *testing.T) { //nolint:paralleltest // global githubAPIBase
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/guionardo/go/releases/latest", r.URL.Path) // derived from module path
		_, _ = fmt.Fprint(w, `{"tag_name": "v2.0.0", "name": "v2.0.0", "assets": []}`)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0") // NO options → module derivation
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.True(t, newer)
}
```

### readme.go fold-in (Pattern 4) — the two fixes
```go
// isBadgeLine final line (current readme.go:134): plain image remainder of only "!" is a badge
return strings.TrimSpace(line) == "" || strings.Trim(line, "!") == ""

// firstRealParagraph — comment-block state (add before the existing cases at readme.go:60-79)
var inComment bool
for _, line := range strings.Split(string(content), "\n") {
    trimmed := strings.TrimSpace(line)
    switch {
    case inComment:
        if strings.Contains(trimmed, "-->") {
            inComment = false
        }
        prev = ""
    case strings.HasPrefix(trimmed, "<!--") && !strings.HasSuffix(trimmed, "-->"):
        inComment = true
        prev = ""
    // ... existing cases unchanged
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Fuzzing as a separate binary/tool (go-fuzz, OSS-Fuzz) | Native `testing.F` fuzz targets in-package | Go 1.18 (2022) | Seeds run under plain `go test`/CI; corpus is repo fixtures; `-fuzz` is a local dev loop — SC3's "runs under normal go test" is native |
| Raw bytes as corpus entries | `go test fuzz v1` encoded entries | Go 1.18 (with native fuzzing) | Corpus files must be encoded (verified this session — raw files fail the build); filenames free-form today, hash-named by the fuzzer |
| `os.Open` manifest reads (FIFO hang) | `O_NONBLOCK` + `Mode().IsRegular()` gate | Phase 11 WR-01 fix | Already shipped; the fuzz bodies exercise this gate on arbitrary bytes |
| Cyclomatic-complexity lint failures blocking commits | `new-from-merge-base` incremental lint + advisory baseline | golangci-lint v2 | 28 pre-existing new-vs-main issues in project_probe are advisory; Phase 14 must be delta-zero |

**Deprecated/outdated:**
- `go-fuzz` (github.com/dvyukov/go-fuzz): superseded by native `testing.F` fuzzing; not used here.
- Raw-manifest "seed corpus" assumptions from pre-1.18 fuzz tooling: the `go test fuzz v1` format is mandatory (verified).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Coverage-file percentage uses the same statement math as my profile parse (52/74 needed) | Pattern 3 | Low — verified against `go-test-coverage`'s own report (68.9% for the same file); the no-options row covers 13 statements regardless of tooling rounding, far past the boundary |
| A2 | `f.Add` seeds and corpus files both count toward `go test -coverprofile` | Pattern 1 | Low — seeds execute as test cases; the coverage gate's `project_probe` rows are already far above thresholds either way |
| A3 | "golangci-lint clean" (D-10) means delta-zero on the 28-issue baseline, not fixing the 28 | Pattern 5 / Pitfall 5 | Medium — if the user meant "fix all issues", the phase grows substantially; the milestone's prior phases treated lint as advisory (13-RESEARCH), so delta-zero is the consistent reading; flag at plan review |
| A4 | README wants a full `### Package project_probe` section, not just the table row | Pattern 9 | Low — the row is the lock (D-10); the section follows the README's own per-package precedent; executor discretion per CONTEXT |
| A5 | Fuzz bodies call detectors directly rather than `Probe` | Pattern 1 | Low — D-05's parenthetical ("readManifest → decode → chains") names the detector path; direct calls are the faithful, faster reading |
| A6 | `url.Parse("github.com/guionardo/go")` yields `Path="github.com/guionardo/go"` (no scheme) | Pattern 3 | Low — standard stdlib behavior for scheme-less input; the no-options test asserts the derived path in the mock handler, so a wrong derivation fails the test visibly |
| A7 | The 16-hex corpus filenames (fuzzer convention) vs readable names both work today | Pattern 2 | Low — verified `ReadCorpus` reads all files; readable names chosen for reviewability, hash names for fuzzer-written files |

## Open Questions

1. **Lint baseline disposition (A3)**
   - What we know: `GOTOOLCHAIN=go1.26.4 golangci-lint run` works and reports 28 new-vs-main issues, all pre-existing in project_probe/release from phases 10-13 (verified this session). D-10 says "go vet + golangci-lint clean".
   - What's unclear: whether "clean" means delta-zero (consistent with prior phases' advisory treatment) or fixing all 28.
   - Recommendation: plan for delta-zero (record baseline in the verification report); if the user wants the 28 fixed, that is a separate scope decision — surface at plan review.

2. **README row position**
   - What we know: the table is sorted by package name; `project_probe` sorts between `path_tools` and `reflect_tools`.
   - What's unclear: whether the user prefers alphabetical strictness or a "newest package last" convention.
   - Recommendation: alphabetical (repo precedent), agent's discretion per CONTEXT.

3. **fuzz target file layout**
   - What we know: one `fuzz_test.go` with three targets vs three files — both valid; the corpus dirs are per-target regardless.
   - What's unclear: reviewer preference for file-per-target.
   - Recommendation: one file (smaller diff, no filename-suffix risk); split only if the executor finds it unwieldy.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | 1.27.0 local (darwin/arm64); module + CI pin 1.26.4 | `GOTOOLCHAIN=go1.26.4` verified working for lint |
| `go-test-coverage` v2 | `make coverage-quick` (D-09 gate) | ✓ | installed (`go install` via Makefile target) | — |
| `golangci-lint` | D-10 lint check | ✓ | installed (`/Users/guionardo/go/bin/golangci-lint`) | requires `GOTOOLCHAIN=go1.26.4` (verified); no CI lint job |
| `make` | coverage/lint/quality targets | ✓ | — | — |
| `testify` / `go-version` / `go-digest` | tests | ✓ | v1.11.1 / v1.9.0 / v1.0.0 in go.mod | — |
| Network services (GitHub API, Redis, …) | none in this phase | — | — | Fuzz + tests are fully local; release tests use `httptest` |
| Docker (cache E2E) | none in this phase | — | — | `coverage-quick` explicitly excludes E2E |

**Missing dependencies with no fallback:** none — every requirement is met with the installed toolchain.

**Missing dependencies with fallback:** none.

## Security Domain

> `security_enforcement: true` in `.planning/config.json` — included. ASVS level 1. This phase adds no attack surface: no new production code paths, no new inputs beyond the fuzz harness itself.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no auth in project_probe) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (root-scoped reads only, no permissions logic) |
| V5 Input Validation | yes | Never-fail presence-match degradation + native fuzz targets (SC3): arbitrary manifest bytes are first-class inputs; parsers return errors, never panic; `readManifest` caps at 1 MB, strips BOM, gates FIFOs |
| V6 Cryptography | no | — (release's sha256/digest is pre-existing, unchanged) |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed-manifest parser panic / crash (the fuzz target's job) | DoS | Native fuzz targets + seed corpus under `go test`; `callDetector` recover as the outer net (registry.go:44-53, unchanged); `encoding/xml`/`json` return errors, never panic (phases 11-13 verified) |
| FIFO / special-file hang at read time | DoS | Existing `O_NONBLOCK` + `IsRegular()` gate (manifest.go:29-43) — exercised by every fuzz body via `readManifest` |
| Oversized manifest memory exhaustion | DoS | Existing 1 MB cap (manifest.go:13,45-48) — arbitrary fuzz inputs route through it |
| Symlink-based discovery / traversal | Tampering | No `EvalSymlinks`/`WalkDir`/`Readlink` anywhere (grep-verified, D-03); root-scoped single-level reads; `f.Stat()` symlink-following on the READ path is documented, not discovery |
| Build-tool execution / network exfiltration | Tampering | No `os/exec`, no `net/http` (grep-verified, D-03); detection is offline by construction |
| Fuzz-corpus poisoning (malformed corpus file) | DoS | Corpus files fail the build loudly (`MalformedCorpusError`, verified) — cannot silently poison CI; corpus is repo-reviewed fixtures |

## Sources

### Primary (HIGH confidence — verified this session)
- **Local toolchain experiments (go1.27.0, darwin/arm64):** fuzz seed-mode execution under `go test` / `-run=FuzzX`; `go test fuzz v1` corpus encoding (raw bytes fail, encoded pass, single-line values, `\n` escapes, backtick single-line OK, multi-line backtick fails); `t.TempDir()` in fuzz bodies (seed + fuzz mode, 5800 execs/3s); corpus filename freedom; `debug.ReadBuildInfo()` in test binaries.
- **Go stdlib source reads:** `internal/fuzz/encoding.go:19-28,105-114` (corpus encoding format + `must include version and at least one value`); `internal/fuzz/fuzz.go:941-1000,1047-1057` (`ReadCorpus` reads all files; `writeToCorpus` 16-hex naming).
- **Repo source reads (this session):** all 7 detectors + `toml.go`/`json.go`/`manifest.go`/`readme.go`/`registry.go`/`probe.go`/`doc.go` (version-semantics quotes in Pattern 7); `release/update.go`, `update_test.go`, `self_update.go`, `self_update_test.go`, `release.go`, `version.go` (coverage inventory, Pattern 3); `.testcoverage-quick.yml`, `.github/workflows/go.yml`, `.golangci.yml`, `.planning/config.json`, `CONVENTIONS.md`, `README.md`.
- **Tool runs:** `go-test-coverage --config=./.testcoverage-quick.yml` (only red file: `release/update.go 68.9%`); profile arithmetic (74 stmts / 51 covered); anti-feature greps (5 patterns, zero matches); `GOTOOLCHAIN=go1.26.4 golangci-lint run` (28 new-vs-main issues, all pre-existing); `go vet` (clean).

### Secondary (MEDIUM confidence)
- **Prior-phase reviews (repo):** 11-REVIEW WR-01/WR-02 (+fix shapes), 12-REVIEW WR-01 (+fix), 13-REVIEW WR-01/WR-02/IN-01/IN-02/IN-03 (+fix shapes) — read this session; the fixes' test rows are prescribed there.
- **13-RESEARCH.md / 13-PATTERNS.md (repo):** XMLName local-name matching, inline-decode shape, `readFirstManifest`, fixture conventions — the fuzz bodies build on these verified contracts.

### Tertiary (LOW confidence)
- Web research: **not performed** — `.planning/config.json` disables all web providers (exa/brave/firecrawl/tavily/ref/perplexity all `false`), and every research question was answerable from the codebase plus direct toolchain experiments (which outrank documentation for behavior claims). Fuzz mechanics are documented in `go.dev/security/fuzzing` for reference, but all behavioral claims here rest on the live experiments.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; all behavior verified on the actual toolchain this session.
- Architecture: HIGH — plan shape (4 plans) mirrors Phases 12/13 granularity; every fold-in has a review-prescribed shape and a verified current-state read.
- Pitfalls: HIGH — the corpus-format trap and the coverage math are directly verified; the lint-baseline interpretation is the one flagged uncertainty (A3).

**Research date:** 2026-09-29
**Valid until:** 2026-10-29 (30 days — Go fuzz mechanics are stable; corpus format is toolchain-versioned and verified against both 1.26.4 and 1.27.0 behaviors where applicable)