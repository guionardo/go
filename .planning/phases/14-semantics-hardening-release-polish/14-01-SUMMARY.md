---
phase: 14-semantics-hardening-release-polish
plan: 01
subsystem: testing
tags: [fuzzing, seed-corpus, go-fuzz, project-probe, no-panic, hardening]

# Dependency graph
requires:
  - phase: 13-xml-detectors-csharp-net-java-kotlin
    provides: all 7 detectors live (detectJS/detectPHP/detectPython/detectRust/detectCSharp/detectJavaKotlin) with readManifest → decode → chains
provides:
  - Three locked fuzz targets (D-04/D-05) — FuzzJSONManifest, FuzzTOMLManifest, FuzzXMLManifest — exercising detector parse paths with a no-panic invariant
  - 20-file `go test fuzz v1`-encoded seed corpus (real-world manifests + pinned malformed shapes) under project_probe/testdata/fuzz/
  - SC3 proof: fuzz seeds + corpus execute under plain `go test` on all three CI OSes; no -fuzz flag anywhere in committed config
affects: [14-04 (D-03 anti-feature audit), verify-work UAT, ROBT-05 evidence chain]

# Actuals (#2632) — pairs with the plan's estimate (38000 tokens, 2 tasks, low confidence)
actuals:
  tokens: 3246        # chars/4 over the realized diff (12986 chars)
  tasks: 2
  commits: 2          # MEASURED: git rev-list --count 8461196483a99efedcec5249846128c8bee8abae..HEAD

# Commit ledger (#3968)
plan_head_before: 8461196483a99efedcec5249846128c8bee8abae
plan_head_after: 3b420152af79769ac7e1cd402e6798287b2c91de

# Tech tracking
tech-stack:
  added: [none — Go stdlib testing.F fuzzing only; go.mod/go.sum unchanged]
  patterns:
    - "Go native fuzz targets (testing.F) with f.Add seeds + testdata/fuzz/<Target>/ encoded corpus — seed mode under plain go test (SC3)"
    - "go test fuzz v1 corpus encoding: header line + one Go-syntax []byte value per line; \\n escapes for multi-line, escaped literals for BOM bytes"
    - "Fuzz bodies write manifests to t.TempDir and call detectors with results ignored — no-panic is the only invariant (Pitfall 3)"

key-files:
  created:
    - project_probe/fuzz_test.go
    - project_probe/testdata/fuzz/FuzzJSONManifest/ (6 files)
    - project_probe/testdata/fuzz/FuzzTOMLManifest/ (7 files)
    - project_probe/testdata/fuzz/FuzzXMLManifest/ (7 files)
  modified: []

key-decisions:
  - "Fuzz bodies call detectors directly (D-05) — readManifest → decode → chains — not Probe(): detector-direct is ~4x faster and reaches per-manifest targets the cascade order hides"
  - "No detector-result assertions in f.Fuzz bodies (Pitfall 3): presence-match (TOML/XML) vs parse-success (JSON) semantics differ; a panic IS the failure signal"
  - "Corpus mixes real-world shapes (composer.json without version, pyproject dynamic, Cargo version.workspace, old-style padded .csproj, pom parent version) with pinned malformed shapes (truncated, BOM, whitespace-only, 12 CR-01 6-quote/bracket-in-string) — not byte-copies of test fixtures (Pitfall 6)"
  - "FuzzXMLManifest writes 'MyApp.csproj', never bare '.csproj' — the 13 IN-03 exact-name guard makes a bare suffix file non-matchable"

patterns-established:
  - "Pattern: detector-direct fuzz targets in the internal package (package projectprobe reaches unexported detectors), t.TempDir + os.WriteFile(0o600) body, t.Skipf on write error, t.Parallel forbidden"
  - "Pattern: go test fuzz v1 corpus encoding rules (header mandatory; one value per line; backtick literals single-line only; escaped literals for non-UTF8) — raw bytes fail the whole build with MalformedCorpusError"
  - "Pattern: corpus well-formedness gate = go test ./project_probe/... green before commit"

requirements-completed: [ROBT-05]

# Coverage metadata (#1602) — drives deterministic UAT routing in verify-work
coverage:
  - id: D1
    description: "Three fuzz targets (FuzzJSONManifest, FuzzTOMLManifest, FuzzXMLManifest) in project_probe/fuzz_test.go exercising the six manifest-detector parse paths (readManifest → decode → chains) with a no-panic invariant; 11 f.Add seeds run as seed#N cases under plain go test"
    requirement: ROBT-05
    verification:
      - kind: unit
        ref: "go test ./project_probe/... -run 'FuzzJSONManifest|FuzzTOMLManifest|FuzzXMLManifest' -v"
        status: pass
      - kind: unit
        ref: "GOOS=windows go vet ./project_probe/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "20-file encoded seed corpus under project_probe/testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/ (6 JSON + 7 TOML + 7 XML), go test fuzz v1 format, mixing real-world manifest shapes with pinned malformed shapes; executes as named seed cases and proves no-panic on malformed input"
    requirement: ROBT-05
    verification:
      - kind: unit
        ref: "go test ./project_probe/... (270 passed, no MalformedCorpusError)"
        status: pass
      - kind: unit
        ref: "head -1 project_probe/testdata/fuzz/*/* == 'go test fuzz v1' (all 20); grep -c '^\\[\\]byte' == 1 per file"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-09-29
status: complete
---

# Phase 14 Plan 1: Fuzz Targets and Encoded Seed Corpus Summary

**Three fuzz targets (FuzzJSONManifest/FuzzTOMLManifest/FuzzXMLManifest) exercising the six manifest-detector parse paths (readManifest → decode → chains) with a 20-file `go test fuzz v1`-encoded seed corpus — the SC3 no-panic invariant proven by seed-mode execution under plain `go test ./project_probe/...`**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-29T10:08:16Z
- **Completed:** 2026-09-29T10:14:XXZ
- **Tasks:** 2
- **Files modified:** 21 (1 new test file + 20 new corpus fixtures)

## Accomplishments

- **Fuzz targets (D-04/D-05):** `project_probe/fuzz_test.go` (package projectprobe) defines FuzzJSONManifest (writes package.json → detectJS, composer.json → detectPHP), FuzzTOMLManifest (writes pyproject.toml → detectPython, Cargo.toml → detectRust), and FuzzXMLManifest (writes MyApp.csproj → detectCSharp, pom.xml → detectJavaKotlin) — 11 f.Add seeds total, bodies write to t.TempDir with `os.WriteFile(..., 0o600)` and call detectors with results ignored. No testify, no t.Parallel, no require/assert/t.Fatal anywhere (Pitfall 3 — no-panic is the only invariant; a panic IS the failure signal).
- **Encoded seed corpus:** 20 files across `testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/` in `go test fuzz v1` format (mandatory header + exactly one Go-syntax `[]byte` value per line). Real-world shapes (package.json name/version/description, composer.json without version, pyproject `[project]` PEP 621 + `[tool.poetry]` + `dynamic = ["version"]`, Cargo `version.workspace = true`, SDK-style + old-style namespaced .csproj with padded `<Version> 1.2.3 </Version>` (13 WR-01), pom.xml with `<parent><version>` and no own version (Maven inheritance)) plus pinned malformed shapes (truncated JSON/XML, UTF-8 BOM-prefixed via escaped literal, duplicate keys, `"version": null`, whitespace-only XML elements, 12 CR-01 6-quote close+reopen and bracket-in-string TOML).
- **SC3 proof:** all 3 fuzz targets + 11 f.Add seeds + 20 corpus files execute and pass under plain `go test ./project_probe/...` (270 tests green) — exactly what CI's `go test -v ./...` runs on Linux/macOS/Windows. The unbounded `-fuzz` flag appears nowhere in committed config (grep-verified).
- **Coverage verified per commit:** `go test ./project_probe/... -cover` = 94.0% (≥80% package threshold). The repo-wide `make coverage-quick` known-red on `release/update.go` 68.9% is pre-existing and closed by plan 14-03 — documented, not caused by this plan.

## Task Commits

Each task was committed atomically:

1. **Task 1: fuzz_test.go with FuzzJSONManifest, FuzzTOMLManifest, FuzzXMLManifest + f.Add seeds** - `ea63422` (test)
2. **Task 2: encode the seed corpora (20 files)** - `3b42015` (test)

**Plan metadata:** `docs(14): complete fuzz-targets plan` (committed with STATE/ROADMAP updates)

## Files Created/Modified

- `project_probe/fuzz_test.go` - Three fuzz targets exercising detector parse paths (D-05); bodies write manifest fixtures via t.TempDir + os.WriteFile and call detector pairs with results ignored; 11 f.Add seeds
- `project_probe/testdata/fuzz/FuzzJSONManifest/{package.json,composer.json,malformed-truncated.json,bom-prefixed.json,duplicate-keys.json,version-null.json}` - Encoded JSON seeds: real package.json shape, composer norm (no version), truncated, BOM-prefixed (escaped literal), duplicate-keys (last-wins), version-null
- `project_probe/testdata/fuzz/FuzzTOMLManifest/{pyproject-real.toml,pyproject-poetry.toml,pyproject-dynamic.toml,cargo-real.toml,cargo-workspace.toml,toml-6quote.toml,toml-bracket-string.toml}` - Encoded TOML seeds: PEP 621, legacy poetry, dynamic version, Cargo real, version.workspace, 12 CR-01 adversarial shapes
- `project_probe/testdata/fuzz/FuzzXMLManifest/{csproj-sdk.xml,csproj-oldstyle.xml,csproj-whitespace-only.xml,pom-child.xml,pom-pretty.xml,xml-truncated.xml,xml-whitespace-only.xml}` - Encoded XML seeds: SDK-style, old-style with padded Version, whitespace-only elements (13 WR-01), child pom with parent version, pretty-printed namespaced pom, truncated, whitespace-only

## Decisions Made

- Detector-direct fuzz bodies (D-05) rather than `Probe()` — exercises exactly readManifest → decode → chains and is ~4x faster (RESEARCH measured 1933 execs/s); the cascade order would make per-manifest targets unreachable.
- Fuzz bodies never assert detector results — presence-match (TOML/XML `ok=true` on garbage) vs parse-success (JSON `ok=false`) semantics differ; asserting either crashes the fuzzer on correct behavior. No-panic needs no assertion.
- Corpus seeds are real-world shapes, not byte-copies of test fixtures (Pitfall 6) — each format's seed differs from existing fixture content (extra fields like `license`, distinct names/versions).
- `MyApp.csproj` (real project name) is the only .csproj fixture name — a bare `.csproj` file is non-matchable under the 13 IN-03 exact-name guard.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `gsd_run query git.base-branch --is-protected` returns the base branch name (`main`) rather than a boolean — the #3819 pre-commit guard treated it as unavailable and fell back to the documented five-name protected-branch check (branch `gsd/v1.7-project-probe` is not protected). Not a plan deviation; guard semantics confirmed before committing.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The fuzz harness + corpus are the hardening tracer for Phase 14 — the no-panic invariant is proven and will be re-exercised by every future `go test ./project_probe/...` run.
- Ready for 14-02 (correctness fold-in: readme.go badge/comment fixes, XML trim, registry test hygiene), 14-03 (release/update.go coverage closure — closes the repo-wide `make coverage-quick` known-red), 14-04 (docs + D-03 anti-feature audit — the fuzz targets' no-assertion contract and seed corpus are audit evidence).
- Threat register T-14-01..T-14-05 + T-14-SC carried in the plan's threat_model; no new threat surface introduced by this plan (test-only files; corpus content is repo-reviewed fixture data).

## Self-Check: PASSED

- FOUND: project_probe/fuzz_test.go; project_probe/testdata/fuzz/*/* (20 files); 14-01-SUMMARY.md
- FOUND: commit ea63422 (task 1); commit 3b42015 (task 2)
- `go test ./project_probe/...` exits 0 (270 tests)

---
*Phase: 14-semantics-hardening-release-polish*
*Completed: 2026-09-29*