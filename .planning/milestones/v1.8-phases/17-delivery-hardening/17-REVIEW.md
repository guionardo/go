---
phase: 17-delivery-hardening
reviewed: 2026-10-26T22:58:42Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - .github/workflows/go.yml
  - README.md
  - cache/bench_test.go
  - cache/sqlite/doc.go
  - cache/sqlite/example_test.go
findings:
  critical: 1
  warning: 4
  info: 6
  total: 11
status: issues_found
---

# Phase 17: Code Review Report

**Reviewed:** 2026-10-26T22:58:42Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Delivery-only phase: CI CGO-free proof, sqlite benchmark registration, and batch-surface
documentation. The scope is small, so every changed line was read and cross-checked against the
shipped implementation in `cache/sqlite/{batch,dsn,schema,sqlite,options}.go` and against `cmd/go`'s
own `-race` rules.

What holds up (verified, no finding):

- **CGO-free claim is real for build/test.** `CGO_ENABLED=0 go build ./...`, `go vet ./...` and
  `CGO_ENABLED=0 go test -race ./cache/sqlite/` were re-run locally and pass; `modernc.org/sqlite`
  is pure Go, so the job-level env genuinely exercises the claim and now also de-CGOs the three
  cross-compiled swapper builds.
- **Env precedence is correct.** Step-level `env` overrides job-level `env` in GitHub Actions, so the
  race step does get `CGO_ENABLED=1` inside a `CGO_ENABLED=0` job.
- **The race-step comment is accurate.** `cmd/go/internal/work/init.go` (`go1.27`) rejects
  `-race` when cgo is off for every GOOS except darwin — the comment's "every non-darwin OS" matches
  the toolchain check, not a guess.
- **`shell: bash` is genuinely load-bearing.** The pwsh wrapper only appends `exit $LASTEXITCODE`
  after the whole script, so a failing `go build` followed by a passing `go vet` would report green.
- **Benchmark cleanup ordering is right.** `b.Cleanup(Close)` is registered after `b.TempDir()`, so
  LIFO closes the sqlite handle before temp removal (Windows-safe). `b.Context()` is called inline at
  every use site, so no child benchmark ever runs against a canceled parent context. Sub-benchmark key
  prefixes (`sqlite_memory` / `sqlite_file` / `prim_*`) do not collide, and both families were actually
  executed here (`-bench 'BenchmarkBatch/sqlite|BenchmarkPrimitives'`) with no duplicate names or errors.
- **Doc claims match `batch.go`.** Chunked IN-list MGet with silent skips and partial-on-cancel, one
  TTL resolved once per MSet under one `BEGIN IMMEDIATE` (`_txlock=immediate` is in both DSNs, memory
  included), rollback-on-error, chunked idempotent MDel — all accurate. `ExampleNew_withPath` passes
  and its `// Output` really is produced only by a successful reopen.

One blocker remains, and it is the one thing that can silently fabricate the numbers this phase
exists to produce: the new primitive benchmark arms discard every error, and sqlite's constructor
defers its open errors by design.

No security issues were found in the changed lines (the `test:test` container credentials are
pre-existing, ephemeral, and test-only).

## Critical Issues

### CR-01: Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers

**File:** `cache/bench_test.go:712-741` (arms at `:717`, `:723`, `:730-731`, `:738`)

**Issue:** `runPrimitiveBenchmarks` discards the result of every operation:

```go
_, _ = c.Get(b.Context(), keys[0])
_ = c.Set(b.Context(), fmt.Sprintf("prim_set_%d", i%primitiveKeyRing), "v")
_ = c.Set(b.Context(), key, "v")
_ = c.Delete(b.Context(), key)
_, _ = c.GetOrSet(b.Context(), key, func(context.Context) (string, error) { return "v", nil })
```

This is not a theoretical error path. `sqlite.New` **never fails** — `sqlite.go:72` stores the open
result in `c.initErr`, and `check()` (`sqlite.go:177-187`) returns it from every operation. So any
open/validation failure (unwritable temp location, a path the driver rejects, a filesystem where the
WAL conversion fails, Windows path issues, an exhausted busy timeout during bootstrap) produces a
cache whose every `Get`/`Set`/`Delete` returns an error immediately, and the benchmark happily reports
~tens of nanoseconds per op with no indication anything failed. A maintainer reading
`BenchmarkPrimitives/sqlite/File` would conclude the durable path is faster than `mem`.

`runBatchBenchmarks` in the same file (`:389`, `:401`, `:418`, `:423`, `:433`, `:439`) uses
`b.Fatal(err)` for exactly this reason; the new arms are inconsistent with the established pattern in
their own file.

**Fix:** fail loudly. A one-shot usability probe per provider plus checked calls in the hot loop
(the branch is free after the first successful op, and `b.Fatal` never fires on a healthy provider):

```go
func runPrimitiveBenchmarks(b *testing.B, c cache.BatchCache[string, string]) {
	// sqlite defers open errors to the first operation — probe once so a broken
	// provider fails the benchmark instead of reporting fabricated ns/op.
	if err := c.Set(b.Context(), "prim_smoke", "v"); err != nil {
		b.Fatalf("provider unusable: %v", err)
	}
	if _, err := c.Get(b.Context(), "prim_smoke"); err != nil {
		b.Fatalf("provider unusable: %v", err)
	}

	b.Run("Get", func(b *testing.B) {
		keys := prePopulateKeys(b, c, 1, "prim_get")

		for i := 0; i < b.N; i++ {
			if _, err := c.Get(b.Context(), keys[0]); err != nil {
				b.Fatal(err)
			}
		}
	})
	// ... same treatment for Set/Delete/GetOrSet
}
```

## Warnings

### WR-01: The "Delete" arm measures Set+Delete, so the reported number is not the delete cost

**File:** `cache/bench_test.go:727-733`

**Issue:** Each iteration performs a `Set` and then a `Delete`. On the file-backed sqlite cache both
are separate autocommit transactions (each with its own fsync), so the published "Delete" figure is
roughly double the real delete cost and is not comparable with `Get`/`Set`. It also re-inserts rows
without a `b.ResetTimer()`, so ring warm-up is folded into the average.

**Fix:** name the measurement honestly and reset after seeding:

```go
	b.Run("SetDelete", func(b *testing.B) { // Set+Delete, not Delete alone
		keys := prePopulateKeys(b, c, primitiveKeyRing, "prim_del")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := keys[i%primitiveKeyRing]
			if err := c.Set(b.Context(), key, "v"); err != nil {
				b.Fatal(err)
			}
			if err := c.Delete(b.Context(), key); err != nil {
				b.Fatal(err)
			}
		}
	})
```

A delete cannot be measured in isolation without re-seeding the key, so renaming (or documenting the
composition in the arm's comment) is the honest option.

### WR-02: "GetOrSet" silently degrades to the warm hit path and the doc comment does not say so

**File:** `cache/bench_test.go:735-740` (comment at `:709-712`)

**Issue:** With a `primitiveKeyRing`-sized key space and no TTL, only the first 1024 calls ever run
the setter; every later call is a hit. The helper's doc comment claims it "measures the single-key
operations (Get/Set/Delete/GetOrSet)", which reads as the full GetOrSet cost (miss → setter →
singleflight → store). The result therefore changes with `b.N` and hides the singleflight/setter path
entirely — the number is only meaningful next to the `Get` arm, and nothing in the file says that.

**Fix:** state the caveat in the doc comment, and/or add a miss arm with unique keys:

```go
	// GetOrSet measures the warm hit path: the bounded ring means only the first
	// primitiveKeyRing calls run the setter. Compare it against Get, not as the
	// cost of a miss.
	b.Run("GetOrSet", func(b *testing.B) { ... })
```

### WR-03: The CGO-free step prints its evidence but never asserts it

**File:** `.github/workflows/go.yml:31-37`

**Issue:** `go env CGO_ENABLED` and `go list -m modernc.org/sqlite modernc.org/libc` only write to
the log. Nothing in the step fails if `CGO_ENABLED` resolves to `1` (someone drops the job-level env,
a `setup-go` change, a step-level override added later) or if the driver is swapped for a cgo-backed
one. The green check therefore does not, by itself, prove the phase's deliverable; it proves only
that the code compiles in whatever configuration actually ran. Calling `go list -m` a "pin" (as the
commit message does) overstates it — it is the Go equivalent of `echo`.

**Fix:** turn both into gates. `go list -deps` also gives a direct, positive proof that no package in
the build graph carries cgo files:

```yaml
      - name: Build (CGO-free)
        shell: bash
        run: |
          cgo=$(go env CGO_ENABLED)
          test "$cgo" = "0" || { echo "::error::CGO_ENABLED=$cgo, expected 0"; exit 1; }
          go build ./...
          go vet ./...
          # Positive proof: no package in the dependency graph compiles cgo files.
          cgo_pkgs=$(go list -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./...)
          test -z "$cgo_pkgs" || { echo "::error::cgo packages in graph: $cgo_pkgs"; exit 1; }
```

### WR-04: The macOS race step forces cgo even though darwin is the one OS where `-race` does not need it

**File:** `.github/workflows/go.yml:48-57`

**Issue:** `CGO_ENABLED: "1"` is applied to both non-Windows runners, but `cmd/go` explicitly exempts
darwin (`"Note: On macOS, -race does not require cgo."`, `cmd/go/internal/work/init.go:194`; verified
locally: `CGO_ENABLED=0 go test -race ./cache/sqlite/` passes on darwin). The consequence is twofold:
macOS loses the strongest available cgo-free proof (a `-race` build *with cgo disabled*), and the
macOS step now depends on a C toolchain it does not need — a future runner image without clang turns
that step red for no reason. The comment's own wording ("on every non-darwin OS") says the value
should be conditional; the code is not.

**Fix:**

```yaml
        env:
          # -race requires cgo only outside darwin (cmd/go init.go); keep macOS
          # CGO-free so its -race run still proves the CGO-free claim.
          CGO_ENABLED: ${{ runner.os == 'macOS' && '0' || '1' }}
```

## Info

### IN-01: `ExampleNew_withPath` leaks the first cache handle on every error path

**File:** `cache/sqlite/example_test.go:52-63`

**Issue:** `first` is only closed on the success path (`:59`). Each early `return` above it leaves the
`*sql.DB`, its file handle and the WAL sidecars open; the deferred `os.RemoveAll` (`:47`) then runs
against an open database, and on Windows the removal fails — silently, because that error is also
discarded. The example is exactly the artifact users copy, so it should model the safe pattern.

**Fix:** `CloseFunc` is idempotent (`sqlite.go:261-274`), so register the close immediately:

```go
	first := sqlite.New[string, string](sqlite.WithPath(path))
	defer func() { _ = first.Close() }() // idempotent; safe after the explicit Close below
```

(and keep the explicit `first.Close()` before reopening, since the ordering is the point of the example).

### IN-02: The canceled-MGet caveat is stated twice in the package comment

**File:** `cache/sqlite/doc.go:42-43` and `:49-50`

**Issue:** The pre-existing "Data and state" paragraph already says "A canceled MGet returns the
partial results gathered so far — the frozen MGet signature has no error channel"; the new batch
paragraph repeats it almost verbatim. Two copies of one caveat in a 65-line package comment is how
doc drift starts (one gets updated, the other does not).

**Fix:** keep the sentence in the batch paragraph (where it belongs now) and shorten the earlier one
to a cross-reference, or drop it from the batch paragraph.

### IN-03: MDel is described as "idempotent" next to an all-or-nothing MSet, but it is fail-fast and partial

**File:** `cache/sqlite/doc.go:51-54`; `README.md:92-93`

**Issue:** The MSet sentence spends three lines on whole-batch rollback, so the following "MDel deletes
in chunks and is idempotent for duplicate and missing keys" reads as the same atomicity guarantee.
`MDelFunc` (`batch.go:215-233`) is actually fail-fast: the first failing chunk returns an error and
earlier chunks stay deleted. The idempotency claim is true and is what makes that safe — but the doc
never says so, and the README compresses it to "idempotent chunks".

**Fix:** one clause — "MDel deletes in chunks and is idempotent for duplicate and missing keys; the
first failing chunk returns an error and earlier chunks remain deleted (retry is safe)".

### IN-04: The `coverage` job was left out of the hardening

**File:** `.github/workflows/go.yml:59-67`

**Issue:** The file is under review for "delivery hardening", yet the second job still runs
`actions/checkout@v3` / `actions/setup-go@v3` (the `test` job is on `v4`/`v5`), sets no
`CGO_ENABLED=0`, and therefore measures coverage with cgo enabled on a different configuration than
the one being certified. No action is SHA-pinned anywhere in the file, which is the standard
supply-chain hardening for `pull_request` workflows that render fork-supplied code paths.

**Fix:** align the job on `checkout@v4` / `setup-go@v5`, add `env: CGO_ENABLED: "0"` so coverage runs
the same configuration as the test matrix, and consider SHA pinning with a Dependabot `github-actions`
group.

### IN-05: Nothing in CI ever exercises the newly registered benchmarks

**File:** `.github/workflows/go.yml` (whole file), `cache/bench_test.go:540-555`, `:746-768`

**Issue:** `go test ./...` compiles the benchmark functions, so a compile break is caught, but no step
runs them — a runtime failure (a `b.Fatal` fired by a bad seed, a cleanup-ordering bug on a runner
with a different temp-dir layout, a hang) would only surface when a human runs `make benchmark`. That
is also how CR-01 stayed invisible: nothing ever observes the numbers.

**Fix:** add a cheap smoke step (sub-second per family, no Docker because the Docker arms skip without
`DOCKER_HOST`):

```yaml
      - name: Benchmark smoke
        run: go test ./cache/ -run '^$' -bench 'BenchmarkPrimitives|BenchmarkBatch/sqlite' -benchtime=1x -count=1
```

### IN-06: The primitive arms bound key growth; the batch arms do not, so file-mode numbers drift with `b.N`

**File:** `cache/bench_test.go:364-447` (batch) vs `:30`, `:709-741` (bounded ring)

**Issue:** The comment at `:28-30` states the principle — bound the key space so file mode measures the
operation, not b-tree growth — and the primitive arms follow it, but the batch arms generate globally
unique keys (`%s_mset_native_%d_%d_%d`, `%s_mdel_pk_%d_%d_%d`) for every iteration. On
`sqlite/File` those arms therefore mix index growth and WAL/checkpoint cost into the per-op figure, and
the numbers are not comparable across `-benchtime` settings. Flagged as a measurement-consistency
note only; the underlying cost is performance work and out of scope for this review.

**Fix:** either state in `runBatchBenchmarks` that file-mode batch numbers include growth by design, or
give the sqlite file family a bounded-key variant analogous to `primitiveKeyRing`.

---

_Reviewed: 2026-10-26T22:58:42Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
