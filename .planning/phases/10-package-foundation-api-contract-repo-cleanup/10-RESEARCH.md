# Phase 10: Package Foundation — API Contract + Repo Cleanup - Research

**Researched:** 2026-09-28
**Domain:** Go stdlib filesystem probing — public API contract, ordered detector registry, safe manifest I/O, repo cleanup
**Confidence:** HIGH

## Summary

Phase 10 builds the `project_probe/` package foundation — a new top-level directory at `github.com/guionardo/go/project_probe`, package name `projectprobe` (repo concatenation convention, `httptestmock` precedent) — with a never-fail `Probe(folder) (ProjectData, error)` contract, and deletes the deprecated `project_detector/` sample so `go build ./...` goes green. The public surface is small and locked by CONTEXT.md: `ProjectData{Folder, Language, Name, Version, Description}`, `type Language string` with 8 constants, 3 error sentinels (`ErrFolderNotFound`, `ErrNotDirectory`, `ErrPermissionDenied`) plus a plain-wrapped fallback, and the private ordered detector registry with the `(ProjectData, bool)` contract. Detector bodies arrive in phases 11–13; Phase 10 ships the empty registry plus the full probe pipeline, ignore list, and shared `readManifest`.

Three verified platform facts drive the design. (1) `os.ReadDir` returns entries **sorted by filename** — determinism is free and testable [VERIFIED: pkg.go.dev/os, src/os/dir.go]. (2) `os.ReadDir` on a file path returns `syscall.ENOTDIR` on Linux, macOS, and Windows — one call cleanly distinguishes "path is a file" from "folder missing" [VERIFIED: golang/go issue #75157]. (3) `errors.Is(err, fs.ErrNotExist)` matches BOTH ENOENT and ENOTDIR, so D-04's three-way mapping must be implemented with `errors.Is(err, syscall.ENOENT) / syscall.ENOTDIR / syscall.EACCES`, which behave identically on Windows (ERROR_FILE_NOT_FOUND and ERROR_PATH_NOT_FOUND → ENOENT, ERROR_DIRECTORY → ENOTDIR, ERROR_ACCESS_DENIED → EACCES) [VERIFIED: src/os/error_windows_test.go, os/error.go]. One planning-critical repo fact: `project_detector/` is **untracked in git** (0 tracked files) — FND-01's deletion is a filesystem `rm -rf`, not a `git rm`, and cannot produce a deletion diff/commit; the "own commit" clause must be interpreted as "deletion is its own isolated task with its own `go build ./...` verification".

**Primary recommendation:** `Probe` = clean-once (`filepath.Clean`) → `os.Stat` (dir check + errno mapping) → `os.ReadDir` (entries + errno mapping) → ignore-list content gate → ordered registry dispatch with per-detector panic recovery (cache `callSetter` precedent, cache/singleflight.go:77-88). Registry = package-level ordered slice of `detectorFunc` in a single `registry.go`; order locked by literal position; phases 11–13 extend that literal. `readManifest(folder, name string) ([]byte, bool)` = `os.Open` + `io.LimitReader(1<<20+1)` + `io.ReadAll` + `len > 1<<20 → not-ok` + `bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})`. Empty registry in Phase 10 → every content-bearing folder returns `LanguageUnknown`, nil error. Delete `project_detector/` as the phase's first task with `rm -rf` + `go build ./...` verification.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Error Contract
- **D-01:** Three exported sentinel errors: `ErrFolderNotFound` (missing folder), `ErrNotDirectory` (path is a file), `ErrPermissionDenied` (EACCES). Any other I/O failure returns a plain wrapped error. — **Reversibility:** one-way — exported sentinels are a published contract; removing them later breaks consumer `errors.Is` checks.
- **D-02:** Probe errors wrap the failing folder path: `fmt.Errorf("probe %s: %w", folder, err)` — callers get context AND `errors.Is` still matches. Matches repo convention (`config`, `fraction` sentinels).
- **D-03:** Manifest-read failures inside detectors (e.g., unreadable go.mod in a readable folder) degrade to Unknown / nil error — never surface as Probe errors. The never-fail contract reserves errors for hard folder-level I/O failures only. — **Reversibility:** costly — changing this later means detectors need an error return channel, touching the `(ProjectData, bool)` registry contract.
- **D-04:** Error mapping: `os.Stat`/`os.ReadDir` ENOENT → `ErrFolderNotFound`; ENOTDIR → `ErrNotDirectory`; EACCES **or EPERM** → `ErrPermissionDenied`; everything else → wrapped raw error. *(Quoted per the 2026-09-28 amendment in CONTEXT.md — EPERM added to the permission arm following assumption A2; the original research-time text mapped EACCES only.)*

#### Language Type
- **D-05:** `type Language string` with exported constants: `LanguageGo`, `LanguagePython`, `LanguageJavaScript`, `LanguageCSharp`, `LanguageRust`, `LanguageJava`, `LanguagePHP`, `LanguageUnknown`. — **Reversibility:** one-way — exported type + constants are a published contract for switch statements and JSON output.
- **D-06:** Constant string values are human-readable display values matching success criteria: `"Go"`, `"Python"`, `"JavaScript"`, `"C#/.NET"`, `"Rust"`, `"Java"`, `"PHP"`. Identifiers are PascalCase for Go code; values are what users see/log.
- **D-07:** `LanguageUnknown = "unknown"` — explicit sentinel value, NOT the zero-value `""`. Detectors always set Language explicitly; `"unknown"` reads better in logs/JSON than `""`. Extensible for v2 `typescript` value (REFN-01).

#### Folder Field Semantics
- **D-08:** `ProjectData.Folder` = `filepath.Clean(as-given)` — relative stays relative, never absolutized. Deterministic for temp-dir tests; Windows paths survive. — **Reversibility:** costly — absolutizing later changes returned values for every relative probe.
- **D-09:** Clean once at entry; the cleaned path is reused for `os.Stat`/`os.ReadDir`, error wrapping, and the Folder field — one canonical path, errors and data always agree.

#### Ignore List (ROBT-01)
- **D-10:** Default ignore list (unexported): the 5 criteria-named dirs `node_modules`, `vendor`, `.git`, `dist`, `.idea` PLUS common ecosystem noise `.venv`, `venv`, `__pycache__`, `target`, `build`, `bin`, `obj`, `.cache`.
- **D-11:** Exact-case name matching on all platforms — no case-insensitivity branching. Windows-created `Node_Modules` edge documented as acceptable.
- **D-12:** Root-scoped only (consistent with DETC-01); ignore list only affects the "folder contains nothing but ignored dirs" case → `LanguageUnknown`, nil error.

#### Registry & Shared Helpers (FND-03, DETC-01, ROBT-02) — agent discretion
- Ordered private detector registry with `func(folder string) (ProjectData, bool)` contract — first match wins. Detector order from research: Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP (empty registry in Phase 10; entries added phases 11-13).
- Shared `readManifest` — 1 MB size cap, BOM strip, no panics on pathological input (precedent: singleflight `PanicError` recovery pattern from v1.6 spikes).

### the agent's Discretion
- Registry data structure (slice of funcs vs. small registry type), readManifest internals, file layout within `project_probe/` (snake_case, doc.go), ignore-list data structure (map vs slice) — follow repo conventions (CONVENTIONS.md).

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. (Framework detection, TOML dependency, monorepo detection already tracked in v2 requirements/Out of Scope.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FND-01 | Delete deprecated `project_detector/` sample in its own commit — it fails `go build ./...` (missing gs-dev, BurntSushi go.sum) | Deletion = `rm -rf` (directory is **untracked**, 0 files in git index — verified this session); 3 build errors reproduced verbatim (BurntSushi go.sum, gs-dev console, gs-dev files); `go build ./...` after removal is the acceptance probe; see Runtime State Inventory |
| FND-02 | `project_probe/` skeleton with `doc.go` documenting the never-fail contract (error reserved for hard I/O failures; Unknown = nil error) | Package-doc format precedent: `cache/doc.go` (`// Package cache provides …` + Usage code block); godoclint enforces package comments (CONVENTIONS.md §Comments) |
| FND-03 | `Probe(folder) (ProjectData, error)` entry point backed by private ordered detector registry with `(ProjectData, bool)` contract | Pattern 1 + Pattern 2 below; `(ProjectData, bool)` means no error channel anywhere — `readManifest` returns `([]byte, bool)` to stay consistent (D-03) |
| DETC-01 | Ordered manifest-first cascade, first match wins, root-scoped only (no subdir probing) | `os.ReadDir` sorted guarantee makes iteration order deterministic; registry slice literal order = cascade order; `readManifest` joins only `folder + constant name` → root-scoped by construction |
| DETC-09 | Unknown folders → `LanguageUnknown` value, nil error | `Probe` returns `ProjectData{Folder: clean, Language: LanguageUnknown}` when the content gate finds nothing and the (empty) registry finds no match |
| DATA-01 | `ProjectData{Folder, Language, Name, Version, Description}` with independent per-field fallbacks | Model shape locked by D-05/D-08; `Probe` owns `Folder` + `Language` defaults; detectors (11–13) fill the rest — merge rule: only the winning detector's fields override the Unknown defaults |
| ROBT-01 | Ignore-list hygiene (vendored/build/IDE dirs) so they never masquerade as markers | D-10 list (13 entries, exact-case); `hasContent` gate before cascade; old sample's `ignoreDirs` map (project_detector/detector.go:15-19) is the shape precedent, its 13-entry map is nearly identical to D-10 |
| ROBT-02 | Shared `readManifest` helper — size cap + BOM strip | Pattern 4: `io.LimitReader(1<<20+1)` + truncation probe; `bytes.TrimPrefix` EF BB BF; `([]byte, bool)` signature keeps the never-fail contract |
| ROBT-04 | Stdlib-only imports; `path/filepath` not `path`; Windows-safe | Old sample's `path.Base` (project_detector/detector.go:56) is the anti-pattern being corrected; `filepath.Clean/Join/Base` everywhere; Windows errno mapping verified (see Summary) |
</phase_requirements>

## Architectural Responsibility Map

This repo is a single-package-per-directory stdlib library — there are no browser/SSR/CDN/database tiers. Every capability in this phase belongs to the package core tier; the map exists to prevent a planner from pushing library logic into tests, docs, or the `cmd/` tree.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Folder validation + error mapping | API/Backend (package core) | — | `os.Stat`/`os.ReadDir` + errno→sentinel mapping is public-API behavior (D-04); must live in `probe.go`, not in detectors |
| Probe orchestration (clean-once, content gate, dispatch) | API/Backend (package core) | — | D-09 single canonical path; one `ReadDir` per probe shared by the gate and the registry |
| Detection cascade | API/Backend (package core) | — | Private `registry.go` — order, first-match, panic-recovery all live here |
| Manifest I/O safety | API/Backend (package core) | — | `manifest.go` shared helper consumed by detectors in phases 11–13 |
| Ignore-list hygiene | API/Backend (package core) | — | `ignore.go` gate runs before the cascade (D-12) |
| Docs (never-fail contract) | Docs | — | `doc.go` + `example_test.go` are the only non-code deliverables (FND-02) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `os` | 1.26.4 (go.mod) | `Stat`, `ReadDir`, `Open` — all folder I/O | Stdlib-only mandate (ROBT-04); verified: ReadDir sorts entries, returns ENOTDIR on files across all 3 CI platforms |
| Go stdlib `path/filepath` | 1.26.4 | `Clean`, `Join` — Windows-safe paths | ROBT-04; replaces old sample's `path` package |
| Go stdlib `io` | 1.26.4 | `LimitReader` + `ReadAll` for the 1 MB cap | Verified: LimitReader stops with EOF after n bytes (pkg.go.dev/io) |
| Go stdlib `bytes` | 1.26.4 | `TrimPrefix` UTF-8 BOM strip | Simplest correct BOM removal; no state machine |
| Go stdlib `errors` + `fmt` | 1.26.4 | Sentinel errors + `%w` wrapping | Repo convention (CONVENTIONS.md §Error Handling; fraction/cache precedents) and Go 1.13 errors doctrine |
| Go stdlib `syscall` | 1.26.4 | `ENOENT`/`ENOTDIR`/`EACCES` errno matching | The only cross-platform way to implement D-04's three-way mapping (fs.ErrNotExist conflates ENOENT+ENOTDIR) |
| `github.com/stretchr/testify` | v1.11.1 | `assert`/`require` in tests | Existing module dependency [VERIFIED: go.mod:13]; AGENTS.md mandates testify |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `path_tools` (`DirExists`, `FileExists`) | in-repo | Pre-built existence helpers | Optional inside detector bodies (phases 11–13). NOT needed in Phase 10 — `readManifest` does its own `os.Open` and needs the stat/error detail anyway |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `errors.Is(err, syscall.ENOENT/ENOTDIR/EACCES)` | `errors.Is(err, fs.ErrNotExist)` only | `fs.ErrNotExist` matches ENOENT **and** ENOTDIR — cannot implement D-04's three-way split [VERIFIED: src/os/error.go + issue #75157] |
| `io.LimitReader(f, 1<<20+1)` + `len > cap` probe | `os.Stat` size check before `os.ReadFile` | Stat-then-read has a TOCTOU window and duplicates `os.ReadFile`'s internal stat; the limit+1 probe is exact and race-free |
| Package-level slice of funcs | Small `registry` struct type | Slice matches the old sample's shape (detector.go:14) and repo minimalism; a struct adds ceremony. Struct only wins if parallel tests need isolated state — solve with "no `t.Parallel()` on registry-mutating tests" instead (AGENTS.md rule) |
| `([]byte, bool)` readManifest | `([]byte, error)` | An error channel at manifest level would leak into detector dispatch and contradict D-03's never-fail design; `bool` keeps `(ProjectData, bool)` consistent |

**Installation:** none — stdlib only; testify already present.
**Version verification:** `go.mod` declares `go 1.26.4` [VERIFIED: go.mod:3] and `github.com/stretchr/testify v1.11.1` [VERIFIED: go.mod:13]. Local toolchain is go1.27.0 (darwin/arm64) — forward-compatible; CI pins via `go-version-file: go.mod` [VERIFIED: .github/workflows/go.yml].

## Package Legitimacy Audit

> Gate outcome: **N/A — no external packages installed by this phase.** All runtime code is Go stdlib; `testify` (test-only) is an established module dependency, not a new install.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|----------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/stretchr/testify` v1.11.1 | Go modules | ~10 yrs | widely used | stretchr/testify | OK (existing dep) | Approved — already in go.mod, no install action |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
*Note: the go ecosystem is not covered by the package-legitimacy seam (returns empty); the gate is satisfied because zero installs occur in this phase — the planner must not add any `go get` task.*

## Architecture Patterns

### System Architecture Diagram

```
                        ┌─────────────────────────────────────────────┐
                        │              projectprobe.Probe(folder)      │
                        └──────────────────────┬──────────────────────┘
                                               │ 1. filepath.Clean(folder)  (D-08/D-09)
                                               ▼
                                   ┌───────────────────────┐
                                   │   os.Stat(clean)      │◄── err? ──┐
                                   └───────────┬───────────┘            │
                                               │ !info.IsDir()          │ mapFolderError(clean, err):
                                               ▼                        │   errors.Is ENOTDIR ─► ErrNotDirectory
                                   ErrNotDirectory ────────────────────►│   errors.Is ENOENT ─► ErrFolderNotFound
                                               │                        │   errors.Is EACCES ─► ErrPermissionDenied
                                               ▼                        │   else ─────────────► fmt.Errorf("probe %s: %w", clean, err)
                                   ┌───────────────────────┐            │
                                   │   os.ReadDir(clean)   │◄── err? ───┘
                                   └───────────┬───────────┘
                                               │ entries (sorted by filename — deterministic)
                                               ▼
                                   ┌───────────────────────┐     empty / only-ignored-dirs
                                   │  hasContent(entries)  │─────────────────────────────►  ProjectData{Folder: clean, Language: LanguageUnknown}, nil
                                   └───────────┬───────────┘
                                               │ has content (any file or non-ignored dir)
                                               ▼
                                   ┌───────────────────────────────────────────┐
                                   │  runDetectors(clean)  (registry.go)       │
                                   │  for i, fn := range detectors:            │
                                   │      defer-recover around fn(clean)       │
                                   │      if pd, ok := fn(clean); ok:          │
                                   │          merge Language/Name/Version/     │
                                   │          Description into result; break   │
                                   └───────────────────────┬───────────────────┘
                                                           │
                              no match (empty registry     ▼
                              in Phase 10) ─────────►  ProjectData{Folder: clean, Language: LanguageUnknown}, nil
```

Entry point: `Probe`. Processing: clean → stat → readdir → content gate → registry. Branching: three sentinel error exits, one Unknown exit. External dependencies: filesystem only (no network, no build tools, no symlink walks — ROBT-05 anti-features). Every exit path returns either a wrapped sentinel error (hard I/O) or `LanguageUnknown` with nil error (content) — the never-fail contract.

### Recommended Project Structure

```
project_probe/
├── doc.go              # Package docs — never-fail contract (FND-02); // Package projectprobe provides …
├── project.go          # ProjectData struct + Language type + 8 constants (D-05..D-08); decorder: type before const
├── errors.go           # 3 sentinels (var block) + mapFolderError (D-01/D-02/D-04)
├── probe.go            # Probe entry point — clean-once, stat, readdir, content gate, dispatch (D-09)
├── registry.go         # type detectorFunc; var detectors = []detectorFunc{}; runDetectors with panic-recover
├── ignore.go           # var ignoreDirs map[string]struct{} (D-10); isIgnoredDir; hasContent (D-12)
├── manifest.go         # maxManifestSize = 1 << 20; readManifest (ROBT-02)
├── probe_test.go       # table-driven Probe tests (internal package projectprobe — needs unexported access)
├── errors_test.go      # errno mapping tests (ENOENT/ENOTDIR/EACCES table)
├── registry_test.go    # order, first-match, empty-registry, panic-recovery tests (NO t.Parallel — mutates global)
├── manifest_test.go    # cap boundary, BOM, missing file
└── example_test.go     # ExampleProbe — prints only deterministic fields (Language), never the temp path
```

Conventions honored: snake_case files [VERIFIED: STRUCTURE.md §Naming Conventions], package-name concatenation `projectprobe` (`httptestmock` precedent [VERIFIED: STRUCTURE.md:237]), `example_test.go` per package [VERIFIED: STRUCTURE.md:217], internal test packages allowed (`config` precedent [VERIFIED: TESTING.md]).

### Pattern 1: Never-fail Probe — the error/Unknown boundary

**What:** `Probe(folder) (ProjectData, error)` returns a sentinel-wrapped error ONLY for hard folder-level I/O failures (missing, not-a-directory, permission); everything content-related (empty folder, ignored-only folder, unrecognized content, unreadable *manifests*) returns `ProjectData` with `LanguageUnknown` and nil error. The registry contract `(ProjectData, bool)` makes error-as-control-flow impossible in detectors — this is the fixed design that replaces the old sample's `DetectorFunc func(folder string) (projectData *ProjectData, err error)` + `errors.New("folder is empty")` pattern [VERIFIED: project_detector/detector.go:11,45] and its `err == nil` success check [VERIFIED: project_detector/detector.go:49].

**When to use:** This phase — the exact locked contract. The empty-string edge (`filepath.Clean("") == "."` would probe the cwd) is worth an explicit guard (see Open Questions OQ-1).

### Pattern 2: Ordered detector registry with panic-recover dispatch

**What:** A package-level ordered slice plus a dispatch loop that recovers panics per detector. `detectors = []detectorFunc{}` in `registry.go` — **empty in Phase 10**, extended by phases 11–13 *in the same file* so order stays explicit (Go `init()` ordering across files is filename-lexical — never rely on it for cascade order).

**Example (design skeleton — new code, follows cache/callSetter precedent [VERIFIED: cache/singleflight.go:77-88]):**

```go
// detectorFunc reports a project match for folder. It must never return an
// error: a non-match is (ProjectData{}, false). A recovered panic is treated
// as a non-match so Probe can never fail on content (D-03).
type detectorFunc func(folder string) (ProjectData, bool)

// detectors is the ordered cascade. Order is the contract (DETC-01): first
// match wins. Phases 11-13 append entries HERE, in cascade position.
var detectors = []detectorFunc{} // Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP

func runDetectors(folder string) (pd ProjectData, ok bool) {
    for _, fn := range detectors {
        if pd, ok := callDetector(fn, folder); ok {
            return pd, true
        }
    }
    return ProjectData{}, false
}

// callDetector runs one detector with panic recovery, mirroring
// cache.SingleflightGetOrSet.callSetter: a recovered panic never escapes and
// the detector counts as a non-match.
func callDetector(fn detectorFunc, folder string) (pd ProjectData, ok bool) {
    defer func() {
        if r := recover(); r != nil {
            // log at debug level; treat as non-match (never re-panic)
            pd, ok = ProjectData{}, false
        }
    }()
    return fn(folder)
}
```

**When to use:** Any first-match-wins cascade with untrusted input parsing arriving later (11–13). The recover choke point here protects all future detector bodies — strictly more protective than recover-inside-`readManifest` (which is pure I/O and cannot panic).

### Pattern 3: Errno-based sentinel mapping (D-04)

**What:** Map `os.Stat`/`os.ReadDir` errors via `errors.Is` against `syscall` errnos, not `fs.ErrNotExist`. Verified cross-platform behavior:

- `errors.Is(err, syscall.ENOENT)` → `ErrFolderNotFound` — on Windows both ERROR_FILE_NOT_FOUND and ERROR_PATH_NOT_FOUND count as not-exist [VERIFIED: src/os/error_windows_test.go — `isnot: true` for both].
- `errors.Is(err, syscall.ENOTDIR)` → `ErrNotDirectory` — `os.ReadDir` on a file returns `syscall.ENOTDIR` on Linux, macOS, and Windows [VERIFIED: golang/go#75157]; Windows ERROR_DIRECTORY maps to ENOTDIR.
- `errors.Is(err, syscall.EACCES)` → `ErrPermissionDenied` — ERROR_ACCESS_DENIED counts as permission [VERIFIED: src/os/error_windows_test.go — `want: true`]. Consider also matching `syscall.EPERM` (some Unix filesystems report EPERM for permission failures).
- else → `fmt.Errorf("probe %s: %w", clean, err)` raw wrap (D-02 shape, `errors.Is` still traverses).

**When to use:** The exact D-04 contract. Note the mapped sentinel errors discard the raw OS error by design — per Go 1.13 guidance, don't expose underlying OS error types unless committing to them as API [CITED: go.dev/blog/go1.13-errors].

### Pattern 4: Size-capped, BOM-stripped readManifest

**What:** A shared helper that reads a root-level manifest safely. Size cap via `io.LimitReader(limit+1)` with a truncation probe (LimitReader silently stops with EOF at n bytes [VERIFIED: pkg.go.dev/io, src/io/io.go] — `len > limit` after `ReadAll` means the file exceeded the cap). UTF-8 BOM strip via `bytes.TrimPrefix`.

```go
// maxManifestSize caps manifest reads at 1 MB (ROBT-02). Pathological inputs
// (giant or binary files) yield a non-match, never a panic or OOM.
const maxManifestSize = 1 << 20 // 1 MB

var utf8BOM = []byte{0xEF, 0xBB, 0xBF} // UTF-8 encoded U+FEFF

// readManifest returns the BOM-stripped content of <folder>/<name> when it
// exists, is a regular file, and is at most maxManifestSize bytes. Any other
// outcome returns ok=false so callers degrade to Unknown (D-03).
func readManifest(folder, name string) (content []byte, ok bool) {
    f, err := os.Open(filepath.Join(folder, name))
    if err != nil {
        return nil, false
    }
    defer f.Close()

    content, err = io.ReadAll(io.LimitReader(f, maxManifestSize+1))
    if err != nil || len(content) > maxManifestSize {
        return nil, false
    }
    return bytes.TrimPrefix(content, utf8BOM), true
}
```

`name` is always a compile-time constant at call sites (phases 11–13), so `filepath.Join` cannot traverse — root-scoped by construction (DETC-01). If gosec flags the join, annotate `// #nosec G304` with that rationale [CONVENTIONS.md precedent].

### Anti-Patterns to Avoid

- **`fs.ErrNotExist` for the three-way split:** matches ENOENT *and* ENOTDIR — `errors.Is(err, fs.ErrNotExist)` cannot tell "missing folder" from "path is a file"; D-04 requires syscall errnos [VERIFIED].
- **Error-as-control-flow detectors (old sample's shape):** `DetectorFunc ... err error` + `if err == nil` success check [VERIFIED: project_detector/detector.go:11,49] made the old cascade treat "folder is empty" as a returned error and Unknown as `errors.New(...)` — exactly the contract FND-03/D-03 replace.
- **`path` package on Windows:** the old sample used `path.Base(assertedFolder)` [VERIFIED: project_detector/detector.go:56] — ROBT-04 bans it; use `filepath`.
- **`t.Parallel()` on registry-mutating tests:** appending test detectors to the package-level slice races under parallel subtests; AGENTS.md explicitly warns about global-state tests (mid/`collectFuncs` precedent).
- **`os.ReadFile` without a cap:** unbounded read of a pathological manifest defeats ROBT-02; also `os.ReadFile` internally stats then reads the whole file — the LimitReader probe is the cap.
- **Asserting `entries == nil` vs `[]`:** Windows returned a non-nil empty slice from ReadDir on a file pre-Go 1.26 (fixed by CL 699375 but not guaranteed across versions) [VERIFIED: golang/go#75157]; assert with `require.Empty`/`assert.Len`, never nil-ness.
- **Printing absolute temp paths in examples:** `ExampleProbe` must print only deterministic fields (`data.Language`, `err == nil`) or the example breaks on every machine.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform error classification | Platform-detection branches (`runtime.GOOS` + raw error inspection) | `errors.Is(err, syscall.ENOENT/ENOTDIR/EACCES)` | Go's os package already maps Windows ERROR_* to the same errnos; verified in this research — one code path, 3 CI platforms |
| Sorted directory iteration | Manual sort of `os.ReadDir` results | `os.ReadDir` (sorted by filename, guaranteed) | Sorting is in the stdlib contract; re-sorting adds code for zero benefit |
| File size cap | Manual `ReadFull` loop with counter | `io.LimitReader` + truncation probe | LimitReader is the stdlib primitive; hand-rolled counters are where off-by-one truncation bugs live |
| BOM removal | Hand-rolled leading-byte state machine | `bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})` | Three bytes, one call, no edge cases |
| Test assertions | Hand-rolled if/else compare helpers | `testify` `assert`/`require` (already in go.mod) | Repo mandate (AGENTS.md); `testifylint` enforces idioms |
| Panic containment | Trust detector bodies not to panic | Named-return + `defer recover()` dispatch wrapper | Proven repo pattern (cache `callSetter`); future detectors parse untrusted input |

**Key insight:** every "hard" problem in this phase (error mapping, determinism, size caps, BOM, panic safety) has a verified stdlib or in-repo precedent. Hand-rolling any of them trades verified behavior for novel bugs on exactly the pathological-input paths the phase exists to survive.

## Runtime State Inventory

> Required for this phase: it deletes a directory (FND-01). The canonical question — *after every file in the repo is updated, what runtime systems still hold the old state?* — answered per category.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `project_detector`/`project_probe` have no databases, caches, or datastores | — |
| Live service config | None — no external services configure these packages | — |
| OS-registered state | None — no schedulers, daemons, or registries reference `project_detector` | — |
| Secrets/env vars | None — no env vars or secret keys reference either package | — |
| Build artifacts | `project_detector/` on disk: 11 files, **0 tracked in git** (`git ls-files project_detector` → empty; `git status` → `?? project_detector/`) | **Code edit** (filesystem removal): `rm -rf project_detector/` — NOT `git rm` (would fail: "pathspec did not match"). `go.mod`/`go.sum` need **no** cleanup: the 3 build errors are *missing* gs-dev and BurntSushi entries, i.e. they were never added [VERIFIED: reproduced build errors]. |

**Nothing-found note:** categories 1–4 verified explicitly (empty) — the only state is the untracked directory itself.

## Common Pitfalls

### Pitfall 1: `git rm` on the untracked `project_detector/` fails
**What goes wrong:** Executor runs `git rm -r project_detector/` per "delete in its own commit" and gets `fatal: pathspec 'project_detector' did not match any files`; the deletion never happens.
**Why it happens:** The directory has never been committed — 0 tracked files (verified this session).
**How to avoid:** Task FND-01 = `rm -rf project_detector/` then `go build ./...` (must pass). Because git cannot represent the deletion of untracked files, the "own commit" clause is satisfied by sequencing: deletion is task #1, isolated from package-creation work, with its own verification step. Optionally record the untracked reality in the phase's first commit message.
**Warning signs:** `git status` shows `?? project_detector/` before deletion, or `git rm` errors.

### Pitfall 2: `fs.ErrNotExist` cannot split ENOENT from ENOTDIR
**What goes wrong:** `errors.Is(err, fs.ErrNotExist)` is true for both a missing folder and a file-in-path — D-04's `ErrNotDirectory` never fires; "path is a file" tests fail.
**Why it happens:** `os` maps ENOENT and ENOTDIR to `fs.ErrNotExist` [VERIFIED: src/os/error.go, isErrNotExist semantics].
**How to avoid:** Match `syscall.ENOENT` / `syscall.ENOTDIR` / `syscall.EACCES` explicitly (Pattern 3). Order the checks ENOENT → ENOTDIR → EACCES → fallback.
**Warning signs:** A test probing a file path returns `ErrFolderNotFound` instead of `ErrNotDirectory`.

### Pitfall 3: Windows-only test failures on the permission-denied case
**What goes wrong:** `chmod 000` + probe → `ErrPermissionDenied` passes on macOS but fails on CI's windows-latest (or fails on Unix when run as root, which ignores modes).
**Why it happens:** Windows ACLs don't map `chmod 000` to a guaranteed EACCES on `os.Stat`; root bypasses permission checks entirely.
**How to avoid:** Guard with `if runtime.GOOS == "windows" || os.Geteuid() == 0 { t.Skip(...) }`; assert the EACCES mapping itself via a synthetic `&os.PathError{Err: syscall.EACCES}` through `mapFolderError` instead of the filesystem. Keep the filesystem permission test OS-gated.
**Warning signs:** CI windows job red on the permission test; local mac run green.

### Pitfall 4: Registry tests race under `t.Parallel()`
**What goes wrong:** Tests appending fake detectors to the package-level slice run concurrently with each other and with the cascade test; flaky order/panic assertions.
**Why it happens:** Global mutable state + parallel subtests — the exact hazard AGENTS.md documents for `mid/` `collectFuncs`.
**How to avoid:** All tests that write `detectors` (append/replace) run WITHOUT `t.Parallel()`; read-only tests (empty-registry probe) may run in parallel. Simplest: keep the whole `registry_test.go` non-parallel.
**Warning signs:** `go test -race` reports a race in `registry_test.go`.

### Pitfall 5: Coverage gate fails on the new package
**What goes wrong:** `make coverage-quick` (mandatory before every commit, AGENTS.md) fails: `project_probe` has no override in `.testcoverage-quick.yml` — it must meet package ≥80%, file ≥70%, and contribute to total ≥75% [VERIFIED: .testcoverage-quick.yml].
**Why it happens:** The phase's own tests cover `Probe` happy paths but miss the dispatch/panic/errno branches — easy to leave uncovered with an empty registry.
**How to avoid:** Test the unexported surface directly (internal `package projectprobe`): `runDetectors` with injected fake detectors (order, first-match, panic-recover, empty), `mapFolderError` with synthetic PathErrors, `readManifest` boundary cases (missing, >1 MB, exactly 1 MB, BOM), `hasContent` with ignored-only mixes. `example_test.go` adds a few percent for free.
**Warning signs:** `make coverage-quick` reports `project_probe` below 80% / a file below 70%.

### Pitfall 6: `Probe("")` silently probes the current directory
**What goes wrong:** `filepath.Clean("") == "."` — an empty folder argument stats and reads the cwd instead of failing; surprising results in tests and consumers.
**Why it happens:** `filepath.Clean`'s documented empty-input behavior.
**How to avoid:** Decide explicitly (see Open Question OQ-1). Recommended: reject `folder == ""` with `ErrFolderNotFound` before cleaning — a cheap guard that makes the API honest.
**Warning signs:** A test asserting `Probe("")` returns an error fails because it returns the cwd's result.

## Code Examples

Verified patterns from official sources plus repo precedents:

### Probe entry point (design skeleton — new code; every value derives from locked decisions D-01..D-12)

```go
// Probe inspects folder and returns its project data.
//
// Probe never fails on content: an empty, unrecognized, or ignore-list-only
// folder returns ProjectData with LanguageUnknown and a nil error. Errors are
// reserved for hard folder-level I/O failures — ErrFolderNotFound,
// ErrNotDirectory, ErrPermissionDenied — always wrapped with the failing
// path (D-02). The returned Folder field is filepath.Clean(as-given);
// relative input stays relative (D-08).
func Probe(folder string) (ProjectData, error) {
    if folder == "" { // OQ-1: recommended guard
        return ProjectData{}, fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
    }
    clean := filepath.Clean(folder) // D-09: one canonical path

    info, err := os.Stat(clean)
    if err != nil {
        return ProjectData{}, mapFolderError(clean, err)
    }
    if !info.IsDir() {
        return ProjectData{}, fmt.Errorf("probe %s: %w", clean, ErrNotDirectory)
    }
    entries, err := os.ReadDir(clean)
    if err != nil {
        return ProjectData{}, mapFolderError(clean, err)
    }

    data := ProjectData{Folder: clean, Language: LanguageUnknown}
    if !hasContent(entries) { // empty or only ignored dirs (D-12, DETC-09)
        return data, nil
    }
    if pd, ok := runDetectors(clean); ok { // empty registry in Phase 10
        data.Language = pd.Language
        data.Name = pd.Name
        data.Version = pd.Version
        data.Description = pd.Description
    }
    return data, nil
}
```

### Error mapper (Pattern 3)

```go
// mapFolderError maps stat/readdir failures onto the sentinel contract (D-04).
func mapFolderError(folder string, err error) error {
    switch {
    case errors.Is(err, syscall.ENOENT):
        return fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
    case errors.Is(err, syscall.ENOTDIR):
        return fmt.Errorf("probe %s: %w", folder, ErrNotDirectory)
    case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
        return fmt.Errorf("probe %s: %w", folder, ErrPermissionDenied)
    default:
        return fmt.Errorf("probe %s: %w", folder, err)
    }
}
```

### Table-driven Probe tests (repo pattern — TESTING.md §Test Structure)

```go
func TestProbe(t *testing.T) { //nolint:funlen
    t.Parallel()

    tests := []struct {
        name    string
        setup   func(t *testing.T) string // returns a path
        wantErr error
        want    Language
    }{
        {"missing_folder", func(t *testing.T) string { return filepath.Join(t.TempDir(), "nope") }, ErrFolderNotFound, ""},
        {"path_is_a_file", func(t *testing.T) string { return writeTempFile(t) }, ErrNotDirectory, ""},
        {"empty_folder", func(t *testing.T) string { return t.TempDir() }, nil, LanguageUnknown},
        {"ignored_only", func(t *testing.T) string { return mkdirs(t, "node_modules", ".git", "dist") }, nil, LanguageUnknown},
        {"unknown_content", func(t *testing.T) string { return writeTempFileInDir(t) }, nil, LanguageUnknown},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            folder := tt.setup(t)
            data, err := probe.Probe(folder)
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, data.Language)
            assert.Equal(t, filepath.Clean(folder), data.Folder) // D-08/D-09
        })
    }
}
```

### Determinism + Folder-cleanliness assertion (Success Criterion 3)

```go
func TestProbe_Deterministic(t *testing.T) {
    t.Parallel()
    folder := mkProjectDir(t) // e.g. "a/../proj" to exercise Clean
    first, err := probe.Probe(folder)
    require.NoError(t, err)
    second, err := probe.Probe(folder)
    require.NoError(t, err)
    assert.Equal(t, first, second)               // identical repeated results
    assert.Equal(t, filepath.Clean(folder), first.Folder) // clean-once, relative stays relative
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `project_detector` — error-as-control-flow, `errors.New` for Unknown, `path` package, external gs-dev/BurntSushi deps [VERIFIED: project_detector/detector.go:22-60] | `project_probe` — never-fail `(ProjectData, bool)` contract, sentinels + `%w`, `path/filepath`, stdlib-only | This phase (v1.7) | `go build ./...` green again; public contract usable by consumers; Windows-safe |
| `os.ReadDir` on a file: non-nil empty slice on Windows (pre-1.22 and 1.22–1.25 regression window) | Consistent `nil, syscall.ENOTDIR` across platforms (CL 699375, Go 1.26) | Go 1.26 | Tests must not assert nil-ness of entries; `errors.Is(err, syscall.ENOTDIR)` is the reliable discriminator |
| `io.LimitedReader` returns plain EOF at the cap | `LimitedReader.Err` field to signal truncation (Go PR #76156, fixes #51115) | Go 1.27 (unreleased at research time) | Do NOT depend on it — the limit+1 probe works on 1.26.4; revisit only if the module's minimum Go version rises |

**Deprecated/outdated:**
- `os.IsNotExist`/`os.IsPermission` (predate `errors.Is`, only unwrap legacy os types) — docs explicitly say "New code should use `errors.Is(err, fs.ErrNotExist)`" [VERIFIED: src/os/error.go]; for the three-way split use syscall errnos as shown.
- `ioutil.ReadFile` — superseded by `os.ReadFile`; neither is capped, so the LimitReader pattern stands.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `syscall.ENOENT`/`syscall.ENOTDIR`/`syscall.EACCES` constants exist on Windows with the ERROR_* mappings described (ERROR_DIRECTORY → ENOTDIR) | Pattern 3 | The not-exist/permission sides are verified via `error_windows_test.go`; ENOTDIR-on-file behavior verified via issue #75157. Residual risk: an exotic Windows path edge (e.g. ERROR_BAD_NETPATH on UNC drives, which Go maps to not-exist [VERIFIED test]) lands in the fallback wrap instead of a sentinel — acceptable per D-04's "everything else → wrapped raw error" |
| A2 | Adding `syscall.EPERM` to the EACCES check is beneficial on Unix (some filesystems report EPERM for permission failures) | Pattern 3 | Harmless if wrong (EPERM also exists on Windows, mapping to ERROR_INVALID_FUNCTION — never hit by Stat/ReadDir); worst case an exotic permission error maps to the raw fallback |
| A3 | UTF-8 BOM byte sequence is `EF BB BF` (U+FEFF encoded) | Pattern 4 | Fixed Unicode fact; wrong only for UTF-16 manifests, which the criteria do not require (JSON/TOML/XML with UTF-16 BOMs would fail parsing in later phases — acceptable, documented as a future extension) |
| A4 | `filepath.Clean("")` returns `"."` | Pitfall 6 / OQ-1 | Standard Go behavior; the recommended `folder == ""` guard makes the assumption irrelevant to behavior |
| A5 | The panic-recover belongs at registry dispatch, not inside `readManifest` | Pattern 2 | If a future detector body panics *inside* readManifest's call stack, the dispatch recover still catches it — strictly more protective; no scenario where readManifest-local recovery is needed |
| A6 | `.testcoverage-quick.yml` thresholds apply to the new package as-is (no override added) | Pitfall 5 | Verified file contents (pkg 80 / file 70 / total 75); if a future phase adds an override for `project_probe`, the threshold changes — Phase 10 must pass without one |
| A7 | Detector-merge rule: winning detector's `Language/Name/Version/Description` override the Unknown defaults; `Folder` always owned by `Probe` | Pattern 1 | Not explicitly locked in CONTEXT.md; consistent with D-08/D-09 (one canonical path) and DATA-01's per-field fallbacks; cheap to adjust in 11–13 |

## Open Questions (RESOLVED)

1. **`Probe("")` — error or cwd probe?** — RESOLVED (2026-09-28): guard `folder == ""` → `ErrFolderNotFound`.
   - What we know: `filepath.Clean("") == "."`, so an unguarded Probe would stat/read the current directory; CONTEXT.md D-08/D-09 don't address empty input.
   - What's unclear: whether the empty-string case should be `ErrFolderNotFound` (recommended — honest API, avoids silent cwd probing) or documented-as-probing-cwd.
   - Recommendation: guard `folder == ""` → `ErrFolderNotFound` (shown in the Probe skeleton); planner should add one table row + one test. If the user prefers literal D-08 semantics, drop the guard and add a doc sentence — either way, decide now, not at execution.
   - **Resolution adopted:** the recommended guard. Plan 10-02 Task 1 adds the `empty_string` Probe row (`""` → `require.ErrorIs ErrFolderNotFound`), Task 2 implements the guard in `probe.go` before cleaning, and Task 3 documents it in `doc.go` ("never probes the cwd").

2. **FND-01 "in its own commit" with an untracked directory** — RESOLVED (2026-09-28): task-isolation + `go build ./...` verification satisfies the clause; no empty commit is created.
   - What we know: `project_detector/` has 0 tracked files; git cannot produce a deletion commit; the build failure is local-only (CI clones never contained the dir).
   - What's unclear: whether the phase must still produce a commit attributable to FND-01.
   - Recommendation: treat FND-01 as satisfied by (a) `rm -rf` as the phase's first task, (b) `go build ./...` green as its acceptance check, (c) the untracked reality noted in the phase's first commit message. No empty commits (repo rule). If strict commit separation is required, a `chore:` commit carrying the first `project_probe` skeleton immediately after the deletion task is the closest honest representation — planner's call; do not force a no-op commit.
   - **Resolution adopted:** the recommendation. Plan 10-01 Task 1 performs the `rm -rf` as the phase's first isolated task with `go build ./...` as its acceptance check; plan 10-02 Task 2's first real commit records the untracked reality in its message body; no no-op commit is created.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All code/tests | ✓ | go1.27.0 local (module declares 1.26.4; CI pins via `go-version-file: go.mod`) | Toolchain auto-download per go.mod |
| `make` | `make coverage-quick` gate (AGENTS.md) | ✓ | GNU Make 3.81 | — |
| `go-test-coverage` | Coverage gate | ✓ | installed at ~/go/bin | `make check-go-test-coverage` auto-installs |
| `golangci-lint` | Pre-commit / CI lint | ✓ | v2.12.2 | `make deps` |
| `pre-commit` | Commit hooks | ✓ | 4.6.0 | CI enforces the same checks |
| Docker | — (not required by this phase) | ✓ | — | — |
| 3-OS CI matrix (ubuntu/macos/windows) | Cross-platform guarantees (ROBT-04) | ✓ (GitHub Actions) | — | Local darwin + targeted skips for Windows-only cases |

**Missing dependencies with no fallback:** none — this phase is stdlib-only and all tooling is present.

## Security Domain

> Required — `workflow.security_enforcement: true` in .planning/config.json. ASVS level 1; block on HIGH.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no users/sessions) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (no authorization model; OS permissions surface as `ErrPermissionDenied`) |
| V5 Input Validation | yes | `filepath.Clean` on the folder argument (D-08); manifest names are compile-time constants (never user input) joined via `filepath.Join`; 1 MB cap on all file reads |
| V6 Cryptography | no | — (no crypto) |

### Known Threat Patterns for {stdlib filesystem probe}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via manifest name (`filepath.Join(folder, name)` with attacker-controlled `name`) | Tampering | Impossible by construction: `readManifest` call sites pass constants ("go.mod", "package.json", …); root-scoped (DETC-01) means no recursive walk; document with `// #nosec G304` rationale if gosec flags the join |
| Unbounded file read / memory exhaustion on pathological manifests | DoS | `io.LimitReader(1<<20+1)` cap (ROBT-02) — verified pattern |
| Panic escape from parsing untrusted content | DoS | Per-detector `defer recover()` at registry dispatch (Pattern 2) — never re-panics |
| Hidden/ignored content masquerading as project markers (e.g. `node_modules/package.json`) | Spoofing | Ignore-list gate before cascade (ROBT-01/D-12); exact-case names (D-11); root-scoped so subdir markers are never read |
| Symlink-following into unexpected locations | Tampering | Anti-feature (ROBT-05, audited Phase 14): no `EvalSymlinks`, no recursive discovery; the probe only stats/reads the one cleaned path |

## Project Constraints (from AGENTS.md)

Directives the planner must honor (verbatim intent from `./AGENTS.md`):

- **Milestone branches:** every milestone gets its own git branch for the PR flow; work on the milestone branch, then open a PR to `main`. Branch name: `gsd/v1.7-{slug}`.
- **Before every commit:** run `make coverage-quick` — enforces thresholds in `.testcoverage-quick.yml` (packages ≥80%, files ≥70%, total ≥75% [VERIFIED: .testcoverage-quick.yml]). Do not commit if it fails; fix uncovered code or add tests first.
- **Before pushing a tag:** regenerate and stage the quality report (`make quality-report`, `git add quality-report.md`).
- **Cross-platform testing (CI runs Linux/macOS/Windows):** tests modifying global state (the detector registry slice, here) must NOT use `t.Parallel()`; Windows permission semantics differ (skip-gate the chmod-based test); never rely on OS-specific file-mode behaviors in assertions.
- **Code style:** standard Go idioms; focused single-purpose functions; `testify` for assertions; unit + edge-case tests; Go doc comments for all exported symbols; Conventional Commits; minimal external dependencies — prefer stdlib.
- **Spike findings:** consult `Skill("spike-findings-go")` before committing (panic-recovery precedent sourced from it).

## Sources

### Primary (HIGH confidence)
- [VERIFIED: pkg.go.dev/os] + [VERIFIED: src/os/dir.go] — `ReadDir` "returning all its directory entries sorted by filename"; `slices.SortFunc` in source
- [VERIFIED: src/os/error.go] — portable error vars; "New code should use errors.Is"; `underlyingError` unwrap semantics
- [VERIFIED: src/os/error_windows_test.go] — ERROR_FILE_NOT_FOUND / ERROR_PATH_NOT_FOUND → not-exist; ERROR_ACCESS_DENIED → permission (Windows mapping for D-04)
- [VERIFIED: golang/go issue #75157] — `os.ReadDir` on a file returns `syscall.ENOTDIR` on Linux, macOS, Windows; Go 1.26 CL 699375 nil-slice consistency fix
- [VERIFIED: pkg.go.dev/io] + [VERIFIED: src/io/io.go] — `LimitReader` "stops with EOF after n bytes"; `ReadAll` growth behavior
- [CITED: go.dev/blog/go1.13-errors] — sentinel + `%w` wrapping doctrine; when to wrap vs not; `errors.Is` over `==`
- [VERIFIED (in-repo): project_detector/detector.go:11-60] — old `DetectorFunc` contract, ignoreDirs map, `path.Base`, error-as-control-flow (the anti-patterns); `git ls-files project_detector` → 0 tracked files; reproduced `go build ./...` failure (3 errors, BurntSushi + gs-dev)
- [VERIFIED (in-repo): cache/singleflight.go:77-88, cache/errors.go:38-44] — `callSetter` named-return + recover precedent; sentinel var-block style
- [VERIFIED (in-repo): .planning/codebase/CONVENTIONS.md, STRUCTURE.md, STACK.md, TESTING.md] — naming, error handling, doc.go, declaration order, package layout, test patterns
- [VERIFIED (in-repo): .testcoverage-quick.yml, Makefile:119-122, .github/workflows/go.yml, go.mod:3,13] — coverage gates, CI matrix, module facts
- [VERIFIED (in-repo): 10-CONTEXT.md D-01..D-12] — locked decisions quoted verbatim in User Constraints

### Secondary (MEDIUM confidence)
- [CITED: pkg.go.dev/errors] — `errors.Is` semantics, sentinel uniqueness (New returns distinct values)
- [CITED: gofaq.org io.LimitReader memory-exhaustion guide] — limit+1 truncation-probe pattern (corroborates the design)
- [CITED: web-developpeur.com sentinel/errors.Is decision table] — caller-distinguishes-causes → sentinel+`%w` (matches D-01/D-02)

### Tertiary (LOW confidence)
- [ASSUMED] Dolt benchmark (errors.Is performance) — noted, not used: the error path is cold in a probe API; performance is irrelevant here

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib-only mandate + testify verified in go.mod; no version risk
- Architecture: HIGH — every pattern anchored to verified stdlib behavior or in-repo precedent; the two open questions are small, explicit decisions
- Pitfalls: HIGH — five of six pitfalls verified against primary sources or reproduced this session; permission-test platform caveats are standard CI knowledge
- Assumptions: 7 logged items, all LOW-risk; none block planning (OQ-1 and OQ-2 were the only decisions the planner had to surface — both resolved and recorded in §Open Questions (RESOLVED))

**Research date:** 2026-09-28
**Valid until:** 2026-10-28 (stdlib semantics stable; Go 1.26/1.27 boundary noted in State of the Art)