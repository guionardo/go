# Milestones

## v1.7 Project Probe (Shipped: 2026-09-30)

**Phases completed:** 5 phases, 16 plans, 36 tasks

**Key accomplishments:**
- Deleted the untracked `project_detector/` sample (12 files) via filesystem `rm -rf`, restoring a green `go build ./...` across the module and satisfying FND-01 as the phase's wave-1 precondition.
- The `projectprobe` package's never-fail API contract shipped end-to-end under TDD: `Probe(folder) (ProjectData, error)` wired through clean-once path handling, syscall errno → sentinel mapping (incl. EPERM), the ordered panic-safe detector registry, and the 13-entry ignore-list content gate — with the empty registry delivering `LanguageUnknown` + nil error for every content-bearing folder, ready for detector bodies in phases 11-13.
- The shared safe-manifest helper `readManifest(folder, name) ([]byte, bool)` shipped under TDD — a 1 MB size cap enforced by the `io.LimitReader(maxManifestSize+1)` truncation probe (exactly 1 MB accepted, over-cap a non-match) and a UTF-8 BOM strip via `bytes.TrimPrefix`, with every failure mode degrading to `ok=false` so the package's never-fail contract (D-03) holds and phases 11-13 detectors get their file reader without a signature change.
- README description fallback (README.md → README.rst → README exact-case chain with first-real-paragraph extraction skipping badges/TOC/headings/HTML comments) plus the WR-01 regular-file gate closing the FIFO hang DoS on readManifest
- Go detector (detectGo + parseGoMod) wired into the production registry at index 0 with a 7-position nil-slot literal (D-11), nil-skip dispatch, refreshed doc.go, and position-pinning test — a probe of any folder with a go.mod now returns Language=Go with module-name, raw toolchain-floor version, and README-derived description
- package.json → LanguageJavaScript and composer.json → LanguagePHP detectors end-to-end through Probe, backed by a shared never-fail readJSONManifest helper (readManifest + encoding/json), the complete 7-slot D-11 registry literal (Go@0, JS@3, PHP@6), and an integration test pinning Go > JS > PHP cascade precedence with broken-manifest fall-through
- Unexported section-aware TOML-subset reader `readTOMLSection` with exact-header matching, strict degrade-to-empty, and three global skip states — the fixture-validated SC4 contract that both Python/Rust detectors (plans 12-02/12-03) read through
- detectPython wired into the production registry at index 1 — pyproject.toml presence-match with PEP 621 `[project]` fields, whole-section `[tool.poetry]` legacy fallback (D-disc-3), DATA-02 folder-base + DATA-04 README chains, and the interim TestDetectorPositions pin (Python live, Rust still nil)
- detectRust wired into the production registry at index 4 completing the phase's 5-of-7 live slots — Cargo.toml presence-match with `[package]` fields (D-06), workspace-inherited `version.workspace = true` degrading to empty Version with zero resolution code (D-07, SC3), the final TestDetectorPositions pin (NotNil 0/1/3/4/6, Nil 2/5), doc.go refresh, and two cascade rows proving Python@1 beats Rust@4 and beats broken-JS@3
- Quote-aware skip-state clearing in readTOMLSection (closesMultiLine/clearsBracket/opensMultiLine + cross-line pendingMLS), with a 13-subtest matrix pinning no-fabrication for every CR-01 shape — restores SC4, 12-01 truth 4, and 12-02 truth 5.
- C#/.NET detector end-to-end: readFirstManifest .csproj discovery + inline namespace-agnostic xml.Unmarshal decode (XMLName local-name), D-03 chains, registry slot 2 filled with interim TestDetectorPositions flip — 15 probe-level test rows pinning the DETC-07 contract
- Java/Kotlin detector end-to-end: pom.xml presence-match with single-level `<parent><version>` inheritance (D-05), never-fail `parseRootProjectName` settings.gradle fallback only-when-no-pom (D-06), registry slot 5 filled — the 7-slot literal is COMPLETE with the final all-7-NotNil position test, doc.go all-seven refresh, and 5 cascade precedence rows (C#@2 > Java@5 > PHP@6)
- Three fuzz targets (FuzzJSONManifest/FuzzTOMLManifest/FuzzXMLManifest) exercising the six manifest-detector parse paths (readManifest → decode → chains) with a 20-file `go test fuzz v1`-encoded seed corpus — the SC3 no-panic invariant proven by seed-mode execution under plain `go test ./project_probe/...`
- readme.go plain-badge and multi-line HTML comment extraction fixes, decode-hygiene XML element trimming with exact-name guard in both XML detectors, and explicit empty-slice registry test hygiene — all landed RED-first with review-prescribed shapes (D-06/D-07, DATA-03)
- No-options + seven error-path release tests close the pre-existing coverage-gate known-red (D-09): release/update.go 68.9% → 95.9%, `make coverage-quick` green repo-wide, all test-only (Pitfall 4)
- Version-semantics contract in doc.go (all 7 detectors' raw-string rules), README project_probe package index row + section, recorded anti-feature audit with deferred-disposition ledger, and the final green gates closing milestone v1.7

---

## v1.6 Cache Dedup (Shipped: 2026-08-08)

**Phases completed:** 5 phases, 11 plans, 27 tasks

**Key accomplishments:**

- Exported cache.SingleflightGetOrSet[K,V] blocking dedup helper with panic recovery (cache.Panic) and a double-check Get clobber guard, validated by a 6-behavior TDD suite and a godoc example.
- DoChan cancel-aware variant of SingleflightGetOrSet[K,V] with the cache.ErrCanceled sentinel: canceled waiters abandon at cancel latency (SF-06) returning an error matching both ErrCanceled and context.Canceled, value-if-ready wins (D-08), and the leader path never wraps its own cancellation (D-11)
- The `Cache` interface `GetOrSet` setter now takes `context.Context` — the breaking D-03 change rippled through all 5 provider methods (mem, redis, valkey, memcache, postgres) and every test/E2E call site, with error decoration, valkey's initErr guard, and `t.Parallel()` placement untouched; provider bodies stay hand-rolled (delegation to `SingleflightGetOrSet` is Phase 6).
- Added `BatchCache[K,V]` embedding interface, expanded `cacher` with 3 batch methods, added `concreteCache` delegation, changed `NewConcreteCache` return type, provided provider stubs, and added complete test coverage with 14 batch test subtests.
- mem single-lock batch (D-07), memcache GetMulti + goroutine-per-key batch (D-08), New() return type changed to BatchCache for both providers
- Implemented Pipeline-based batch operations for the redis provider (go-redis Pipeline per D-09) and DoMulti-based batch operations for the valkey provider (valkey-go DoMulti per D-10); changed both `New()` return types to `cache.BatchCache[K,V]`; added provider-level integration tests with skipIfNo* guards.
- Postgres SendBatch batch operations, postgres.New returning BatchCache, provider-level integration tests, E2E infrastructure updated for BatchCache, batch E2E subtests, and race detector verification across all 5 providers
- Thundering-herd singleflight benchmark in cache/bench_test.go quantifies the dedup win: naive path runs the setter N× per iteration, singleflight runs it exactly 1×, across concurrency levels 10–500, with mem (always) and 4 Docker-gated providers
- Batch benchmarks in cache/bench_test.go quantify the MGet/MSet/MDel win factor: native MGet 10-17% faster, native MDel up to 49% faster, and native MSet consistently lower allocations across batch sizes 1-1000 using the mem provider
- `make benchmark` and `make benchmark-quick` targets added to Makefile alongside existing test/coverage targets, with Docker-gated batch benchmark subtests for redis, valkey, memcache, and postgres providers using the shared runBatchBenchmarks helper

---

## v1.5 Self-Update (Shipped: 2026-07-21)

**Phases completed:** 1 phases, 3 plans, 0 tasks

**Key accomplishments:**

- (none recorded)

---

## v1.4 Core Packages (Shipped: 2026-07-21)

**Phases completed:** 1 phase, 3 plans, 50 tasks

**Timeline:** 2026-07-21 (4 hours)
**Commits:** 24
**Files changed:** 50 files, +6,625 / -51

**Key accomplishments:**

1. Generic `Cache[K, V]` interface with Get, Set, Delete, GetOrSet, Close across 5 backends
2. In-memory provider with TTL sweep, sync.RWMutex concurrency safety
3. Redis + Valkey providers using go-redis/v9 and valkey-go
4. Memcache provider with goroutine-per-call context wrapping
5. Postgres provider with UNLOGGED table, pg_prewarm, background TTL sweep
6. 50 E2E tests across all providers using testcontainers-go
7. Build-tag separation: e2e tests require Docker, regular tests pass without

**Verification:** passed
**UAT:** 8/8 tests passed

**Archived artifacts:**

- `milestones/v1.4-ROADMAP.md`
- `milestones/v1.4-REQUIREMENTS.md`
- `milestones/v1.4-phases/`

---

## v1.5 (Planned)

### Planned phases:

- STRNG: String utilities package (truncation, padding, join/split)
- RETRY: Retry with backoff strategies and jitter support
