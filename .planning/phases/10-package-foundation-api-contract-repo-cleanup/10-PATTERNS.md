# Phase 10: Package Foundation — API Contract + Repo Cleanup - Pattern Map

**Mapped:** 2026-09-28
**Files analyzed:** 12 new files + 1 deletion action
**Analogs found:** 10 / 12 (2 design-provided, no in-repo analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `project_probe/doc.go` | docs | n/a (package contract doc) | `cache/doc.go` | exact |
| `project_probe/errors.go` | utility | transform (OS errno → sentinel) | `cache/errors.go` | role-match |
| `project_probe/project_data.go` | model | pure data (struct + type + consts) | `httptest_mock/mock.go` | role-match |
| `project_probe/probe.go` | service (entry point) | request-response + file-I/O | `path_tools/root_folder.go` | role-match |
| `project_probe/registry.go` | service | first-match-wins dispatch, panic-recover | `cache/singleflight.go` | exact (panic part) |
| `project_probe/ignore_list.go` | utility | transform (entries → bool gate) | `set/set.go` | role-match |
| `project_probe/read_manifest.go` | utility | file-I/O (capped read) | `httptest_mock/request.go` | partial |
| `project_probe/probe_test.go` | test | table-driven + sentinels | `path_tools/root_folder_test.go` | exact |
| `project_probe/errors_test.go` | test | errno mapping table | `fraction/fraction_test.go` | role-match |
| `project_probe/registry_test.go` | test | panic-recovery, global-state (NO parallel) | `cache/singleflight_test.go` | role-match |
| `project_probe/manifest_test.go` | test | boundary cases (cap/BOM) | — (research Pattern 4) | none |
| `project_probe/example_test.go` | test (example) | deterministic output | `fraction/example_test.go` | exact |
| `rm -rf project_detector/` (FND-01) | deletion action | n/a | — (untracked dir, see note) | none |

**Naming note (agent discretion):** CONTEXT.md leaves file layout inside `project_probe/` open; RESEARCH.md suggests `errors.go`, `project.go`, `ignore.go`, `manifest.go`. Roles/data flows below are name-independent. All analog paths below are **git-tracked** (verified via `git ls-files`). `project_detector/` is **untracked** (0 files in index, `??` in status — verified) and is deleted by FND-01; it is referenced only as an anti-pattern, never as a copy source.

## Pattern Assignments

### `project_probe/doc.go` (docs)

**Analog:** `cache/doc.go` (exact — the only package doc that documents sentinel errors + Usage block, which is what FND-02 needs for the never-fail contract)

**Package doc pattern** (cache/doc.go:1-17, 50-62):
```go
// Package cache provides a generic key-value cache abstraction
// with pluggable backend providers.
//
// The Cache[K, V] interface exposes Get, Set, Delete, GetOrSet, and Close —
// all accepting context.Context for cancellation and timeout propagation.
// ...
// Usage:
//
//	import "github.com/guionardo/go/cache"
//
//	var c cache.Cache[string, string]
//	c = mem.New[string, string]()
//	c.Set(ctx, "key", "value")
//	v, err := c.Get(ctx, "key")
//
// ...
// Sentinel errors (wrapped with provider prefix):
//
//	var ErrMiss    = errors.New("cache: key not found")
//	var ErrClosed  = errors.New("cache: cache is closed")
//	var ErrCanceled = errors.New("cache: canceled") // wraps the waiter's ctx.Err()
package cache
```

**What to copy:** the `// Package <name> provides <summary>.` first line (godoclint-enforced, CONVENTIONS.md §Comments), a `Usage:` indented code block, and a `Sentinel errors` section mirroring D-01's three sentinels. The never-fail contract sentence (FND-02) goes in the prose: errors reserved for hard folder I/O; `LanguageUnknown` + nil error for content outcomes.

### `project_probe/errors.go` (utility, transform)

**Analog:** `cache/errors.go` (role-match — sentinel var block with doc comments); `release/self_update.go` (sentinel + `%w` mixing)

**Sentinel var block pattern** (cache/errors.go:38-44):
```go
var (
	// ErrMiss is returned by Get when the key is not in the cache.
	ErrMiss = errors.New("cache: key not found")

	// ErrClosed is returned when operations are attempted on a closed cache.
	ErrClosed = errors.New("cache: cache is closed")
)
```

**Sentinel + contextual wrap mixing** (release/self_update.go:43, 87-97):
```go
var (
	ErrUpdateInProgress = errors.New("update already in progress")
)
// ...
	if err != nil {
		return nil, fmt.Errorf("update lock path: %w", err)
	}
	...
		return nil, ErrUpdateInProgress
```

**%w wrap-with-context style** (config/environment/environment.go:160-165):
```go
return fmt.Errorf("invalid struct value for field %s: %w", field.Name, err)
```

**What to copy:** the grouped `var (...)` block with one doc comment per sentinel (D-01: `ErrFolderNotFound`, `ErrNotDirectory`, `ErrPermissionDenied`); the `fmt.Errorf("probe %s: %w", folder, err)` wrap shape (D-02 — same shape as `"update lock path: %w"`). `mapFolderError` (D-04) has **no in-repo analog** (no package maps syscall errnos) — use RESEARCH.md Pattern 3 verbatim; import `syscall` for `ENOENT`/`ENOTDIR`/`EACCES` and match with `errors.Is`, checking ENOENT → ENOTDIR → EACCES(+EPERM) → raw fallback.

### `project_probe/project_data.go` (model, pure data)

**Analog:** `httptest_mock/mock.go` (role-match — exported type + exported const block); `fraction/fraction.go` (grouped type declarations with doc comments)

**Typed type + const block pattern** (httptest_mock/mock.go:40-49):
```go
	RequestMatchLevel uint8
)

const (
	// MatchLevelNone indicates no match.
	MatchLevelNone RequestMatchLevel = iota
	// MatchLevelPartial indicates a partial match.
	MatchLevelPartial
	// MatchLevelFull indicates a full match.
	MatchLevelFull
)
```

**Grouped type declarations** (fraction/fraction.go:8-21):
```go
type (
	// Fraction represents a fraction. It is an immutable type.
	Fraction struct {
		numerator   int64
		denominator int64
	}
)
```

**What to copy:** `type Language string` + the 8 exported PascalCase constants (D-05/D-06/D-07) following the `RequestMatchLevel` shape — but note the divergence: `Language` constants use **explicit string values** (`LanguageGo Language = "Go"` … `LanguageUnknown Language = "unknown"`), NOT `iota`, because D-06/D-07 lock display values. `ProjectData{Folder, Language, Name, Version, Description}` struct in a grouped `type (...)` block. Declaration order `type` → `const` → `var` → `func` is enforced by `decorder` (CONVENTIONS.md:112).

### `project_probe/probe.go` (service entry point, request-response + file-I/O)

**Analog:** `path_tools/root_folder.go` (role-match — the repo's only folder-I/O entry point with sentinel errors, stat + filepath walking, doc comment)

**Entry-point + stat + sentinel pattern** (path_tools/root_folder.go:11-38):
```go
// GetRootFolder returns the base folder of a golang project (finding the go.mod file)
func GetRootFolder(base string) (rootFolder string, err error) {
	if len(base) == 0 {
		base, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	if !DirExists(base) {
		return "", os.ErrNotExist
	}
	// ... filepath.Join(base, "go.mod") existence loop ...
	return "", ErrNotAGoProject
}
```

**What to copy:** the doc-comment-first exported function shape, `os.Stat` + early `return ..., err` style, sentinel return at the end, `filepath` (never `path` — ROBT-04). `Probe`'s body itself is **design-provided** — use RESEARCH.md Code Examples "Probe entry point" verbatim (clean-once D-09, `folder == ""` guard per OQ-1, `os.Stat` → `!info.IsDir()` → `os.ReadDir` → `hasContent` gate → `runDetectors`, merge rule A7). `DirExists`/`FileExists` from `path_tools/path_tool.go:7-26` are available but NOT needed here (readManifest does its own `os.Open`).

### `project_probe/registry.go` (service, dispatch + panic-recover)

**Analog:** `cache/singleflight.go` (exact — research-cited precedent for the panic-recover dispatch)

**Named-return + defer-recover pattern** (cache/singleflight.go:73-88):
```go
// callSetter runs the setter with panic recovery. A recovered panic becomes a
// *cache.PanicError wrapping the value and stack (mirrors x/sync's panicError), so
// a panicking setter yields the typed error to every waiter and no panic ever
// escapes to a caller goroutine (SF-05 / D-16 / D-17). It never re-panics.
func (s *SingleflightGetOrSet[K, V]) callSetter(
	setter func(context.Context) (V, error),
	ctx context.Context,
) (v V, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = newPanic(r)
		}
	}()

	return setter(ctx)
}
```

**What to copy:** the named-return `(v V, err error)` + `defer func() { if r := recover(); r != nil { ... } }()` skeleton — but adapted per D-03/Pattern 2: a recovered panic is a **non-match** (`pd, ok = ProjectData{}, false`), never an error, never a re-panic. RESEARCH.md Pattern 2 gives the full `detectorFunc` / `detectors = []detectorFunc{}` / `runDetectors` / `callDetector` skeleton — copy verbatim; the empty slice literal is the Phase-10 state, extended in phases 11–13 in the same file (init-order across files is lexical — never rely on it).

### `project_probe/ignore_list.go` (utility, transform)

**Analog:** `set/set.go` (role-match — the repo's `map[T]struct{}` set shape)

**Set-as-map pattern** (set/set.go:7-10):
```go
// Set values methods
type Set[T comparable] map[T]struct{}

var emptyStruct = struct{}{}
```

**What to copy:** the `map[string]struct{}` shape for the 13-entry ignore list (D-10) + membership test `_, ok := m[name]; ok`; `var emptyStruct = struct{}{}` style (or inline `struct{}{}` per agent discretion). `hasContent(entries)` (D-12) is new code — iterate `os.ReadDir` entries, skip `entry.IsDir() && ignored(name)`, return true on first file or non-ignored dir. Exact-case matching (D-11), no normalization. Note: the old sample's `ignoreDirs` map (project_detector/detector.go:15-19) is the documented shape precedent but is **untracked** — do not copy from it; the 13-entry D-10 list is in CONTEXT.md.

### `project_probe/read_manifest.go` (utility, file-I/O)

**Analog:** `httptest_mock/request.go` (partial — the repo's only `io.LimitReader` usage, same capped-read data flow)

**Capped-read precedent** (httptest_mock/request.go:213):
```go
body, err := io.ReadAll(io.LimitReader(req.Body, 10<<20)) // nolint: mnd
```

**What to copy:** `io.ReadAll(io.LimitReader(...))` composition. The full `readManifest` is **design-provided** — copy RESEARCH.md Pattern 4 verbatim: `maxManifestSize = 1 << 20` const, `utf8BOM = []byte{0xEF, 0xBB, 0xBF}` var, `LimitReader(f, maxManifestSize+1)` truncation probe, `bytes.TrimPrefix` strip, `([]byte, bool)` signature (never `error` — D-03 consistency). If gosec flags `filepath.Join`, annotate `// #nosec G304` with the compile-time-constant rationale (CONVENTIONS.md precedent).

### `project_probe/probe_test.go` (test, table-driven)

**Analog:** `path_tools/root_folder_test.go` (exact — table of subtests with `require.ErrorIs` sentinel assertions + `t.TempDir()`)

**Table-driven sentinel-test pattern** (path_tools/root_folder_test.go:12-32):
```go
func TestGetRootFolder(t *testing.T) {
	t.Parallel()
	t.Run("current_test_folder_should_return_valid_folder", func(t *testing.T) {
		t.Parallel()
		got, gotErr := pathtools.GetRootFolder("")
		require.NoError(t, gotErr)
		require.True(t, pathtools.DirExists(got))
	})
	t.Run("not_a_go_project", func(t *testing.T) {
		t.Parallel()
		got, gotErr := pathtools.GetRootFolder(t.TempDir())
		require.ErrorIs(t, gotErr, pathtools.ErrNotAGoProject)
		require.Empty(t, got)
	})
}
```

**What to copy:** the subtest table structure, `t.Parallel()` per subtest (safe — no global state), `require.ErrorIs` for sentinels (asserts wrapping, not identity), `t.TempDir()` for fixtures. Use RESEARCH.md "Table-driven Probe tests" skeleton (missing_folder / path_is_a_file / empty_folder / ignored_only / unknown_content rows). Assert `data.Folder == filepath.Clean(folder)` (D-08/D-09) and `assert.Equal`, never entries nil-ness (anti-pattern: assert with `require.Empty`/`assert.Len`).

### `project_probe/errors_test.go` (test, errno mapping)

**Analog:** `fraction/fraction_test.go` (role-match — `require.ErrorIs` sentinel assertions on direct calls); RESEARCH.md Pitfall 3 for the OS-gated permission test

**Sentinel assertion pattern** (fraction/fraction_test.go:60-62):
```go
	require.ErrorIs(t, err, fraction.ErrZeroDenominator)
```

**What to copy:** table-driven `mapFolderError` tests with **synthetic** `&os.PathError{Err: syscall.EACCES}` inputs (per RESEARCH Pitfall 3 — never rely on `chmod 000` for the EACCES row; it fails on Windows and under root). The filesystem permission test must be OS-gated: `if runtime.GOOS == "windows" || os.Geteuid() == 0 { t.Skip(...) }`.

### `project_probe/registry_test.go` (test, global-state mutation — NO t.Parallel)

**Analog:** `cache/singleflight_test.go` (role-match — the panic-recovery test shape)

**Panic-recovery test pattern** (cache/singleflight_test.go:94-131, 135-151):
```go
// TestSingleflightGetOrSet_PanicRecovery verifies that a panicking setter
// yields a *cache.PanicError to every waiter and no panic escapes the Do call
func TestSingleflightGetOrSet_PanicRecovery(t *testing.T) {
	t.Parallel()
	t.Run("string_panic_value", func(t *testing.T) {
		t.Parallel()
		...
		setter := func(context.Context) (string, error) { //nolint:unparam
			panic("boom")
		}
		...
		require.ErrorAs(t, err, &p, ...)
	})
}
```

**What to copy:** the panic-injection test shape (`panic("boom")` in the injected func, assert the dispatch swallowed it and returned non-match/Unknown). **Critical divergence:** registry tests mutate the package-level `detectors` slice, so the whole `registry_test.go` runs **without `t.Parallel()`** — AGENTS.md explicitly forbids parallel tests on global state (mid/`collectFuncs` precedent). `runDetectors`/`callDetector` are unexported — tests use internal `package projectprobe` (config precedent, TESTING.md). Cover: empty registry (Phase 10 default), injected order, first-match-wins, panic→non-match, and restore the slice after each test (save/restore the original `detectors`).

### `project_probe/manifest_test.go` (test, boundary cases)

**Analog:** none in-repo — use RESEARCH.md Pattern 4 + Pitfall 5 guidance

**Design-provided:** cap boundary (exactly 1 MB ok, 1 MB + 1 byte not-ok), BOM strip, missing file → `(nil, false)`, non-regular file. Direct unexported calls (internal `package projectprobe`). Assert with `assert.Equal`/`require.False`, never nil-ness of slices. No `t.Parallel()` restrictions apply (readManifest is stateless) — parallel is fine here.

### `project_probe/example_test.go` (test, example)

**Analog:** `fraction/example_test.go` (exact — external test package + deterministic `// Output`); `cache/example_test.go` (in-test stub for determinism)

**External-package example pattern** (fraction/example_test.go:1-15):
```go
package fraction_test

import (
	"fmt"

	"github.com/guionardo/go/fraction"
)

func ExampleNew() {
	f, _ := fraction.New[int, int](1, 2)
	fmt.Printf("%d/%d", f.Numerator(), f.Denominator())

	// Output:
	// 1/2
}
```

**What to copy:** external test package (`package projectprobe_test`), import with alias `projectprobe "github.com/guionardo/go/project_probe"` (underscore dir convention, CONVENTIONS.md §Path Aliases — `httptestmock` precedent), `// Output:` comment. **Determinism constraint:** `ExampleProbe` must print only deterministic fields — `data.Language` and `err == nil` — never the temp path (RESEARCH anti-pattern list). Use `t.TempDir()`-style fixtures **inside** the example via a real temp dir but print only Language/error.

## Shared Patterns

### Sentinel error var blocks
**Source:** `cache/errors.go:38-44`, `fraction/fraction.go:34-50`, `release/self_update.go:43`
**Apply to:** `errors.go`, `doc.go`
```go
var (
	// ErrX is returned when ...
	ErrX = errors.New("probe: ...")
)
```
One doc comment per sentinel; exported only (D-01 — published contract); unexport aggressively otherwise (CONVENTIONS.md §Module Design).

### %w wrapping with context (D-02)
**Source:** `release/self_update.go:87-97`, `config/environment/environment.go:160-165`, `config/profile/profile.go:49`
**Apply to:** `probe.go`, `errors.go` (mapFolderError)
```go
return fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
```
Callers get path context AND `errors.Is` traversal. Mapped sentinels discard the raw OS error by design (go1.13 doctrine, RESEARCH Pattern 3).

### Panic-recovery named-return + defer
**Source:** `cache/singleflight.go:77-88`
**Apply to:** `registry.go` (callDetector) — adapted: panic → non-match, never error
```go
func callDetector(fn detectorFunc, folder string) (pd ProjectData, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			pd, ok = ProjectData{}, false
		}
	}()
	return fn(folder)
}
```

### No t.Parallel on global-state tests
**Source:** AGENTS.md (mid/`collectFuncs` precedent), RESEARCH Pitfall 4
**Apply to:** `registry_test.go` only — all other test files may use `t.Parallel()`.

### Example-test determinism
**Source:** `fraction/example_test.go`, `cache/example_test.go`
**Apply to:** `example_test.go` — print only `Language`/`err == nil`, never absolute paths.

### Coverage gate
**Source:** AGENTS.md + `.testcoverage-quick.yml` (pkg ≥80%, file ≥70%, total ≥75%)
**Apply to:** ALL files — `make coverage-quick` before every commit; no override exists for `project_probe` (Pitfall 5). Internal test package `projectprobe` needed to hit dispatch/errno/readManifest branches.

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `project_probe/read_manifest.go` | utility | file-I/O (capped read) | No in-repo capped-file-read + BOM helper; RESEARCH Pattern 4 is verified end-to-end (`io.LimitReader` + `bytes.TrimPrefix`) — partial analog only `httptest_mock/request.go:213` (LimitReader usage) |
| `project_probe/errors.go` (mapFolderError fn) | utility | transform (errno → sentinel) | No package maps syscall errnos; RESEARCH Pattern 3 verified cross-platform (Windows ERROR_* → same errnos); sentinel var-block style has analogs |
| `project_probe/registry.go` (ordered cascade) | service | first-match-wins | The old cascade (`project_detector/detector.go`) is **untracked** (tracked-source gate) and is the anti-pattern (error-as-control-flow); RESEARCH Pattern 2 skeleton is authoritative |
| `rm -rf project_detector/` (FND-01) | deletion | n/a | Directory is untracked (0 files in index) — `git rm` fails with "pathspec did not match"; deletion is `rm -rf` + `go build ./...` verification as task #1 (RESEARCH Pitfall 1) |

## Metadata

**Analog search scope:** `cache/`, `fraction/`, `config/`, `path_tools/`, `set/`, `httptest_mock/`, `release/`, `mid/` (per AGENTS.md), `.planning/codebase/` (CONVENTIONS/STRUCTURE/TESTING), `.opencode/skills/spike-findings-go/`
**Files scanned:** ~20 tracked files across 7 packages
**Pattern extraction date:** 2026-09-28
**Tracked-source gate:** every analog path above verified via `git ls-files` (non-empty). `project_detector/` explicitly excluded (untracked, deleted this phase).