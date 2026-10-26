# Phase 17: Delivery Hardening - Research

**Researched:** 2026-10-26
**Domain:** Go 1.26 build/test delivery — CGO-free CI matrix, coverage gates, package docs/examples, benchmark registration
**Confidence:** HIGH (every load-bearing claim was executed locally on the repo or read from the Go 1.26.4 toolchain source / GitHub Actions documentation; the two Windows-runtime claims are the only ones resting on prior CI evidence rather than a local run)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### CGO-Free CI Proof (QUAL-01)
- **D-01:** `CGO_ENABLED=0` is set as **job-level `env`** on the existing three-OS `test` job, so the build, the unit tests, AND the two-process E2E all run CGO-free — the E2E is the milestone's headline promise, so proving it under `CGO_ENABLED=0` is the strongest available evidence.
- **D-02:** The existing `SQLite race detector` step sets **`CGO_ENABLED=1` in its own step env**. `-race` requires cgo and a C toolchain (which the ubuntu/macos runners provide); it cannot run under the job-level `CGO_ENABLED=0`. The Windows gate on that step is untouched.
- **D-03:** CI evidence is an explicit step named **`Build (CGO-free)`** that runs `go env CGO_ENABLED` (so the `0` is visible in the log) followed by `go build ./...`, placed before the existing `go test -v ./...`. The job env is the enforcement; the build step is the visible proof. Matches the `CGO_ENABLED=0 go build ./...` mode recorded in milestone research.
- **D-04:** The Windows `-race` broadening stays **deferred** — it is a Phase 16 reviewer-flagged follow-up with no requirement behind it, and QUAL-01 asks only for CGO-free build+test on three OSes. — **Reversibility:** reversible.

#### Benchmarks (QUAL-04)
- **D-05:** Reuse the v1.6 batch helpers (`runBatchBenchmarks`, `prePopulateKeys`; native vs per-key at sizes 1/10/100/1000) for sqlite, **plus** a new primitive-ops benchmark comparing **mem and sqlite** across `Get`/`Set`/`GetOrSet`. No singleflight herd arm for sqlite: it measures the shared dedup helper, not the backend.
- **D-06:** sqlite is measured in **two named sub-benchmark families** — `Memory` (`:memory:`) and `File` (`WithPath` on `b.TempDir()`). Memory keeps the mem comparison apples-to-apples; file measures the durable path users actually ship.
- **D-07:** Benchmark code lives in **`cache/bench_test.go`** (`package cache_test`) and registers sqlite as an **always-run provider like mem — no Docker gate**. Placement follows from helper reuse: `runBatchBenchmarks` and `prePopulateKeys` are unexported in `cache_test`, so a `cache/sqlite/bench_test.go` file would duplicate or force export. `./cache/...` in `make benchmark`/`benchmark-quick` picks it up automatically; no CI benchmark step is added.
- **D-08:** Ship **benchmark entries only** — no measured numbers in the README or in phase artifacts, so nothing goes stale when a dependency or machine changes. — **Reversibility:** reversible.

#### Docs (QUAL-03)
- **D-09:** `cache/sqlite/doc.go` keeps its current structure and gains one **batch paragraph**: `MGet` silently skips missing, expired, and undecodable entries (already partly stated via the canceled-MGet sentence), `MSet` writes one transaction with one TTL and rolls back the whole batch on any error, `MDel` is chunked and idempotent. The four QUAL-03 topics (TTL, reopen persistence, `:memory:`, multi-process constraints) are **re-verified against shipped code** as part of the same pass.
- **D-10:** A **second runnable example** is added to `cache/sqlite/example_test.go` demonstrating durability: `WithPath` on `t.TempDir()` → `Set` → `Close` → reopen → `Get`. The existing in-memory `Example()` stays as the entry-point example.
- **D-11:** README changes are **verification + one clause**: confirm the package-index row, provider-table row, and sqlite paragraph match shipped behavior, and add a single batch-surface clause mirroring doc.go. No new sections, no duplicated examples, no performance numbers.

#### Test Completeness (QUAL-02)
- **D-12:** QUAL-02 closes by **audit, not churn**: map each named class — unit, `:memory:`, reopen/persistence, concurrent, sweep — to the tests that already cover it, run `make coverage-quick`, and record the mapping as evidence. A test is added only if a named class genuinely has none. No coverage-margin hardening.
- **D-13:** **Coverage gate inputs stay unchanged.** `make coverage`'s E2E merge remains `./cache/`-root-only, so the `e2e`-tagged sqlite two-process tests never enter the profile. Rationale to be recorded so a later agent does not "fix" it: the unit gate already passes at ~92.5% with no override, and E2E correctness is proven by its dedicated CI step.
- **D-14:** **No new guard machinery** for the "no override" promise. A passing gate is the evidence, and any future `cache/sqlite` override appears as a reviewable config diff.

### the agent's Discretion
- Exact CI step name/ordering beyond D-03, and whether `go vet` joins the build step.
- Whether `runBatchBenchmarks` needs a signature change (label/cleanup hook) to host a second provider cleanly; how the file-mode cache is closed (`b.Cleanup`) and whether file-mode uses a fresh temp path per sub-benchmark.
- Which primitive operations the comparison benchmark includes beyond `Get`/`Set`/`GetOrSet` (e.g. `Delete`), and the exact sub-benchmark naming.
- Wording of the doc.go batch paragraph and README clause; the new example's name (`ExampleNew_withPath` vs package-level `Example_newWithPath`) and whether it asserts `errors.Is(err, cache.ErrMiss)`.
- The format of the five-class coverage mapping evidence in the plan/SUMMARY.

### Deferred Ideas (OUT OF SCOPE)
- **Windows `-race` broadening** — needs a mingw toolchain setup step and has no requirement behind it. Stays a documented follow-up in the workflow comment (Phase 16 reviewer-flagged).

### Reviewed Todos (not folded)
None — `todo.match-phase 17` returned no matches.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| QUAL-01 | Package builds and tests with `CGO_ENABLED=0` on Linux, macOS, Windows; `modernc.org/sqlite` v1.60.1 pinned with the exact `modernc.org/libc` pin | Q1 + Q2 below. Verified locally: `CGO_ENABLED=0 go build ./...`, `go test ./...`, and `-tags=e2e -run TestTwoProcess` all green; cross-compiled windows/amd64 + linux/amd64 + linux/arm64 builds and the e2e test binary; `go list -m` confirms `v1.60.1` / `v1.77.1`. `-race` needs step-level `CGO_ENABLED=1` on Linux (source-verified) — D-02 is correct and required. |
| QUAL-02 | Tests cover unit, `:memory:`, reopen/persistence, concurrent, sweep; coverage gates pass with no `.testcoverage-quick.yml` override | Q3 + the "Test Scenario Coverage Map" section. `make coverage-quick` reproduces PASS: package 92.8%, total 81.8%, all three thresholds PASS; CI-profile repro (`-coverpkg=./...` + `.testcoverage.yml`) also PASS. `cache/sqlite` appears in neither config's override list. Every named class already has a named test with a file:line citation. |
| QUAL-03 | `doc.go` documents the provider (TTL, reopen persistence, `:memory:`, multi-process) and a runnable example exists; README index updated | Q5 + docs section. All four topics verified present in `doc.go`; the batch surface is the only gap (three facts + chunk size, all traced to `batch.go`). A validated `ExampleNew_withPath` (runs green, lint-clean) is included verbatim. README row/table/paragraph already exist; the one batch clause attaches to the sqlite paragraph at `README.md:87-90`. |
| QUAL-04 | Benchmark entries measure sqlite against mem and per-key loops, following v1.6 conventions | Q4 + benchmark section. Exact helper signatures reported; a validated registration block (lint 0 issues, `go test -bench` runs 12 sqlite batch sub-benchmarks × 2 families) is included verbatim, plus a validated primitive-ops benchmark. |

</phase_requirements>

## Summary

This phase changes no provider code. It is a **delivery-evidence phase**: one workflow file, one benchmark file, one package doc comment, one example function, one README paragraph — and an audit record proving the five test classes and the coverage gates are already satisfied.

Three findings change how the plan should be written:

1. **`-race` under `CGO_ENABLED=0` is impossible on Linux and fine on macOS.** The Go 1.26.4 `cmd/go` source gates the race build on cgo for every `GOOS` except `darwin` (`... && (cfg.Goos != "darwin" || ASan || MSan)`), so on ubuntu-latest the job-level `CGO_ENABLED=0` would break the existing race step with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` (exit 2). D-02's step-level override is not a nicety — it is mandatory on the Linux leg and harmless on macOS. `.planning/research/STACK.md`'s "`-race` still works (pure Go)" claim is **wrong for Linux** and right only by accident on macOS; it must be superseded, not merely annotated.

2. **A multi-command `run:` block is silently fail-open on Windows.** The Windows default shell is `pwsh`, and GitHub appends only the *last* exit code (`exit $LASTEXITCODE`) to the script — so `go env CGO_ENABLED` followed by `go build ./...` in one step can report success while the build failed. The `Build (CGO-free)` step (D-03) must either set `shell: bash` (Git-for-Windows bash runs with `-eo pipefail`) or split into separate steps. This is the single most likely way the phase "proves" something that is not true.

3. **Everything else is already green — the phase's job is to record it, not to earn it.** `CGO_ENABLED=0` build, full unit suite, both two-process E2E arms, and both coverage profiles pass locally on Go 1.26.4 today; `cache/sqlite` sits at 92.8% with no override; `go.mod` carries both pins. The plan's real work is the four additive edits plus a reproducible verification log.

**Primary recommendation:** Write the plan as four small, independently verifiable tasks — (a) workflow env + `Build (CGO-free)` step (`shell: bash`) + step-level race override; (b) sqlite batch families + primitive-ops benchmark in `cache/bench_test.go`, using the validated blocks below; (c) `doc.go` batch paragraph + `ExampleNew_withPath` + README clause; (d) a QUAL-02 audit record mapping the five classes to existing tests with the `make coverage-quick` numbers — then gate the phase on a single CI run where all three OS legs show `CGO_ENABLED=0` in the build step and the ubuntu race step runs green.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CGO-free build proof (`go env CGO_ENABLED` + `go build ./...`) | CI workflow (job-level `env` + one step) | Go toolchain (`cmd/go`) | The property is environmental, not code; only the runner can prove it. The Go toolchain is the enforcement point. |
| CGO-free unit-test proof | CI workflow (job-level `env` → `go test -v ./...`) | `cache/sqlite` test suite | Same env propagates; no test change needed. |
| CGO-free two-process E2E proof | CI workflow (job-level `env` → e2e step) | `cache/sqlite/twoprocess_e2e_test.go` | Children are re-executed `os.Args[0]`, so they inherit the CGO-free binary — no rebuild, nothing to change. |
| Race detection | CI workflow (step-level `env: CGO_ENABLED: "1"`) | Go race runtime + gcc/clang on runner | `-race` is a cgo build mode on every non-darwin OS. Ownership sits with the step, not the job. |
| Coverage gate evidence | Local `make coverage-quick` (pre-commit rule) | CI `coverage` job (`go-test-coverage@v2`) | Two distinct profiles; both already pass. No config change is in scope (D-13/D-14). |
| Package documentation | Package source (`cache/sqlite/doc.go`) | README (index + paragraph) | Godoc is the canonical surface; README is a discovery mirror. |
| Runnable usage proof | `cache/sqlite/example_test.go` (external test package) | `go test` example runner | Executable docs must live in `_test.go` to be run by `go test`. |
| Benchmark measurement | `cache/bench_test.go` (`package cache_test`) | Make targets `benchmark` / `benchmark-quick` | Helpers are unexported in `cache_test`; the harness tier owns provider registration. |

## Standard Stack

**This phase installs zero new packages.** No dependency is added, removed, or upgraded; the Package Legitimacy Gate is therefore not triggered for new installs — but the two existing sqlite pins are re-verified below because QUAL-01 names them explicitly.

### Core (already present — verify, do not change)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | `v1.60.1` (direct require) | CGO-free `database/sql` driver | Only mature pure-Go SQLite driver; builds with `CGO_ENABLED=0` on all three matrix OSes. `[VERIFIED: go.mod:19]` |
| `modernc.org/libc` | `v1.77.1` (indirect, exact pin) | Runtime libc port used by the transpiled SQLite | Upstream (cznic/sqlite#177) requires downstream `go.mod` to pin the same libc version; `go mod tidy` MVS-resolves it today. `[VERIFIED: go.mod:119-120]` |

Verbatim from `go.mod`:

```go
go 1.26.4
...
	modernc.org/sqlite v1.60.1
...
	// Pinned exact per cznic/sqlite#177: modernc.org/sqlite v1.60.1 requires this libc version.
	modernc.org/libc v1.77.1 // indirect
```

Resolution confirmed this session:

```
$ go list -m modernc.org/sqlite modernc.org/libc
modernc.org/sqlite v1.60.1
modernc.org/libc v1.77.1
```

### Supporting (tooling already installed locally / on runners)

| Tool | Version observed | Purpose | When to Use |
|------|------------------|---------|-------------|
| Go toolchain | `go1.26.4` (via `GOTOOLCHAIN=auto`, downloaded on demand) | Build/test/bench | Every verification step; `go.mod` pins `go 1.26.4` and CI uses `go-version-file: go.mod` |
| `golangci-lint` | `2.14.0` | Lint gate (repo pre-commit) | Before every commit; must stay at 0 issues |
| `go-test-coverage` | `v2.18.9` | Coverage gate | `make coverage-quick` locally; `vladopajic/go-test-coverage@v2` in CI |
| `gocovmerge` | present | Merge E2E + unit profiles | `make coverage` only (needs Docker) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Job-level `env: CGO_ENABLED: "0"` | A dedicated `cgo-free` job per OS | Doubles matrix cost and duplicates the test suite; CONTEXT already declines it (declined-and-recorded alternative). |
| `shell: bash` on the build step | Separate `go env CGO_ENABLED` step + `go build` step | Also correct; two steps instead of one, but avoids depending on Git-Bash availability. Either is acceptable — see Pitfall 2. |
| New top-level `BenchmarkBatchSQLite` | sqlite as a `b.Run` family inside `BenchmarkBatch` | Both lint-clean and both run (validated). Inside `BenchmarkBatch` gives names directly comparable with the mem family; a separate function is a smaller blast radius. Recommendation: **inside `BenchmarkBatch`** (see Q4). |

## Package Legitimacy Audit

No packages are installed in this phase. `modernc.org/sqlite` v1.60.1 and `modernc.org/libc` v1.77.1 are pre-existing pins carried from Phase 15; this phase only asserts their presence (QUAL-01) and verifies them by `go list -m`. No `package-legitimacy check` run is required.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none — no installs this phase)* | — | — | — | — | — | — |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Verification Log — what was actually executed (2026-10-26)

All runs on `darwin/arm64`, repo at `main` @ `d9ac94c`, `GOTOOLCHAIN=go1.26.4` (the CI toolchain). Raw outputs, unedited:

| # | Command | Result |
|---|---------|--------|
| 1 | `CGO_ENABLED=0 go build ./...` | exit 0 |
| 2 | `CGO_ENABLED=0 go test ./cache/sqlite/ -count=1` | `ok github.com/guionardo/go/cache/sqlite 5.383s` |
| 3 | `CGO_ENABLED=0 go test ./...` (CI "Run tests" equivalent) | all 25 packages `ok`, exit 0 |
| 4 | `CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -timeout=180s -v` | `--- PASS: TestTwoProcessContention (0.03s)`, `--- PASS: TestTwoProcessCrashRecovery (0.03s)` |
| 5 | `CGO_ENABLED=0 go vet ./...` | clean, exit 0 |
| 6 | `CGO_ENABLED=0 make swapper` | 4 cross-builds OK (linux/amd64, darwin/amd64, darwin/arm64, windows/amd64) |
| 7 | `CGO_ENABLED=0 go test -race ./cache/sqlite/ -count=1` (darwin) | `ok ... 6.499s` — **works on macOS**. Corroborated: a race binary built under `CGO_ENABLED=0` carries 292 `tsan` symbols and the `WARNING: DATA RACE` runtime string, i.e. the detector is genuinely linked, not silently skipped |
| 8 | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c ./cache/sqlite/` | `go: -race requires cgo` — **exit 2** |
| 9 | `GOOS in {windows/amd64, linux/amd64, linux/arm64} CGO_ENABLED=0 go build ./...` | all `build OK` |
| 10 | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -c -tags=e2e ./cache/sqlite/` | test binary produced OK |
| 11 | `make coverage-quick` | `File coverage threshold (70%) satisfied: PASS`, `Package coverage threshold (80%) satisfied: PASS`, `Total coverage threshold (75%) satisfied: PASS`, `Total test coverage: 81.8% (2104/2572)`; `cache/sqlite` = **92.8%** |
| 12 | `go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...` + `go-test-coverage --config=./.testcoverage.yml` (CI coverage-job repro) | all three thresholds PASS; `Total test coverage: 81.8% (2105/2572)` |
| 13 | Validated example probe: `ExampleNew_withPath` using `os.MkdirTemp` + `os.RemoveAll` | `--- PASS: ExampleNew_withPath (0.00s)`; `golangci-lint run ./cache/sqlite/` → `0 issues` |
| 14 | Validated benchmark probe: sqlite `Memory`/`File` families via `runBatchBenchmarks` inside `BenchmarkBatch`, plus a primitives benchmark | benchmarks ran (`BenchmarkBatch/sqlite/Memory/MGet/Size1000/native 1458125 ns/op`, `…/per-key 5170458 ns/op`, …); `golangci-lint run ./cache/` → `0 issues` |

Files 13/14 were scratch files, created for the probe and **deleted afterwards** — `git status --porcelain` was re-checked clean before writing this document.

## Q1 — `CGO_ENABLED=0` × `-race` interaction (the decisive finding)

**The rule, verbatim from the Go 1.26.4 toolchain source** (`$GOROOT/src/cmd/go/internal/work/init.go:193-204`, `$GOROOT=/Users/guionardo/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.4.darwin-arm64`):

```go
	// Check that cgo is enabled.
	// Note: On macOS, -race does not require cgo. -asan and -msan still do.
	if !cfg.BuildContext.CgoEnabled && (cfg.Goos != "darwin" || cfg.BuildASan || cfg.BuildMSan) {
		if runtime.GOOS != cfg.Goos || runtime.GOARCH != cfg.Goarch {
			fmt.Fprintf(os.Stderr, "go: %s requires cgo\n", modeFlag)
		} else {
			fmt.Fprintf(os.Stderr, "go: %s requires cgo; enable cgo by setting CGO_ENABLED=1\n", modeFlag)
		}

		base.SetExitStatus(2)
		base.Exit()
	}
```

The identical block exists in Go 1.27.0 (`/opt/homebrew/Cellar/go/1.27.0/libexec/src/cmd/go/internal/work/init.go:193-204`), so the rule is not a version artifact.

**Answers to the four sub-questions:**

1. **Does `go test -race` refuse to run under `CGO_ENABLED=0` on Go 1.26?** Yes on every `GOOS` except darwin. Exact text: native build → `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`; cross build → `go: -race requires cgo`. Exit status **2**. Reproduced locally (cross-compile path, since the host is darwin): `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c ./cache/sqlite/` → `go: -race requires cgo`. On ubuntu-latest the host==target branch applies, so the longer message with the `CGO_ENABLED=1` hint is the one CI will print.
2. **Is a step-level `env:` override sufficient to re-enable cgo inside a job whose `env:` sets it to 0?** Yes. GitHub's workflow syntax reference states: *"An environment variable defined in a step will override job and workflow environment variables with the same name, while the step executes."* `[CITED: docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax]` A plain `env: { CGO_ENABLED: "1" }` on the race step is the documented mechanism — no `GITHUB_ENV` write, no shell prefix needed.
3. **Does the two-process E2E pass under `CGO_ENABLED=0` with `modernc.org/sqlite`?** Yes — executed (`--- PASS: TestTwoProcessContention`, `--- PASS: TestTwoProcessCrashRecovery`). It also *cannot* fail for an inheritance reason: `startChild` re-executes the already-built test binary via `exec.Command(os.Args[0])` with `cmd.Env = append(os.Environ(), …)` (`cache/sqlite/twoprocess_e2e_test.go:302-322`), so no child rebuild occurs and the CGO-free property is structural, not environmental. This is the strongest single artifact for QUAL-01.
4. **Reconcile `.planning/research/STACK.md`.** The claims are:
   - line 62: `` | `CGO_ENABLED=0` | CI build mode | … needs **no gcc/msys2/TDM-GCC**. `-race` still works (pure Go). Cross-compilation works. | ``
   - line 152: `` - **Cross-compile + race detector both work** (pure Go), unlike CGO drivers. ``

   **The correct reading:** "pure Go" describes the *provider/driver* — it needs no cgo to build or run. It does **not** describe `-race`, which is a separate build mode that requires cgo on every non-darwin GOOS regardless of how pure the dependencies are. So:
   - Cross-compilation: STACK.md is right (`CGO_ENABLED=0` cross-builds verified for windows/linux).
   - `-race` "still works (pure Go)": right on macOS only; **wrong on Linux**. Phase 16's workflow comment already contradicts the claim in practice — `go.yml:34-36` says *"`-race` requires a C toolchain, which Linux and macOS runners provide"*, which is exactly why the step exists.
   - Consequence for the plan: **do not** treat the race gate as something that survives a job-wide `CGO_ENABLED=0`. The step-level override (D-02) is load-bearing. Record a one-line supersession in the plan/SUMMARY so the stale STACK.md sentence is not used later to "simplify" the workflow.

**Bonus finding (useful for the plan's framing):** the macOS leg does **not** need the override, and the Windows leg's `-race` remains impossible without mingw (D-04, unchanged). So the override is exactly one OS-legs ' worth of necessity — ubuntu — and is a harmless no-op on macOS. Do not add `if: runner.os == 'Linux'` gymnastics: keeping one override on the existing `runner.os != 'Windows'` step is simpler and self-documenting.

## Q2 — CI matrix mechanics

**Toolchain selection.** `actions/setup-go@v5` with `go-version-file: go.mod` reads the `go` directive (`go 1.26.4`) and installs that toolchain. `[CITED: github.com/actions/setup-go README — "Path to go.mod, go.work, .go-version, or .tool-versions file"; "if both go-version and go-version-file are provided, go-version takes precedence"]` The repo has **no `toolchain` directive** in `go.mod` (verified: lines 1-3 are `module`, blank, `go 1.26.4`), so there is no v5-vs-v6 resolution difference to worry about. `GOTOOLCHAIN=auto` is the local default, which is why `GOTOOLCHAIN=go1.26.4` is the right prefix for local reproductions of CI.

**Does job-level `CGO_ENABLED=0` break anything else in the job?**

| Step | Under job-level `CGO_ENABLED=0` | Evidence |
|------|--------------------------------|----------|
| `make swapper` (cross-compiles 4 binaries) | **No — works, and is arguably safer.** `CGO_ENABLED=0` is what makes cross-OS builds toolchain-free in the first place. | `CGO_ENABLED=0 make swapper` → 4× `go build` OK (probe #6) |
| `go test -v ./...` | **No.** Full suite green; Docker-backed providers self-skip on missing `DOCKER_HOST`/`docker info`. | probe #3 |
| `go test -tags=e2e …` | **No.** Both arms PASS. | probe #4 |
| `go test -race …` | **Yes on ubuntu** — this is the only step that breaks, hence D-02. | probes #7/#8 + source above |
| `actions/setup-go@v5` caching | No effect. The build cache keys include cgo mode internally; Go handles that. | — |

**Windows caveat for `go build ./...` on a library repo.** No problem in itself: with multiple packages (or a single non-main package) `go build` compiles and **discards** results, so nothing is written next to the source and no `.exe` appears. The repo does contain one `main` package (`cmd/example-updater`), but it is compiled as part of a multi-package pattern, so still discarded. `[VERIFIED: docs semantics + probe #9 windows/amd64 build OK]`

**The real Windows caveat is shell behaviour, not Go.** GitHub's default Windows shell is `pwsh -command ". '{0}'"`, and the docs state: *"we append `if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }` to powershell scripts so action statuses reflect the script's **last** exit code."* `[CITED: docs.github.com …/workflow-syntax#exit-codes-and-error-action-preference]` Only the **final** command in a multi-line block determines the step's status. So this shape is safe only by accident:

```yaml
run: |
  go env CGO_ENABLED
  go build ./...
```

— the build happens to be last, so its failure **is** caught. But the moment a discretion item is appended after it, the step becomes fail-open:

```yaml
run: |
  go env CGO_ENABLED
  go build ./...
  go vet ./...
```

— now `go vet` is last, and a failing `go build` is silently reported as success. **Recommendation:** set `shell: bash` on the step. GitHub documents that `bash` on Windows uses *"the bash shell included with Git for Windows"* invoked as `bash --noprofile --norc -eo pipefail {0}`, which fails fast on any command. `[CITED: same page, "Supported shells" table]` Alternatively, split into one command per step. Pick one; do not leave a 3-command `pwsh` block.

**Other mechanics worth pinning in the plan:**
- Quote the env value: `CGO_ENABLED: "0"`. GitHub stringifies numbers in `env`, but quoting keeps the YAML unambiguous and matches the repo's existing style (`go-version: "1.26"` in `release.yml:21`).
- The `coverage` job is a **separate job** with no `env:` block — it is untouched by this phase, and its `setup-go@v3` (no `go-version-file`) keeps using whatever the runner ships. Not in scope; noted in Open Questions.
- Do **not** add a benchmark step to CI (D-07). `make benchmark` stays local.

## Q3 — Coverage tooling

**Thresholds** — identical in both configs, no `cache/sqlite` entry in either:

`[VERIFIED: .testcoverage-quick.yml:3-6, :8-24]`

```yaml
threshold:
  file: 70
  package: 80
  total: 75
```

The full override list is `cache/memcache`, `cache/postgres`, `cache/redis`, `cache/valkey`, `cmd/example-updater`, `release/swapper`, `mid` (50), `release` (70). `.testcoverage.yml` carries the same eight rules (lines 30-50) with explanatory comments. **`cache/sqlite` is absent from both** — read in full this session.

**Does CI's `threshold-total: 75` bypass the config's file/package thresholds?** **No.** The action's `action.yml` defines each threshold as an *independent* input whose behavior is "Overrides value from configuration", all defaulting to `-1` (meaning "not set"):

```yaml
  threshold-file:
    description: Minimum coverage percentage required for individual files. Overrides value from configuration.
    required: false
    default: -1
  threshold-package:
    description: Minimum coverage percentage required for each package. Overrides value from configuration.
    required: false
    default: -1
  threshold-total:
    description: Minimum overall project coverage percentage required. Overrides value from configuration.
    required: false
    default: -1
```
`[CITED: raw.githubusercontent.com/vladopajic/go-test-coverage/main/action.yml]`

Because only `threshold-total` is passed, it overrides *only* the total (with the same value `75` the file already states), and `file: 70` / `package: 80` come from `./.testcoverage.yml` unchanged. D-14's "no guard machinery" is sound.

**Current numbers (both profiles, executed):**

| Profile | Command | Result |
|---------|---------|--------|
| Local quick gate (`make coverage-quick`, no `-coverpkg`) | `go test ./... -coverprofile=./cover.out -covermode=atomic -count=1` → `go-test-coverage --config=./.testcoverage-quick.yml` | **PASS**; `cache/sqlite` **92.8%**; total **81.8% (2104/2572)** |
| CI coverage job repro (`-coverpkg=./...` + `.testcoverage.yml`) | `go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...` → `go-test-coverage --config=./.testcoverage.yml` | **PASS**; total **81.8% (2105/2572)** |

Note for the plan: the two profiles produce *different statement denominators* (2104 vs 2105) because `-coverpkg=./...` attributes cross-package statements. Do not try to reconcile the two numbers or "fix" one to match the other — they are different measurements of a passing gate. Also note `make coverage` (the E2E-merged profile) needs Docker, which is **not reachable on this machine** (`docker info` fails), so `make coverage-quick` is the only local gate available — which is exactly what AGENTS.md requires before each commit.

D-13's rationale, re-verified: `make coverage` merges E2E coverage from `./cache/` only (`Makefile:105-117`), so `cache/sqlite/twoprocess_e2e_test.go` (`//go:build e2e`) never enters that profile. Widening to `./cache/...` looks like an improvement and is not: the unit gate already passes at 92.8% with no override, and E2E correctness has its own dedicated CI step on all three OSes.

## Q4 — Benchmark harness reuse (QUAL-04, D-05/D-06/D-07)

**Exact shapes** (all read this session from `cache/bench_test.go`, `package cache_test`):

| Symbol | Signature | Line range | Notes |
|--------|-----------|-----------|-------|
| `skipIfNoDocker` | `func skipIfNoDocker(t testing.TB, reason string)` | 31-40 | Checks `DOCKER_HOST` **and** `docker info`; calls `t.Skipf`. Takes `testing.TB` so it works for `*testing.B`. |
| `prePopulateKeys` | `func prePopulateKeys(b *testing.B, c cache.BatchCache[string, string], n int, prefix string) []string` | 320-330 | Loops `for i := range n`, key = `fmt.Sprintf("%s_%d", prefix, i)`, `c.Set(b.Context(), …)`, `b.Fatal` on error. Returns keys. |
| `runBatchBenchmarks` | `func runBatchBenchmarks(b *testing.B, c cache.BatchCache[string, string], name string)` | 358-441 | Hard-codes `batchSizes := []int{1, 10, 100, 1000}`; runs `MGet`, `MSet`, `MDel`, each with `SizeN/native` and `SizeN/per-key`; all generated keys embed `name` (`fmt.Sprintf("%s_mget_%d", name, n)`, `…_mset_native_…`, `…_mdel_pk_…`). |
| `BenchmarkBatch` | `func BenchmarkBatch(b *testing.B)` | 446-682 | mem is registered **inline** (`c := mem.New[string, string](b.Context())`, 447-448) with the three blocks duplicated rather than via `runBatchBenchmarks`; then a `dockerProvider` slice loop (redis/valkey/memcache/postgres) calling `runBatchBenchmarks(b, c, p.name)` at 674-681. |
| `BenchmarkSingleflightGetOrSet` | `func BenchmarkSingleflightGetOrSet(b *testing.B)` | 156-314 | Herd benchmark; mem + Docker providers. **Not** a primitive-ops benchmark. |

**How mem is registered:** `c := mem.New[string, string](b.Context())` + `defer c.Close()` at the top of `BenchmarkBatch`, then three inline `b.Run` blocks. There is **no** existing primitive-ops (Get/Set/GetOrSet) benchmark anywhere in the file — it must be authored.

**What breaks if a second always-run provider is added — and what does not:**

| Concern | Finding |
|---------|---------|
| Key collisions across sub-benchmarks | **Handled if you pass a distinct `name`.** Every key `runBatchBenchmarks` generates embeds `name`, and each family gets its own cache instance anyway. Pass `"sqlite_memory"` / `"sqlite_file"` — never the same string twice. |
| `b.Context()` usage | Fine. `prePopulateKeys` and the per-op helpers all call `b.Context()`. Do **not** hoist `ctx := b.Context()` into a helper that then calls `prePopulateKeys` — `contextcheck` fires (`Function 'prePopulateKeys' should pass the context parameter`). Call `b.Context()` inline at each call site. Verified: the first probe tripped `contextcheck` ×2, the inline version is clean. |
| Cleanup of file-backed caches | Use `b.Cleanup(func() { _ = c.Close() })` **after** `b.TempDir()`. Testing cleanups run LIFO, so Close (registered later) runs before the TempDir `RemoveAll` — this is the Windows-correct ordering (PITFALLS Pitfall 6, line 150: *"In tests, `t.Cleanup(func() { _ = c.Close() })` immediately after `New`, so cleanup runs before `t.TempDir` removal (cleanups fire LIFO…)"*). A `defer` inside the `b.Run` closure also works, but `b.Cleanup` matches the house pattern and the PITFALLS guidance. |
| Fresh temp path per sub-benchmark | Recommended: create the path **inside** the `File` family closure (`filepath.Join(b.TempDir(), "batch.db")`). `b.TempDir()` returns a unique dir per benchmark-function invocation, so `-count=5` yields five fresh databases — no cross-run growth. |
| `runBatchBenchmarks` signature change | **Not needed.** The caller owns lifecycle: create the cache, register `b.Cleanup`, pass `(b, c, name)`. Adding a cleanup hook would be churn for zero benefit and would touch the Docker call sites. |
| Lint on an already-long function | `BenchmarkBatch` spans `cache/bench_test.go:446-682` and `funlen`/`gocognit`/`maintidx` are enabled — but adding a nested block inside it reported **0 issues**, because golangci-lint is configured with `new: true` / `new-from-rev: HEAD` and the function-declaration position is not new. Verified empirically (probe #14). `wsl_v5` does require a blank line before `b.Cleanup(...)` following a declaration — include it. |
| Runtime cost of `make benchmark` | `-benchtime=1s -count=5` over 48 new batch sub-benchmarks (2 families × 3 ops × 4 sizes × 2 modes) plus 12 primitive ones ≈ **5 minutes added**. `-benchtime=100ms -count=1` (`make benchmark-quick`) stays ~5 s. File-mode `MDel`/`MSet` at size 1000 is the expensive one (~20 ms/op observed). Acceptable; mention it so a future contributor does not think the suite hung. |

**Primitive-ops benchmark: authored, not reused.** Shape that passed lint and ran (see Code Examples). Recommended operation set: `Get`, `Set`, `Delete`, `GetOrSet` — `Delete` is cheap to add and completes the `Cache` interface surface. Two design notes:
- `Set`/`Delete`/`GetOrSet` should cycle through a **bounded key ring** (e.g. 1024 keys) rather than minting a unique key per iteration; otherwise file-mode `Set` grows the database without bound during a 1 s run and the benchmark drifts into measuring b-tree growth. A ring also makes mem-vs-sqlite an overwrite-vs-overwrite comparison.
- `GetOrSet` on a ring of pre-populated keys measures the **warm fast-path** (what production pays). Measuring the miss→setter path is a different question; if both are wanted, add an explicit `GetOrSet_Miss` arm rather than overloading one name.

## Q5 — Docs targets (QUAL-03, D-09/D-10/D-11)

### `cache/sqlite/doc.go` — what it omits

Current file: 56 lines, all verified accurate against shipped code for the four QUAL-03 topics:

| QUAL-03 topic | doc.go lines | Verified against |
|---------------|-------------|------------------|
| `:memory:` + location precedence | 19-26 | `options.go:41-45` (`WithPath`), `:49-53` (`WithMemory`), `sqlite.go:62` (`func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V]`) |
| TTL resolution / absolute expiry / no delete-on-read / sweep | 28-32 | Phase 15 tests (`TestReadsNeverDeleteRows`, `TestSweepOnOpen`) |
| Reopen persistence + WAL sidecar lifecycle | 34-36 | `TestReopenPersistence` (`sqlite_test.go:587-619`) |
| Multi-process constraints (same-host, 5 s busy timeout, bounded open retry, WAL/local storage only) | 37-45 | Phase 16 retry/E2E work; `twoprocess_e2e_test.go` |

**The single gap is the batch surface**, which appears only as an aside at the end of the data/state paragraph:

> `A canceled MGet returns the partial results gathered so far — the frozen MGet signature has no error channel.` (`doc.go:42-43`)

Behaviours that exist in code but are undocumented:

| Fact | Source |
|------|--------|
| `MGet` skips missing, expired and undecodable entries; a failed chunk is logged and skipped, remaining chunks still run | `batch.go:68-73` (func doc), `98-141` (`mgetChunk`: `logger().Debug("cache/sqlite: batch get undecodable entry skipped", …)`) |
| `MSet` marshals all values **before** the transaction opens; one TTL for the whole batch; single `BEGIN IMMEDIATE` transaction; any error rolls the whole batch back | `batch.go:143-148`, `183-209` |
| `MDel` deletes in chunks; duplicate and missing keys are no-ops (idempotent), fail-fast on first chunk error | `batch.go:211-214`, `215-233` |
| Chunk size is a fixed internal constant of 100 | `batch.go:19` — `const chunkSize = 100` |

Suggested paragraph (fits the existing voice, ≤120 chars/line for `lll`):

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

Place it **after** the "Data and state" paragraph (i.e. immediately before line 47's `// Long-running processes can tune…`) and leave the existing canceled-MGet sentence where it is — moving text enlarges the diff for no gain. `godoclint` only requires the `// Package sqlite provides …` prefix (present at line 1) and `.golangci.yml:34` enables it; the new paragraph must stay inside the package comment block, not become a detached comment.

### `cache/sqlite/example_test.go` — house style for a second example

Existing file (34 lines) establishes: `package sqlite_test`; early-return on error printing `"set failed:"`/`"get failed:"`; `defer func() { _ = c.Close() }()`; `// Output: value: hello`.

Repo-wide naming convention (`grep '^func Example'` across all `*_test.go`):

| Package | Examples |
|---------|----------|
| `cache/mem` | `ExampleNew`, **`ExampleNew_withTTL`** |
| `time_tools` | `ExampleParse`, `ExampleParse_rfc3339`, `ExampleSetLayouts` |
| `set` | `ExampleNew`, `ExampleSet_Union`, … |
| `cache/sqlite` | `Example` (only) |

So the idiomatic name for a constructor example with a variant suffix is **`ExampleNew_withPath`** — it mirrors `ExampleNew_withTTL` exactly and is a valid example suffix (lowercase suffix after `_`).

**`t.TempDir()` is NOT usable in an example function.** `go/doc` examples take no parameters, so there is no `*testing.T` in scope. The idiomatic substitute is `os.MkdirTemp` + `defer os.RemoveAll(dir)`. Validated: this passes `golangci-lint run ./cache/sqlite/` with **0 issues** — notably the `usetesting` linter does *not* flag `os.MkdirTemp` here, because no `*testing.T` is available to replace it with. Do not reach for `sqlite.WithName` (it writes into the user's real cache dir) or a fixed path.

Validated example (ran green: `--- PASS: ExampleNew_withPath (0.00s)`):

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

Discretion note on `errors.Is(err, cache.ErrMiss)`: the existing `Example()` does *not* demonstrate sentinel matching, and adding it here would import `cache` into the example for one assertion. Recommendation: skip it in the example; the miss contract is already covered by `TestClosedOps`/`TestParity` and stated in `doc.go:17`.

### `README.md` — where the clause attaches

| Surface | Location | Current content | Action |
|---------|----------|-----------------|--------|
| Package index row | `README.md:22` | `| [sqlite](#package-cache) | `cache/sqlite` | SQLite cache backend (embedded, survives restarts) |` | Verify only — accurate. |
| Provider table row | `README.md:85` | `| `cache/sqlite` | SQLite (embedded) | modernc.org/sqlite | Eager — local file, lazy errors |` | Verify only — accurate ("lazy errors" = deferred `initErr`). |
| sqlite paragraph | `README.md:87-90` | three lines on pure-Go, zero-value = memory, `WithPath`/`WithName` durable WAL, multi-process 5 s busy timeout | **Attach the batch clause here.** |
| Batch strategies (generic) | `README.md:72` and `:159` | both already mention "chunked IN queries + single-transaction upsert for sqlite" | Verify only — do not duplicate. |
| Benchmark results table | `README.md:161-172` | mem-only numbers from v1.6 | **Do not touch** (D-08: no sqlite numbers). |

Suggested clause to append to the 87-90 paragraph:

> `MGet` silently skips missing, expired, and undecodable keys, `MSet` writes the whole batch in one transaction under one TTL (all-or-nothing), and `MDel` deletes in idempotent chunks.

One sentence, mirrors `doc.go`, adds no numbers, no new section, no duplicated example — exactly D-11.

## Architecture Patterns

### Data flow of the evidence (not a file map)

```
push/PR
  └─> test job  (env: CGO_ENABLED=0)
        ├─ setup-go@v5 ── reads go.mod "go 1.26.4" ──> toolchain 1.26.4
        ├─ Build (CGO-free)   [shell: bash]
        │     ├─ go env CGO_ENABLED      -> prints 0 (visible proof)
        │     ├─ go build ./...          -> QUAL-01 compile half
        │     └─ [optional] go vet ./..., go list -m <pins>  -> pin visibility
        ├─ make swapper                  -> 4 cross-builds (unaffected; needs no cgo)
        ├─ go test -v ./...              -> QUAL-01 test half
        ├─ go test -tags=e2e …           -> QUAL-01 headline proof (two processes, no C toolchain)
        └─ SQLite race detector          (env: CGO_ENABLED=1, if: runner.os != 'Windows')
              └─ re-enables cgo for -race only -> CONC-03 stays proven on ubuntu+macos

separate coverage job (unchanged)
  └─ go test ./... -coverpkg=./... ──> go-test-coverage@v2 (config thresholds + total override)
```

### Recommended project structure (touched files only)

```
.github/workflows/go.yml        # job env + one new step + one step-level env
cache/bench_test.go             # + sqlite batch families, + primitive-ops benchmark
cache/sqlite/doc.go             # + one batch paragraph
cache/sqlite/example_test.go    # + ExampleNew_withPath
README.md                       # + one batch clause in the sqlite paragraph
.testcoverage.yml / -quick.yml  # UNCHANGED (verification only)
go.mod                          # UNCHANGED (verification only)
```

### Pattern 1: Job env for a cross-cutting build property, step env for the exception

**What:** Put the property that must hold for *all* steps at job level; override at the one step that structurally cannot honour it.
**When to use:** Exactly this phase's situation.
**Example:**

```yaml
jobs:
  test:
    name: Run tests
    permissions:
      contents: read
      pull-requests: read
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    env:
      CGO_ENABLED: "0"          # D-01: build, unit tests and two-process E2E all CGO-free
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build (CGO-free)            # D-03
        shell: bash                       # makes multi-command fail-fast on Windows too
        run: |
          go env CGO_ENABLED
          go build ./...
          go vet ./...
          go list -m modernc.org/sqlite modernc.org/libc

      - name: Build swapper binaries
        run: make swapper

      - name: Run tests
        run: go test -v ./...

      - name: SQLite two-process E2E
        run: go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s

      # -race requires a C toolchain, which Linux and macOS runners provide
      # (they satisfy CONC-03). Enabling -race on Windows (mingw setup) is a
      # reviewer-flagged open question for a future phase.
      - name: SQLite race detector
        if: runner.os != 'Windows'
        env:
          CGO_ENABLED: "1"                # D-02: -race needs cgo on every non-darwin OS
        run: go test -race ./cache/sqlite/ -count=1 -timeout=180s
```

The `go vet` and `go list -m` lines are discretion items: `go vet ./...` is clean under `CGO_ENABLED=0` (probe #5) and `go list -m` makes the QUAL-01 pins visible in the log. Neither is a gate; drop them if the plan prefers the minimal step.

### Pattern 2: Provider registration as an additive `b.Run` family

**What:** Add a provider to an existing benchmark function without touching existing blocks or changing helper signatures.
**When to use:** QUAL-04 / D-05 / D-06 / D-07.
**Example (validated — lint 0 issues, benchmarks run):**

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

Insert immediately **before** the `// ---- Docker-backed provider batch subtests (gated by skipIfNoDocker) ----` comment (`cache/bench_test.go:532`) and add `"path/filepath"` + `"github.com/guionardo/go/cache/sqlite"` to the import block. Resulting names: `BenchmarkBatch/sqlite/Memory/MGet/Size100/native` etc., directly comparable to `BenchmarkBatch/MGet/Size100/native` (mem).

### Pattern 3: External-test-package example with a real filesystem lifecycle

**What:** Durability example that closes before reopening and cleans up after itself without `*testing.T`.
**When to use:** D-10. See the validated code under Q5.

### Anti-Patterns to Avoid

- **Pinning `CGO_ENABLED` per step instead of at job level.** Eleven step-level `env:` blocks to express one job-wide property; drift guaranteed.
- **Duplicating the mem benchmark body for sqlite.** `runBatchBenchmarks` exists precisely for this; the mem block's duplication is legacy, not a template.
- **Adding a `cache/sqlite/bench_test.go`.** The helpers are unexported in `package cache_test` (`cache/`), so a second file would force exports or copy-paste (D-07).
- **Widening `make coverage` to `./cache/...`.** Looks like an improvement, breaks D-13's rationale; record why.
- **Registering sqlite in `cache/cache_e2e_test.go`'s `providerCase` list as part of this phase.** Tempting (PITFALLS Pitfall 16 recommends it) but it is a *new test* outside QUAL-02's five named classes; CONTEXT puts it out of scope. See Open Question 1.
- **Publishing benchmark numbers in README/SUMMARY.** D-08.
- **Adding a `-race` step for Windows.** D-04, deferred.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Proving cgo is off | A shell script that greps `go env` and exits non-zero | Job-level `env: CGO_ENABLED: "0"` + a step that *prints* `go env CGO_ENABLED` | The env var is the enforcement; a grep guard duplicates it and can drift (D-14 spirit). |
| Benchmark comparison harness | A bespoke table generator or a new runner | Existing `runBatchBenchmarks` + `go test -bench` + (if ever needed) `benchstat` | The v1.6 harness already produces native/per-key pairs at four sizes; anything else is a second source of truth. |
| Coverage merging / threshold checking | A custom threshold script or a `Makefile` reimplementation | `go-test-coverage` (local binary v2.18.9 / action v2) with the existing configs | Package/file/total thresholds, overrides and regex matching already implemented and already passing. |
| Temp dirs in examples | `os.TempDir()` + manual unique names, or `WithName` (pollutes user cache) | `os.MkdirTemp` + `defer os.RemoveAll` | Only correct route without `*testing.T`; matches the platform's temp semantics. |
| SQLite driver / pure-Go build | Any cgo driver or a build-tag shim | `modernc.org/sqlite` v1.60.1 (already pinned) | The whole milestone premise; no change needed. |
| Doc-comment linting | Manual review of `// Package` prefix | `godoclint` (already enabled) | Repo gate. |

**Key insight:** every problem in this phase already has a first-party or repo-native solution; the phase's risk is *not* in building anything, it is in recording evidence incorrectly (fail-open CI, stale numbers, duplicated harness).

## Test Scenario Coverage Map (QUAL-02 evidence, D-12)

Five named classes → existing tests. Every class already has coverage; **no test needs to be written** (D-12's "audit, not churn"):

| Named class | Test(s) | File:line | What it asserts |
|-------------|---------|-----------|-----------------|
| **unit** | `TestMemoryRoundTrip`, `TestFileCRUD`, `TestParity`, `TestMarshalFailure`, `TestClosedOps`, `TestClosedErrorPrefix`, `TestDoubleClose`, `TestOptimizable*`, `TestResolveTTL`, `TestPoolPinned`, `TestPragmaReadBack`, `TestBootstrapIdempotence`, `TestDeferredErr`, `TestCorruptedRowNotMiss`, `TestCanceledContext`, plus `batch_internal_test.go` (`TestUniqueKeys`, `TestChunksOf`, `TestInPlaceholders`, `TestMGetFunc`, `TestMSetFunc`, `TestMSetNoPartialWrites`, `TestMDelFunc`) and `dsn_internal_test.go`/`retry_internal_test.go`/`options_internal_test.go` | `sqlite_test.go`, `sqlite_internal_test.go`, `batch_internal_test.go`, `dsn_internal_test.go`, `retry_internal_test.go`, `options_internal_test.go` | The provider contract, DSN, schema and batch helpers, at function granularity |
| **`:memory:`** | `TestMemoryRoundTrip`, `TestMemoryIsolation`, `TestPoolPinned`, `TestTTLExpiry` | `sqlite_test.go:23`, `:454`; `sqlite_internal_test.go:231` | Round-trip and per-instance isolation in `:memory:` mode; single-connection pool pinning is what makes it correct |
| **reopen/persistence** | `TestReopenPersistence`, `TestExpiryAcrossRestart`, `TestFileCRUD`, `TestSidecarRemoval` | `sqlite_test.go:587-619`, `:621`, `:319`, `:564` | Set → Close → reopen → Get returns both no-TTL and TTL values; a further write after reopen proves WAL state is consistent; expired entries stay hidden across a restart; `-wal`/`-shm` disappear on last close |
| **concurrent** | `TestConcurrentBatchOpsRace` (8 goroutines × 7 ops × file+memory), `TestBeginContentionBounded`, plus the `e2e`-tagged `TestTwoProcessContention` / `TestTwoProcessCrashRecovery` | `sqlite_test.go:203-226`, `sqlite_internal_test.go:386`; `twoprocess_e2e_test.go:413`, `:475` | No data race on the pinned pool (`-race` CI step); contention surfaces at BEGIN and is bounded; cross-process correctness |
| **sweep** | `TestSweepOnOpen`, `TestSweepFailureSwallowed`, `TestPeriodicSweep`, `TestSweeperDisabled`, `TestReadsNeverDeleteRows` | `sqlite_internal_test.go:116-152`, `:156-166`, `:170-191`, `:195+`, `:98` | Open-time sweep is synchronous and removes only expired rows; failures are logged and swallowed; ticker reclaims post-open rows; unset interval creates no goroutine; reads never delete |

`TestConcurrentBatchOpsRace` carries a doc comment tying it to the concurrency evidence: *"TestConcurrentBatchOpsRace is the CONC-03 evidence: 8 goroutines mixing point and batch operations on one provider — file mode and :memory: — must complete without a data race on the pinned single-connection pool."* (`sqlite_test.go:200-202`)

Gate numbers to record alongside the mapping (from probe #11): `File coverage threshold (70%) satisfied: PASS`, `Package coverage threshold (80%) satisfied: PASS`, `Total coverage threshold (75%) satisfied: PASS`, `Total test coverage: 81.8% (2104/2572)`, `cache/sqlite` 92.8%, and `grep -n 'cache/sqlite' .testcoverage.yml .testcoverage-quick.yml` → **no matches**.

## Common Pitfalls

### Pitfall 1: `-race` under job-level `CGO_ENABLED=0` on the ubuntu leg
**What goes wrong:** The `SQLite race detector` step fails with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` (exit 2) the moment the job env lands — and only on Linux, so a macOS-only local check looks green.
**Why it happens:** `cmd/go` gates the race build on `CgoEnabled` for every `GOOS` except `darwin`; the driver being pure Go is irrelevant to the race runtime.
**How to avoid:** Step-level `env: CGO_ENABLED: "1"` on the race step (D-02). Verify by reproducing CI's error locally: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c ./cache/sqlite/` must print `go: -race requires cgo` before the fix and succeed after (`CGO_ENABLED=1` + a linux-capable cc, or just trust CI).
**Warning signs:** ubuntu leg red, macOS leg green, error mentioning cgo on a "pure Go" project.

### Pitfall 2: Multi-command `run:` step on Windows fails open (pwsh only checks the last exit code)
**What goes wrong:** `go env CGO_ENABLED` + `go build ./...` + `go vet ./...` in one `pwsh` block can report success while the build failed, because GitHub appends `exit $LASTEXITCODE` for the *last* command only.
**Why it happens:** GitHub's documented pwsh wrapper (see Q2) plus `$ErrorActionPreference = 'stop'`, which — critically — does not stop on **native** command non-zero exits.
**How to avoid:** `shell: bash` on the step (Git-for-Windows bash runs `-eo pipefail`), or one command per step. Never place a command *after* the one you actually need to gate unless the shell fails fast.
**Warning signs:** the build step is green on windows while `go test` (the next step) reports compile errors; or a step log showing a `# github.com/...` error banner followed by `##[section]Completed`.

### Pitfall 3: Windows temp-dir removal fails because the cache is still open
**What goes wrong:** `b.TempDir()`/`t.TempDir()` cleanup fails on windows-latest with a sharing violation if the SQLite handle (or a sidecar) is still open; Linux/macOS hide it.
**Why it happens:** Windows cannot unlink open files.
**How to avoid:** `b.Cleanup(func() { _ = c.Close() })` registered **after** `b.TempDir()` (LIFO ⇒ Close first). PITFALLS Pitfall 6 documents this exact pattern for tests; benchmarks inherit the same rule.
**Warning signs:** `TempDir RemoveAll cleanup: … used by another process` on the windows leg only.

### Pitfall 4: `contextcheck` fires when a context is hoisted into a benchmark helper
**What goes wrong:** Declaring `ctx := b.Context()` in a helper that then calls `prePopulateKeys(b, c, …)` (which does not take a ctx) triggers `Function 'prePopulateKeys' should pass the context parameter`.
**Why it happens:** `contextcheck` sees a ctx in scope that is not threaded through.
**How to avoid:** Call `b.Context()` inline at each cache call; never hold a ctx variable in a helper that also calls the ctx-free helpers. Verified: inline form = 0 issues.
**Warning signs:** two `contextcheck` findings appearing the moment the primitive benchmark is added.

### Pitfall 5: `wsl_v5` blank-line rule around `b.Cleanup`
**What goes wrong:** `b.Cleanup(func() { _ = c.Close() })` directly after a `:=` declaration is reported as `missing whitespace above this line (no shared variables above expr)`.
**How to avoid:** Blank line between the cache construction and the `b.Cleanup` registration (see the validated snippet). Cheap to get right up front; annoying to discover after a 10-repo-wide lint run.

### Pitfall 6: Example with `t.TempDir()` does not compile
**What goes wrong:** Writing `func ExampleNew_withPath(t *testing.T)` — example functions take no arguments; the file fails to build with a "wrong signature for Example" style error.
**How to avoid:** `os.MkdirTemp` + `defer os.RemoveAll` (validated, lint-clean). Do not substitute `sqlite.WithName` — that writes to the real user cache dir during `go test`.

### Pitfall 7: Benchmark key collisions / cross-family contamination
**What goes wrong:** Two sqlite families sharing a `name` (or a shared cache instance) means `MGet` keys collide and the per-key arm hits rows the native arm wrote, producing nonsense ratios.
**How to avoid:** One cache instance per family, distinct `name` strings (`sqlite_memory`, `sqlite_file`), and a fresh `b.TempDir()` path inside the `File` closure. The existing harness already embeds `name` in every key — as long as the caller supplies distinct names.

### Pitfall 8: Benchmark run time mistaken for a hang
**What goes wrong:** `make benchmark` (`-benchtime=1s -count=5`) gains ~5 minutes from 60 new sqlite sub-benchmarks, the slowest being file-mode `MDel`/`MSet` at size 1000 (~20 ms/op observed ⇒ ~50 iterations/s of benchtime).
**How to avoid:** Use `make benchmark-quick` (`-benchtime=100ms -count=1`, ~5 s) during development; document the added cost in the plan so a reviewer does not think the suite hung. Do not reduce `batchSizes` (D-05 requires 1/10/100/1000).

### Pitfall 9: Reconciling two coverage numbers that should not match
**What goes wrong:** The local quick gate (no `-coverpkg`) and the CI job (`-coverpkg=./...`) report 2104 vs 2105 statements at the same 81.8%. Someone "fixes" one to match.
**How to avoid:** Record which command produced which number; treat both as passing gates, not as one number with two spellings.

### Pitfall 10: Treating the old STACK.md race claim as licence to simplify
**What goes wrong:** `STACK.md` says `-race` "still works (pure Go)"; a later agent removes the step-level override "because we're pure Go now" and turns ubuntu red.
**How to avoid:** Write the supersession into the plan/SUMMARY in one sentence (see Q1.4): *the driver is pure Go; the race detector is a cgo build mode on every non-darwin OS.*

## Code Examples

### Primitive-ops benchmark (new function, validated: lint 0 issues, runs)

```go
// runPrimitiveBenchmarks measures the single-key Cache surface for one provider
// so mem and both sqlite families can be compared directly.
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

// BenchmarkPrimitives compares mem against both sqlite modes across the
// single-key Cache surface. mem is the baseline; no Docker gate.
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

`primitiveKeyRing` is a small private const (e.g. `const primitiveKeyRing = 1024`) declared per `decorder` order (`const` before `func`) to bound database growth during the timed loop. Observed magnitudes on an M4 (for sanity only — **never publish**, D-08): mem `Get` ≈ 37 ns/op vs `sqlite_memory` ≈ 4.9 µs/op, `sqlite_file` ≈ 6.7 µs/op.

### Verify a workflow change locally before pushing

```bash
# 1. Same toolchain as CI
export GOTOOLCHAIN=go1.26.4

# 2. What the new job env enforces
CGO_ENABLED=0 go env CGO_ENABLED      # -> 0
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go test ./... -count=1
CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -timeout=180s

# 3. The thing that must NOT be CGO-free (proves D-02 is needed)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c -o /dev/null ./cache/sqlite/   # exit 2

# 4. The gate the repo requires before every commit
make coverage-quick
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CGO sqlite drivers (`mattn/go-sqlite3`) needing gcc on all three OSes | `modernc.org/sqlite` (pure-Go transpiled SQLite) | pinned at v1.60.1, Phase 15 | CGO-free matrix possible; no mingw/TDM-GCC on Windows |
| Per-step `CGO_ENABLED` toggling | Job-level env + one step-level exception | this phase (D-01/D-02) | Single source of truth for the job's build property |
| `go test -bench` with hand-written provider blocks | Shared `runBatchBenchmarks`/`prePopulateKeys` helpers | v1.6 (Plan 08-02) | New providers are ~15 lines instead of ~80 |

**Deprecated/outdated in this repo's own docs:**
- `.planning/research/STACK.md:62` and `:152` — the "`-race` still works (pure Go)" claim. Correct for macOS, wrong for Linux; superseded by this phase's D-02. Do not delete the file (it is milestone research), but do not cite it for race-detector behaviour.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | ubuntu-latest and macos-latest runner images ship a C toolchain, so `CGO_ENABLED=1` in the race step is sufficient with no setup action | Q1/Q2, Pitfall 1 | The race step would need an apt/clang setup step. Mitigation: the existing step has been green on both legs through Phase 16 (STATE.md records PR #51 CI run 40340478439 with race green on ubuntu/macos), and `go.yml:34-36` asserts the toolchain is present. |
| A2 | Windows runners have no usable C compiler, so job-level `CGO_ENABLED=0` changes nothing there and `-race` stays impossible (D-04) | Q1/Q2 | Low: the existing `if: runner.os != 'Windows'` gate already encodes this. |
| A3 | `make benchmark` grows by roughly 5 minutes | Q4 / Pitfall 8 | A slower or faster machine shifts the estimate; no functional risk. Derived from measured ≈20 ms/op at the worst sub-benchmark × 48 batch + 12 primitive sub-benchmarks × 1 s × 5 counts. |
| A4 | Quoting `CGO_ENABLED: "0"` matters (GitHub stringifies unquoted numbers safely) | Q2/Pitfall 2 | Cosmetic; unquoted `0` would still work. |
| A5 | Windows *runtime* behaviour of the file-mode benchmarks (close-before-remove) is correct-by-construction via `b.Cleanup` LIFO, not runtime-verified here | Q4 / Pitfall 3 | If wrong, the windows leg fails on temp cleanup. Mitigation: mirror the pattern already proven by `TestConcurrentBatchOpsRace` and PITFALLS Pitfall 6, and watch the first CI run. |
| A6 | `actions/setup-go@v5` resolves `go-version-file: go.mod` to `1.26.4` from the `go` directive | Q2 | Low: documented input; the repo has no `toolchain` directive, so the v6+ resolution nuance does not apply. |

**Everything else in this document was executed locally or read from the Go 1.26.4 toolchain source / GitHub Actions documentation this session.**

## Open Questions (RESOLVED)

All five were answered by the Phase 17 plans; the resolution line under each question records what was planned, not a new decision.

1. **Should `cache/sqlite` be registered in `cache/cache_e2e_test.go`'s `providerCase` list?**
   - What we know: the shared provider-contract E2E covers mem/redis/valkey/memcache/postgres (`cache/cache_e2e_test.go:232`, `:243`, `:279`, `:322`, `:362`); sqlite is **absent**. PITFALLS Pitfall 16 (line 328-332) explicitly recommends adding a non-Docker sqlite case.
   - What's unclear: CONTEXT scopes QUAL-02 to five named classes and says "no new tests beyond filling a genuinely missing named class" — the shared contract is a sixth class with no requirement behind it.
   - Recommendation: **do not** add it in Phase 17; record it as a follow-up in the SUMMARY (one line). If the planner wants it, it is a genuinely cheap addition (no Docker needed), but it changes the phase from "audit" to "new test", which D-12 declines.
   - **Resolution (17-03, Task 3):** not added. The audit's `## Not added in this phase` section records the shared provider-contract registration as a follow-up with no requirement behind it (D-12/D-14); `cache/cache_e2e_test.go` is not in any plan's `files_modified`.

2. **Should `go vet` and the `go list -m` pin line join the `Build (CGO-free)` step?**
   - What we know: `CGO_ENABLED=0 go vet ./...` is clean; `go list -m modernc.org/sqlite modernc.org/libc` prints `v1.60.1` / `v1.77.1`.
   - What's unclear: D-03 specifies only `go env CGO_ENABLED` + `go build ./...`; both additions are discretion.
   - Recommendation: include both — they cost ~2 s, make QUAL-01's pin half visible in the log, and add a second (free) vet gate. Requirement: `shell: bash` (Pitfall 2), otherwise the appended commands mask a build failure.
   - **Resolution (17-01, Task 1):** both included. The step runs `go env CGO_ENABLED` -> `go build ./...` -> `go vet ./...` -> `go list -m modernc.org/sqlite modernc.org/libc`, nothing after them, under `shell: bash` — exercised by the CONTEXT discretion item "whether `go vet` joins the build step".

3. **`GetOrSet` primitive arm: warm fast-path or miss→setter?**
   - What we know: both are legitimate; warm-hit is the steady-state cost and is mem-comparable; miss measures the write path.
   - Recommendation: warm-hit on a bounded key ring; add a separately named `GetOrSet_Miss` arm only if someone wants the other question answered.
   - **Resolution (17-02, Task 2):** warm fast-path only, on the bounded `primitiveKeyRing = 1024` ring; no `GetOrSet_Miss` arm is planned.

4. **Should the doc.go batch paragraph mention the 100-key chunk size?**
   - What we know: `const chunkSize = 100` is an internal constant (`batch.go:19`), deliberately not an option surface (D-01 in Phase 16).
   - Recommendation: mention "chunked" without the number; the number is an implementation detail that would then be a doc-maintenance liability. The README clause likewise says "in chunks".
   - **Resolution (17-03, Task 1):** no chunk-size number in `doc.go` or `README.md`; both say "chunked"/"in chunks" only, and the plan carries it as a prohibition.

5. **The `coverage` job uses `actions/setup-go@v3` with no `go-version-file`.**
   - What we know: it therefore uses whatever Go the ubuntu-latest image provides, which may drift past `go 1.26.4`.
   - Recommendation: out of scope (untouched by QUAL-01..04); flag as a follow-up observation only. Do not "fix" it in this phase — it would add an unrelated diff to a delivery-hardening PR.
   - **Resolution (17-01, `<verification>`):** untouched. The plan's verification section requires recording it in the SUMMARY as an out-of-scope follow-up observation, with an explicit "do NOT fix it here" instruction.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain 1.26.4 | everything | ✓ | `go1.26.4` (via `GOTOOLCHAIN=auto`; host install is 1.27.0) | `GOTOOLCHAIN=go1.26.4` pins CI's exact toolchain (auto-downloaded once) |
| `make` | `make coverage-quick`, `make swapper`, `make benchmark*` | ✓ | `/usr/bin/make` | run the underlying `go` commands directly |
| `golangci-lint` | pre-commit lint gate | ✓ | 2.14.0 | — |
| `go-test-coverage` | `make coverage-quick` | ✓ | v2.18.9 | CI action runs the same checks |
| `gocovmerge` | `make coverage` (E2E merge) | ✓ | present | not needed for QUAL-02 |
| Docker | `make coverage` (E2E merge), `make test-e2e` | ✗ | — | not needed: `make coverage-quick` is the repo's required pre-commit gate; the sqlite E2E runs with `-tags=e2e` and no Docker |
| C compiler | `-race` (cgo), local repro only | ✓ | Apple clang 21.0.0 | CI provides gcc/clang on ubuntu/macos |
| Windows runner | QUAL-01's third OS | ✗ locally | — | cross-compile proxies (`GOOS=windows go build`, `go test -c`) + the CI windows leg |

**Missing dependencies with no fallback:** none that block execution.
**Missing dependencies with fallback:** Docker (covered above), Windows runtime (covered by CI).

## Security Domain

`security_enforcement: true`, `security_asvs_level: 1` (`[VERIFIED: .planning/config.json:47-48]`). This phase changes a CI workflow, a benchmark file, a package doc comment, an example and a README paragraph — no runtime code, no new input surface, no new dependency.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface touched. |
| V3 Session Management | No | No session surface. |
| V4 Access Control | No | No authorization decisions. |
| V5 Input Validation | No (unchanged) | The phase adds no parsing/DSN/query surface. Existing DSN hardening (`dsn.go`, `TestInvalidPathAndName`) is untouched. |
| V6 Cryptography | No | Nothing encrypted; `doc.go:54-55` already states *"Cache contents are unencrypted at rest"* — the new batch paragraph must not imply otherwise. |
| Supply chain (Actions) | **Yes, lightly** | The change adds **no new action**: it reuses `actions/checkout@v4`, `actions/setup-go@v5` and the existing `vladopajic/go-test-coverage@v2`. Job `permissions` stay `contents: read` / `pull-requests: read` for `test`. Do not add `permissions: write` or a new third-party action. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| CI step that fails open (claims a CGO-free build it did not perform) → false assurance about the shipped artifact | Repudiation / Tampering | `shell: bash` (or one command per step) so any non-zero exit fails the step; keep `go env CGO_ENABLED` output in the log as the visible proof |
| Supply-chain drift via unpinned or newly added Actions | Tampering | Add no new action; keep existing refs unchanged |
| Secrets leakage into logs | Information disclosure | The new step prints only `CGO_ENABLED` and module versions — no secrets, no `GITHUB_TOKEN` usage |
| Benchmark writing outside the workspace | Tampering (local) | `b.TempDir()` only; never a fixed path, never the user cache dir |
| Doc drift implying encryption/protection that does not exist | — | Batch paragraph says nothing about confidentiality; `doc.go:54-55` keeps its explicit "unencrypted at rest" statement |
| Path/handle leak on Windows from benchmark caches | Denial of service (CI flake) | `b.Cleanup` close-before-remove (Pitfall 3) |

## Sources

### Primary (HIGH confidence — read or executed this session)
- `$GOROOT/src/cmd/go/internal/work/init.go:193-204` in **go1.26.4** (`/Users/guionardo/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.4.darwin-arm64`) and **go1.27.0** (`/opt/homebrew/Cellar/go/1.27.0/libexec`) — the `-race requires cgo` gate and its darwin exception, quoted verbatim.
- Local executions (probe table above): `CGO_ENABLED=0` build/vet/test/E2E/swapper/race, cross-compiles for linux/amd64, linux/arm64, windows/amd64, `make coverage-quick`, CI coverage-profile repro, validated example + benchmark probes, `golangci-lint`.
- Repo sources read in full or in cited ranges: `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/sqlite/{doc.go,batch.go,example_test.go,options.go,sqlite.go,twoprocess_e2e_test.go,sqlite_test.go,sqlite_internal_test.go}`, `README.md`, `Makefile`, `.testcoverage.yml`, `.testcoverage-quick.yml`, `.golangci.yml`, `go.mod`, `.planning/config.json`.
- Planning artifacts: `17-CONTEXT.md`, `REQUIREMENTS.md`, `STATE.md`, `.planning/research/STACK.md`, `.planning/research/PITFALLS.md`, `.planning/codebase/TESTING.md`.
- `go list -m modernc.org/sqlite modernc.org/libc` → `v1.60.1` / `v1.77.1`.

### Secondary (MEDIUM confidence — official docs, fetched this session)
- GitHub Actions workflow syntax — env precedence (*"An environment variable defined in a step will override job and workflow environment variables with the same name"*), supported-shells table (`bash --noprofile --norc -eo pipefail {0}`, Git-for-Windows bash on Windows), and exit-code semantics (*"we append `if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }` … so action statuses reflect the script's last exit code"*). `docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax`
- `actions/setup-go` README — `go-version-file` input semantics and precedence over `go-version`. `github.com/actions/setup-go`
- `vladopajic/go-test-coverage` `action.yml` — `threshold-file` / `threshold-package` / `threshold-total` are independent inputs, each "Overrides value from configuration", default `-1`.

### Tertiary (LOW confidence — not verified this session)
- Windows runner runtime behaviour of the file-mode benchmarks and the two-process E2E under `CGO_ENABLED=0` (proxied by cross-compilation and by Phase 16's green windows CI leg, not executed here).

## Metadata

**Confidence breakdown:**
- CGO/race interaction: **HIGH** — toolchain source quoted for both installed Go versions + local reproduction of the failure path and the macOS success path.
- CI mechanics: **HIGH** for env precedence and shell/exit-code behaviour (official docs quoted); **MEDIUM** for runner C-toolchain availability (A1, backed by Phase 16 CI evidence).
- Coverage tooling: **HIGH** — both configs read in full, action `action.yml` read, both profiles executed with PASS.
- Benchmark harness: **HIGH** — signatures quoted with line ranges; a working registration + primitive benchmark was compiled, linted and run, then reverted.
- Docs targets: **HIGH** — every claim traced to `doc.go`/`batch.go`/`README.md` line ranges; the replacement example was executed and linted.
- Pitfalls: **HIGH** for the two lint pitfalls and the pwsh fail-open behaviour (all reproduced or documented); **MEDIUM** for the Windows handle pitfall (documented pattern, not run on Windows).

**Research date:** 2026-10-26
**Valid until:** 2026-11-25 (30 days) — re-check only if `go.mod`'s Go version, the `modernc.org/sqlite` pin, or the workflow's action versions change. The `-race`-requires-cgo rule has been stable across 1.26 and 1.27 and is unlikely to move, but re-read `cmd/go/internal/work/init.go` if a toolchain upgrade is proposed.
