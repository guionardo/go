# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.4 — Core Packages

**Shipped:** 2026-07-21
**Phases:** 1 | **Plans:** 3 | **Commits:** 24

### What Was Built
- Generic `Cache[K, V]` interface with 5 backends (in-memory, Redis, Valkey, Memcache, Postgres)
- In-memory provider with TTL sweep and concurrent-safe access
- Redis + Valkey providers with JSON serialization and sub-second TTL support
- Memcache provider with goroutine-per-call context wrapping
- Postgres provider with UNLOGGED table, pg_prewarm, background TTL sweep
- 50 E2E tests using testcontainers-go across all 5 providers
- Build-tag separation (e2e) for Docker-dependent tests

### What Worked
- Generic `Cache[K, V]` interface made testing easy — swap providers with one line
- Functional options pattern consistent across all providers
- Design-first (interface) → provider implementation order was effective
- Build tags kept `go test ./...` working without Docker
- UAT caught real issues (Valkey readiness check race, missing build tags)

### What Was Inefficient
- `ExampleNew` tests required real servers — needed build tag retrofitting
- Valkey container readiness check was unreliable — required fix during UAT
- No STATE.md meant gsd-tools couldn't track progress automatically

### Patterns Established
- Each provider in own sub-package, importable independently
- E2E tests in testcontainers with `//go:build e2e` tag
- VERIFICATION.md + UAT.md as dual verification gates
- Per-provider `Options` type with functional options

### Key Lessons
1. Example tests (`ExampleXxx`) should check env vars or use build tags — they can't skip like regular tests
2. Container readiness checks need `wait.ForListeningPort` alongside log matching for reliability
3. Build tags are essential for separating unit/E2E tests in a package

### Cost Observations
- Model mix: 100% adaptive (no explicit model selection)
- Sessions: 1 session (4 hours)
- Notable: Cache package from zero to shipped in a single session

## Milestone: v1.5 — Self-Update

**Shipped:** 2026-07-21
**Phases:** 1 | **Plans:** 3 | **Commits:** 16

### What Was Built
- `release` package with version detection (hashicorp/go-version), GitHub release checking, platform-specific asset download with SHA256 verification
- Cross-platform swapper binary (Linux, macOS amd64/arm64, Windows amd64) with atomic backup-rename-replace and rollback
- Self-update orchestrator (`PerformSelfUpdate`) with file-lock concurrency protection
- Embedded swapper via `//go:embed` for all 4 target platforms
- Example CLI (`cmd/example-updater`) demonstrating the update flow
- Comprehensive `doc.go` for all 21 library packages + restructured main README with package index

### What Worked
- `--target` flag fix on swapper resolved the self-replacement bug cleanly
- Two-phase SHA256 verification (go-digest + stdlib) provides defense-in-depth
- Functional options pattern (`WithOwner`, `WithRepo`, `WithGitHubToken`) consistent with existing cache package
- `//go:embed` made swapper distribution trivial — no installer needed
- Lock file prevents concurrent updates without external dependencies

### What Was Inefficient
- Swapper binary must be pre-built for all platforms before embedding — requires `make swapper` as a build step
- VERIFICATION.md YAML frontmatter format was initially wrong, causing tool to report "missing" status

### Patterns Established
- Self-update as an embedded binary pattern (spawn → exit → swap → exec)
- Two-phase verification for sensitive operations (download + swap)
- File-based lock for cross-process synchronization

### Key Lessons
1. GSD verification queries expect YAML frontmatter (`status: passed`) — markdown formatting (`**Status:**`) is not parsed
2. Cross-platform builds for embedded binaries need careful Makefile orchestration
3. `hashicorp/go-version` handles edge cases (prereleases, pseudo-versions) that a custom parser would miss

### Cost Observations
- Model mix: 100% adaptive (no explicit model selection)
- Sessions: 2 sessions (Phase 4 execution + documentation pass)
- Notable: Self-update with swapper from zero to shipped in 2 sessions

---

## Milestone: v1.7 — Project Probe

**Shipped:** 2026-09-30
**Phases:** 5 | **Plans:** 16 | **Commits:** 118

### What Was Built
- Never-fail `project_probe` package: `Probe(folder)` detecting all 7 languages (Go, Python, C#/.NET, JS/TS, Rust, Java/Kotlin, PHP) with name/version/description chains and README first-paragraph fallback
- Strict-degrade TOML-subset reader with quote-aware skip states (SC4 fabrication gap found by code review, closed via gap plan)
- Fuzz-hardened: 3 targets + 20-file `go test fuzz v1`-encoded corpus, no-panic invariant
- Coverage gate closed repo-wide: release/update.go 68.9% → 95.9%, `make coverage-quick` green (80.7%)
- doc.go Version-semantics contract, anti-feature audit (14-AUDIT.md), README package index
- 820+ tests across 25 packages; UAT 12/12; security 16/16 threats closed

### What Worked
- Tracer-first planning: each phase led with an end-to-end slice verified before expansion
- TDD adaptation (RED verified-but-uncommitted, evidence in commit bodies) kept CI green under the pre-commit go-test hook
- Gap-closure cycle: verifier found the fabrication gap → /gsd-plan-phase --gaps → 12-04 fix → re-verify — the loop worked end-to-end
- Code review caught real issues (quote-aware skip states, multi-badge lines, goroutine asserts, XML trim) — all fold-in fixed in Phase 14
- Deterministic anti-feature greps (grep -E) prevented vacuous checks (two checker warnings on BRE `|`)

### What Was Inefficient
- Code review finding CR-01 (TOML fabrication) required a full gap-closure cycle — the initial whole-line skip-state design should have been quote-aware from the start
- Verification digests went stale 4× (phases 10-13) as later phases touched shared files — required re-verification before milestone close
- `_js` filename silently excluded from non-js builds (legacy GOARCH) — cost a rename mid-phase
- Lint baseline of 28 pre-existing issues made "clean" need a defined delta-zero contract

### Patterns Established
- Quote-aware parsing for untrusted input: skip states must handle delimiters inside quotes (closesMultiLine/clearsBracket/opensMultiLine + pendingMLS)
- ENOTDIR triggers instead of chmod for cross-platform error-path tests (Windows CI)
- Fuzz corpus must be `go test fuzz v1`-encoded (raw manifests fail the build)
- Anti-feature audits as re-runnable evidence files with disclosed exceptions (14-AUDIT.md)
- Decode hygiene ≠ normalization: TrimSpace at decode time documented explicitly

### Key Lessons
1. Whole-line delimiter detection in parsers fabricates data — always quote-aware from the start
2. Covered-file digests go stale when later phases touch shared packages — re-verify before milestone close
3. A deterministic gate that cannot fail (vacuous grep) is worse than no gate — verify greps match on a sample
4. Test-only fixes can close long-standing coverage debt (one no-options row covered 13 statements)

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Commits | Phases | Key Change |
|-----------|---------|--------|------------|
| v1.0 | 24 | 1 | Initial GSD workflow setup with plan→execute→verify→UAT cycle |
| v1.5 | 16 | 1 | Self-update mechanism with embedded swapper; doc.go for all packages |
| v1.7 | 118 | 5 | project_probe 7-language detection; TDD adaptation; gap-closure cycle; fuzz hardening |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.0 | 50 E2E + unit tests | 95%+ target | 4 (gomemcache, pgx, go-redis, valkey-go) |
| v1.5 | 57 unit tests | 95%+ target | 1 (hashicorp/go-version) |
| v1.7 | 820+ tests / 25 packages | 80.7% total (coverage-quick green) | 0 (stdlib-only) |
