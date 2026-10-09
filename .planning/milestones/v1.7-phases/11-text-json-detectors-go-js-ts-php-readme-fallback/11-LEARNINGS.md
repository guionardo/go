---
phase: 11
phase_name: "Text/JSON Detectors — Go, JS/TS, PHP + README Fallback"
project: "github.com/guionardo/go"
generated: "2026-09-30"
counts:
  decisions: 8
  lessons: 4
  patterns: 7
  surprises: 4
missing_artifacts:
  - "11-UAT.md"
---

# Phase 11 Learnings: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback

## Decisions

### WR-01 gate shape amended: O_NONBLOCK open required
The 10-REVIEW.md fix (stat the opened handle) is insufficient — `os.Open` on a FIFO blocks at open time, before `f.Stat()` can run. The gate is `os.OpenFile(..., os.O_RDONLY|syscall.O_NONBLOCK, 0)` + `f.Stat()` + `Mode().IsRegular()`.

**Rationale:** A FIFO named go.mod/package.json/README.md must yield `(nil, false)` in milliseconds, not hang the probe forever. `syscall.O_NONBLOCK` exists on darwin/linux/windows and is a no-op for regular files.
**Source:** 11-01-SUMMARY.md

### D-08 exact-case enforced against real directory entries
Candidate presence is verified via `os.ReadDir` entry names before reading — a plain open resolves case-insensitively on macOS APFS and Windows volumes.

**Rationale:** The exact-case contract (lowercase `readme.md` must NOT match README.md) is silently broken by case-insensitive filesystems; `os.ReadDir` name comparison is the only portable enforcement.
**Source:** 11-01-SUMMARY.md

### Presence-based match rule for Go
`detectGo` matches on go.mod existence, not parse success — a garbage go.mod still yields Language=Go with folder-base Name and empty Version.

**Rationale:** Ecosystem-correct reading (the go tool itself assumes go 1.16 when the directive is absent); D-disc-1. Fields degrade independently, never fabricated.
**Source:** 11-02-SUMMARY.md

### 7-position nil-slot registry literal (D-11)
`[]detectorFunc{detectGo, nil, nil, detectJS, nil, nil, detectPHP}` — positions are the contract; later phases edit slots in place, never reorder; nil entries skipped in `runDetectors` before the panic-recover path.

**Rationale:** Cascade precedence (Go@0 > JS@3 > PHP@6) must be tamper-proof across phases; `TestDetectorPositions` pins the layout each phase (T-11-07).
**Source:** 11-02-SUMMARY.md

### Version = raw go directive string, never normalized
`1.21rc1` stays `1.21rc1`; `toolchain go1.26.4` and `gopher 1.2` lines never match the `go` directive (exact first-token equality in parseGoMod).

**Rationale:** D-02/DATA-03 — the probe reports the toolchain floor verbatim; normalization would fabricate data (Pitfall 6).
**Source:** 11-02-SUMMARY.md

### isBadgeLine innermost-segment stripping
Repeatedly remove the innermost `[...](...)` segment (first `](`, nearest preceding `[`, matching `)`) so wrapped `[![..](..)](..)` badges strip to empty; a badge followed by real text is not skipped.

**Rationale:** D-disc-4 — inline markdown preserved verbatim except noise; pinned by the `badge_plus_text` matrix row.
**Source:** 11-01-SUMMARY.md

### D-06 as zero-value behavior for JS private manifests
No `Private` field in the decode struct — `private: true` without a version yields Version `""` by the zero value.

**Rationale:** A Private field would be dead code; the zero-value path is pinned by `TestProbe_JSPrivateNoVersion` (D-06 executed per RESEARCH).
**Source:** 11-03-SUMMARY.md

### D-07 full vendor/package name for PHP
composer.json Name = the FULL `acme/logger` string, never the last segment; absent version → `""` is the composer ecosystem norm (Packagist infers from tags).

**Rationale:** Consistent with Go module-path reporting; never fabricated (D-07).
**Source:** 11-03-SUMMARY.md

---

## Lessons

### `_js` filename suffix is a legacy GOARCH — silently excluded from every non-js build
`detect_js.go`/`detect_js_test.go` can never compile on darwin/linux/windows: Go's implicit file-constraint rules treat `_js` as GOARCH=js (verified via `go/build.MatchFile` + `go list IgnoredGoFiles`).

**Context:** Plan 11-03 Task 1 RED — "no tests to run" until root-caused. Renamed to `detect_javascript.go` (identifier `detectJS` unchanged); the plan-mandated filenames were unbuildable.
**Source:** 11-03-SUMMARY.md

### Stale position assertions must be updated in the same task as the registry edit
Filling slot 3 in Task 1 left the plan 11-02 version of TestDetectorPositions asserting `detectors[3]` is nil — Task 1's GREEN requirement could not hold with the stale assertion.

**Context:** Plan 11-03 deviation: the plan assigned registry_test.go to Task 2, but the intermediate update had to land in Task 1, then the final state in Task 2.
**Source:** 11-03-SUMMARY.md

### Literal tokens in doc comments trip literal acceptance greps
The readmeDescription doc comment contained `os.Open` — the plan's `grep -nE 'os\.Open|os\.ReadFile|io\.ReadAll'` matched the prose token although no direct I/O exists.

**Context:** Plan 11-01 compliance deviation. Fix: reword ("a plain open alone resolves case-insensitively…"). Recurred in phase 13 (4 more instances).
**Source:** 11-01-SUMMARY.md

### os.O_NONBLOCK is not exported by the os package
`os` exports only O_RDONLY/O_WRONLY/…; `syscall.O_NONBLOCK` is used instead — verified present on darwin, linux, and windows.

**Context:** Plan 11-01 Task 3 — a would-be compile error on the WR-01 gate; resolved by using the syscall constant.
**Source:** 11-01-SUMMARY.md

---

## Patterns

### Candidate-chain fallback
Ordered exact-case names probed via a shared capped reader, first readable wins, `""` when none (README.md → README.rst → README).

**When to use:** Any "find the best available file" heuristic where presence must be exact-case and reads must stay capped/never-fail.
**Source:** 11-01-SUMMARY.md

### Pinned behavior matrix as the contract
Every extraction threshold (badge, TOC, underline floor, heading, comment) has a named test row — the 15-row `TestFirstRealParagraph` matrix (later 21 rows) IS the spec.

**When to use:** Heuristic text parsing — a named row per skip rule prevents silent regression and documents intent.
**Source:** 11-01-SUMMARY.md

### Build-tagged test helper for Unix-only syscalls
`makeFIFO` lives in `manifest_fifo_test.go` (`//go:build !windows`, syscall.Mkfifo) with a Windows compile stub — the FIFO RED test hangs on a FIFO, never on the build.

**When to use:** Platform-specific test fixtures; keep the stub trivial so Windows CI compiles both branches.
**Source:** 11-01-SUMMARY.md

### Shared never-fail JSON helper
`readJSONManifest(folder, name, v any) bool` = readManifest → json.Unmarshal → bool; missing/oversized/malformed/type-mismatched all degrade to false so the cascade continues. Single `json.Unmarshal` call site in the package.

**When to use:** Any manifest parsing where decode failure must continue the cascade, never error (D-04/D-12).
**Source:** 11-03-SUMMARY.md

### Detector shape
readManifest → parse → ProjectData{Language} → DATA-02 Name chain (manifest → folder base) → D-02 raw Version → DATA-04 readmeDescription chain → `(data, true)`; every failure degrades to `(ProjectData{}, false)`.

**When to use:** All 7 detectors follow this exact shape (11-02/11-03, phases 12-13) — consistency makes edge coverage portable across detectors.
**Source:** 11-02-SUMMARY.md

### Hand-rolled line parser with exact first-token equality
TrimSpace + trailing-`//` strip + strings.Fields + exact first-token match — `gopher 1.2`, `modulex y`, `toolchain go1.26.4` never match the directives.

**When to use:** Format-adjacent parsing where prefix matching would fabricate; the parseGoMod grammar (quoted/backtick modules, first-occurrence-wins) is the reference.
**Source:** 11-02-SUMMARY.md

### Never leave a hanging test in the tree
The FIFO RED was demonstrated with `-timeout 5s` only (evidence in the feat commit body), then the gate made the row pass promptly — a test that hangs without the fix would poison every future run.

**When to use:** DoS-shaped REDs (hang/panic) — prove the RED with an explicit timeout, commit green.
**Source:** 11-01-SUMMARY.md

---

## Surprises

### APFS resolves case-insensitively
`TestReadmeDescription_Candidates/exact_case` failed on macOS: a lowercase `readme.md` was found by readManifest("README.md") — the D-08 exact-case contract would have been silently broken on macOS/Windows.

**Impact:** Required the os.ReadDir exact-name gate (deviation 1); Windows CI would have caught it later — the macOS-first dev loop caught it first.
**Source:** 11-01-SUMMARY.md

### os.Open on a FIFO blocks at open time
The plan's prescribed fix (stat after open) still hung — the stat never executes because the open itself never returns. The review's fix shape was insufficient for FIFOs.

**Impact:** The O_NONBLOCK amendment became the real fix; the fifo_blocks row passes in ~0s (pre-gate: `panic: test timed out after 5s`).
**Source:** 11-01-SUMMARY.md

### encoding/json rejects a leading UTF-8 BOM
The BOM strip in readManifest is load-bearing for JSON manifests — without it, a BOM'd package.json is malformed and the cascade skips a valid project (Pitfall 1, pinned by TestProbe_JSBOM).

**Impact:** Confirmed the Phase 10 BOM design; the strip is now documented as REQUIRED in json.go.
**Source:** 11-03-SUMMARY.md

### RED evidence can be behavioral, not just compile-fail
Task 1's RED was `expected: "Go", actual: "unknown"` — a behavioral end-to-end failure through the production stack (empty registry), not a compile error. Strongest RED form: the pipeline runs, the contract is absent.

**Impact:** Established the e2e-tracer RED pattern reused by every later detector plan (phases 12-13).
**Source:** 11-02-SUMMARY.md