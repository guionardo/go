# Phase 17: Delivery Hardening - Pattern Map

**Mapped:** 2026-10-26
**Files analyzed:** 5 (all modified-in-place; zero new Go source files)
**Analogs found:** 5 / 5 (4 exact in-file analogs, 1 role-match)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.github/workflows/go.yml` | config (CI workflow) | event-driven (push/PR → matrix job) | `.github/workflows/go.yml` `test` job (self) + `.github/workflows/release.yml` (action/env style) | exact (in-file) |
| `cache/bench_test.go` | test (benchmark harness) | batch / measured loop | `cache/bench_test.go:446-448` (mem registration) + `:358-441` (`runBatchBenchmarks`) + `:674-681` (provider loop) | exact (in-file) |
| `cache/sqlite/doc.go` | docs (package doc comment) | static (no runtime) | `cache/sqlite/doc.go:34-45` (paragraph to extend) + `cache/mem/doc.go` (prefix/voice) | exact (in-file) |
| `cache/sqlite/example_test.go` | test (runnable example) | file-I/O + CRUD | `cache/sqlite/example_test.go:10-34` (existing `Example`) + `cache/mem/example_test.go:20-27` (naming) | exact (in-file) |
| `README.md` | docs | static | `README.md:87-90` (sqlite paragraph) + `:22`, `:85`, `:159` (verify-only rows) | exact (in-file) |

**Verification-only (do NOT modify):** `go.mod` (pins), `.testcoverage.yml`, `.testcoverage-quick.yml` (no `cache/sqlite` override), `Makefile` (`benchmark*`, `coverage*`), `cache/sqlite/twoprocess_e2e_test.go`.

---

## Pattern Assignments

### `.github/workflows/go.yml` (config, event-driven)

**Analog:** this file's existing `test` job (lines 10-39); sibling style reference `.github/workflows/release.yml`.

**Job skeleton + permissions pattern** (`go.yml:10-24`) — preserve exactly; only `env:` and one step are added:
```yaml
  test:
    name: Run tests
    permissions:
      contents: read
      pull-requests: read
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
```

**Job-level env pattern (D-01)** — insert `env:` between `runs-on:` and `steps:`; quoting matches repo style (`release.yml:21` uses `go-version: "1.26"`):
```yaml
    runs-on: ${{ matrix.os }}
    env:
      CGO_ENABLED: "0"
    steps:
```

**Visible-proof step pattern (D-03)** — new step placed between `Set up Go` and `Build swapper binaries`. `shell: bash` is load-bearing (pwsh only honors the *last* exit code):
```yaml
      - name: Build (CGO-free)
        shell: bash
        run: |
          go env CGO_ENABLED
          go build ./...
          go vet ./...
          go list -m modernc.org/sqlite modernc.org/libc
```

**Step-level env override pattern (D-02)** — the existing race step keeps its `if:` guard untouched and gains one `env:` key:
```yaml
      # -race requires a C toolchain, which Linux and macOS runners provide
      # (they satisfy CONC-03). Enabling -race on Windows (mingw setup) is a
      # reviewer-flagged open question for a future phase.
      - name: SQLite race detector
        if: runner.os != 'Windows'
        env:
          CGO_ENABLED: "1"
        run: go test -race ./cache/sqlite/ -count=1 -timeout=180s
```

**Do NOT touch:** `coverage` job (`go.yml:41-69`) — separate job, no `env:`, `setup-go@v3`, `threshold-total: 75`. Also keep the existing E2E step (`:31-32`) and `Run tests` (`:28-29`) byte-identical; they inherit the job env.

---

### `cache/bench_test.go` (test harness, batch loops)

**Analog:** `cache/bench_test.go` — the mem registration and the shared helpers.

**Import block convention** (`:3-24`) — stdlib group, then testcontainers group, then first-party group. Add `"path/filepath"` to the stdlib group and `"github.com/guionardo/go/cache/sqlite"` to the first-party group:
```go
import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	...
	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
	...
	"github.com/guionardo/go/cache/sqlite"
)
```

**Provider-registration pattern** (`:446-448`, `:674-681`) — the shape to copy: construct the provider in the parent `b.Run`, register cleanup, delegate to the shared helper:
```go
func BenchmarkBatch(b *testing.B) {
	c := mem.New[string, string](b.Context())
	defer c.Close()
	...
	for _, p := range providers {
		p := p // capture range variable
		b.Run(p.name, func(b *testing.B) {
			c := p.setup(b)
			defer c.Close()
			runBatchBenchmarks(b, c, p.name)
		})
	}
}
```

**New sqlite family (D-05/D-06/D-07)** — insert immediately before `// ---- Docker-backed provider batch subtests (gated by skipIfNoDocker) ----` (`:532`). Distinct `name` per family is mandatory (every key embeds it, `:364`, `:381`, `:411`); `b.Cleanup` registered after construction with a preceding blank line (`wsl_v5`, Pitfall 5):
```go
	// ---- SQLite provider batch subtests (always run, no Docker) ----

	b.Run("sqlite", func(b *testing.B) {
		b.Run("Memory", func(b *testing.B) {
			c := sqlite.New[string, string](sqlite.WithMemory())

			b.Cleanup(func() { _ = c.Close() })
			runBatchBenchmarks(b, c, "sqlite_memory")
		})

		b.Run("File", func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "batch.db")
			c := sqlite.New[string, string](sqlite.WithPath(path))

			b.Cleanup(func() { _ = c.Close() })
			runBatchBenchmarks(b, c, "sqlite_file")
		})
	})
```

**Helper signature — do NOT change** (`:320`, `:358`):
```go
func prePopulateKeys(b *testing.B, c cache.BatchCache[string, string], n int, prefix string) []string
func runBatchBenchmarks(b *testing.B, c cache.BatchCache[string, string], name string)
```

**Primitive-ops benchmark (new, no existing analog)** — new `const` before funcs (`decorder`), new `runPrimitiveBenchmarks`, new `BenchmarkPrimitives`. Bound growth with a key ring (`i%primitiveKeyRing`), never a unique key per iteration:
```go
// primitiveKeyRing bounds key growth during timed loops so file-mode Set
// measures overwrite cost, not b-tree growth.
const primitiveKeyRing = 1024

func runPrimitiveBenchmarks(b *testing.B, c cache.BatchCache[string, string]) {
	b.Run("Get", func(b *testing.B) {
		keys := prePopulateKeys(b, c, 1, "prim_get")

		for i := 0; i < b.N; i++ {
			_, _ = c.Get(b.Context(), keys[0])
		}
	})

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = c.Set(b.Context(), fmt.Sprintf("prim_set_%d", i%primitiveKeyRing), "v")
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("prim_del_%d", i%primitiveKeyRing)
			_ = c.Set(b.Context(), key, "v")
			_ = c.Delete(b.Context(), key)
		}
	})

	b.Run("GetOrSet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("prim_gos_%d", i%primitiveKeyRing)
			_, _ = c.GetOrSet(b.Context(), key, func(context.Context) (string, error) { return "v", nil })
		}
	})
}

func BenchmarkPrimitives(b *testing.B) {
	b.Run("mem", func(b *testing.B) {
		c := mem.New[string, string](b.Context())

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})

	b.Run("sqlite/Memory", func(b *testing.B) {
		c := sqlite.New[string, string](sqlite.WithMemory())

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})

	b.Run("sqlite/File", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), "primitives.db")
		c := sqlite.New[string, string](sqlite.WithPath(path))

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})
}
```

**Lint constraints proven by probe** (`.golangci.yml:31` contextcheck, `:54` wsl_v5, `:24` decorder):
- `b.Context()` inline at every call site; never hoist `ctx := b.Context()` into a helper that also calls `prePopulateKeys` → `contextcheck` fires.
- Blank line between a `:=` construct and `b.Cleanup(...)`.
- `const` declared before `func` in the file.

**Discovery (no config change needed):** `make benchmark` / `benchmark-quick` already glob `./cache/...` (`Makefile:98`, `:103`), so the new families are picked up automatically.

---

### `cache/sqlite/doc.go` (package docs, static)

**Analog:** same file, `Data and state` paragraph (`:34-45`) and `cache/mem/doc.go:1-13` for the `// Package <name> provides` voice.

**Existing paragraph to extend** (`:34-45`) — keep the canceled-MGet sentence where it is; insert the new paragraph after it, still inside the single package comment block (godoclint requires the package prefix at `:1`):
```go
// Data and state: file-backed entries persist across process runs. Close
// closes the database, which checkpoints the WAL and removes the -wal/-shm
// sidecars on the last connection. ...
```

**New paragraph (D-09)** — 120-char `lll` limit, no detached comment, no chunk-size number:
```go
// Batch operations follow the same durability model. MGet issues chunked IN
// queries and silently skips missing, expired, and undecodable entries; a
// failed chunk is logged and skipped, and a canceled context yields the
// partial results gathered so far (the frozen signature has no error channel).
// MSet resolves one TTL for the whole batch and writes every pair in a single
// BEGIN IMMEDIATE transaction: any error — marshal, begin, exec, or commit —
// rolls the entire batch back. MDel deletes in chunks and is idempotent for
// duplicate and missing keys.
```

**Facts the paragraph must match (verified in code):** `cache/sqlite/batch.go:16-19` (`chunkSize = 100`, fixed internal constant), `:68-73` (`MGetFunc` doc), `:143-148` + `:183-209` (`MSet` marshal-then-transaction), `:211-233` (`MDel` chunked, idempotent).

**Do not** imply encryption: `doc.go:54-55` states contents are unencrypted at rest.

---

### `cache/sqlite/example_test.go` (runnable example, file-I/O + CRUD)

**Analog:** the file's existing `Example` (`:10-34`) — house style: `package sqlite_test`, early-return on error with a `"… failed:"` message, `defer func() { _ = c.Close() }()`, `// Output:` assertion.

**Naming convention analog** — `cache/mem/example_test.go:11` / `:20`:
```go
func ExampleNew() { ... }
func ExampleNew_withTTL() { ... }
```
⇒ use `ExampleNew_withPath` (mirrors the lowercase-suffix convention).

**Structure to copy** (`cache/sqlite/example_test.go:13-33`):
```go
func Example() {
	c := sqlite.New[string, string](sqlite.WithMemory())
	defer func() { _ = c.Close() }()

	ctx := context.Background()

	if err := c.Set(ctx, "greeting", "hello"); err != nil {
		fmt.Println("set failed:", err)

		return
	}

	value, err := c.Get(ctx, "greeting")
	if err != nil {
		fmt.Println("get failed:", err)

		return
	}

	fmt.Println("value:", value)
	// Output: value: hello
}
```

**New durability example (D-10)** — no `*testing.T` exists in an example, so `t.TempDir()` is illegal; use `os.MkdirTemp` + `defer os.RemoveAll` (`usetesting` does not flag it here). Never use `WithName` (writes to the user cache dir):
```go
// ExampleNew_withPath demonstrates durability across a close/reopen cycle.
func ExampleNew_withPath() {
	dir, err := os.MkdirTemp("", "sqlite-cache-example")
	if err != nil {
		fmt.Println("temp dir failed:", err)

		return
	}

	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "cache.db")
	ctx := context.Background()

	first := sqlite.New[string, string](sqlite.WithPath(path))
	if err := first.Set(ctx, "greeting", "hello"); err != nil {
		fmt.Println("set failed:", err)

		return
	}

	if err := first.Close(); err != nil {
		fmt.Println("close failed:", err)

		return
	}

	second := sqlite.New[string, string](sqlite.WithPath(path))
	defer func() { _ = second.Close() }()

	value, err := second.Get(ctx, "greeting")
	if err != nil {
		fmt.Println("get failed:", err)

		return
	}

	fmt.Println("value:", value)
	// Output: value: hello
}
```
Add `"os"` and `"path/filepath"` to the import block (`:3-8`). Skip `errors.Is(err, cache.ErrMiss)` — it would import `cache` for one assertion.

---

### `README.md` (docs, static)

**Analog:** the file's own sqlite surfaces. Three are verify-only; one gets a clause.

**Verify-only (accurate as shipped):**
```markdown
| [sqlite](#package-cache) | `cache/sqlite` | SQLite cache backend (embedded, survives restarts) |     # :22
| `cache/sqlite` | SQLite (embedded) | modernc.org/sqlite | Eager — local file, lazy errors |  # :85
```
Also verify `README.md:72` and `:159` already describe sqlite batching ("chunked IN queries + single-transaction upsert") — do not duplicate.

**Clause target (D-11)** — append one sentence to the paragraph at `:87-90`, which currently reads:
```markdown
`cache/sqlite` is pure-Go embedded SQLite with no server to run: the zero-value
configuration opens an in-memory cache, `WithPath`/`WithName` open a durable
WAL-mode file that survives restarts, and the same file can be shared by
multiple processes on one host (a 5 s busy timeout bounds write contention).
```
Append a sentence that mirrors `doc.go` (no numbers, no new section, no example):
```markdown
`MGet` silently skips missing, expired, and undecodable keys, `MSet` writes the
whole batch in one transaction under one TTL (all-or-nothing), and `MDel`
deletes in idempotent chunks.
```

**Do NOT touch:** Benchmark Results table (`README.md:161-172`) — D-08 forbids sqlite numbers.

---

## Shared Patterns

### Git-tracked analog gate
All analog paths above verified with `git ls-files` (non-empty): `.github/workflows/go.yml`, `.github/workflows/release.yml`, `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/batch.go`, `cache/sqlite/example_test.go`, `cache/mem/doc.go`, `cache/mem/example_test.go`, `README.md`. No `.gsd/capabilities/**` mirror or gitignored runtime path is referenced anywhere in this map.

### Lint gate (applies to every edited Go file)
**Source:** `.golangci.yml:6-8` (`new: true`, `new-from-rev: HEAD`) + enabled set including `godoclint`, `lll`, `decorder`, `contextcheck`, `wsl_v5`, `usetesting`.
**Apply to:** `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/example_test.go`.
```bash
golangci-lint run ./cache/... ./cache/sqlite/...   # must be 0 issues
```

### Doc-comment convention
**Source:** `cache/sqlite/doc.go:1-3`, `cache/mem/doc.go:1`.
**Apply to:** `cache/sqlite/doc.go`, all new exported/unexported benchmark helpers (each has a leading `// Name ...` sentence, as in `bench_test.go:318-320`).
```go
// Package sqlite provides an embedded, durable cache backend for the cache
// package — import path github.com/guionardo/go/cache/sqlite — backed by
// pure-Go SQLite (modernc.org/sqlite) with no CGO.
```

### Coverage gate (applies to the whole phase)
**Source:** `Makefile:119-121` (`coverage-quick`) + `.testcoverage-quick.yml` (file 70 / package 80 / total 75, no `cache/sqlite` override).
**Apply to:** every commit in this phase.
```bash
make coverage-quick   # required before each commit by AGENTS.md
```

### Benchmark cleanup ordering (Windows correctness)
**Source:** `cache/sqlite/sqlite_test.go` house pattern (`t.Cleanup(func() { _ = c.Close() })` after `t.TempDir()`) and `.planning/research/PITFALLS.md` Pitfall 6.
**Apply to:** every sqlite cache constructed in `cache/bench_test.go`.
```go
path := filepath.Join(b.TempDir(), "batch.db")   // TempDir first
c := sqlite.New[string, string](sqlite.WithPath(path))

b.Cleanup(func() { _ = c.Close() })              // registered last → runs first (LIFO)
```

## No Analog Found

| File / Symbol | Role | Data Flow | Reason |
|---------------|------|-----------|--------|
| `BenchmarkPrimitives` / `runPrimitiveBenchmarks` (new, inside `cache/bench_test.go`) | test (benchmark) | measured loop | No primitive-ops (Get/Set/Delete/GetOrSet) benchmark exists anywhere in the repo; `BenchmarkSingleflightGetOrSet` (`:156-314`) is a herd benchmark, not a per-op one. Use the RESEARCH.md validated block verbatim. |
| `Build (CGO-free)` CI step (`shell: bash` + `go env CGO_ENABLED`) | config (CI) | event-driven | No existing step proves an environment property; nearest shape is `release.yml:27-31` (`Build` / `Test` steps) for naming, but the `shell: bash` + `go env` proof has no precedent. |

## Metadata

**Analog search scope:** `.github/workflows/`, `cache/` (root + `sqlite/`, `mem/`), `README.md`, `Makefile`, `.golangci.yml`, `.testcoverage*.yml`, `go.mod`.
**Files scanned:** 12 read in full or in cited ranges; 46 candidate `*_test.go` / 25 `doc.go` / 3 workflow files enumerated.
**Key conventions confirmed:** `package cache_test` benchmarks with shared helpers; always-run vs `skipIfNoDocker` split; `// Package <name> provides` doc prefix; `ExampleNew_variant` naming; job-level `env` + step-level exception; additive `b.Run` family registration.
**Pattern extraction date:** 2026-10-26
