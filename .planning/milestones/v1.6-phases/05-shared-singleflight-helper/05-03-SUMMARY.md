---
phase: 05-shared-singleflight-helper
plan: 03
subsystem: cache
tags: [go, cache, context, interface-contract, breaking-change]

# Dependency graph
requires: []
provides:
  - "Cache interface GetOrSet setter is now func(context.Context) (V, error) (D-03)"
  - "All 5 provider GetOrSet methods pass the caller's ctx to the setter with zero error-semantics change (D-06/D-14)"
  - "Provider test suites and E2E suite compile and pass against the new signature"
affects: [06-shared-singleflight-helper-delegation, 05-01, 05-02]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ctx-aware setter closures: func(context.Context) (V, error) at every GetOrSet call site"
    - "compile-time interface assertions (var _ cache.Cache[...]) as the D-03 contract enforcement point"

key-files:
  created: []
  modified:
    - cache/cache.go
    - cache/doc.go
    - cache/cache_e2e_test.go
    - cache/mem/mem.go
    - cache/redis/redis.go
    - cache/valkey/valkey.go
    - cache/memcache/memcache.go
    - cache/postgres/postgres.go
    - cache/mem/mem_test.go
    - cache/redis/redis_test.go
    - cache/valkey/valkey_test.go
    - cache/memcache/memcache_test.go
    - cache/postgres/postgres_test.go

key-decisions:
  - "D-03 applied: GetOrSet setter signature breaks to func(context.Context) (V, error) — accepted public break in v1.6"
  - "D-06 applied: providers pass the caller's ctx verbatim; no separate setterCtx"
  - "D-14 applied: error decoration untouched — redis/valkey prefixes, valkey initErr guard, mem/memcache/postgres raw errors"
  - "Provider bodies stay hand-rolled; delegation to SingleflightGetOrSet is Phase 6 (SF-03)"

patterns-established:
  - "Setter closure shape: func(context.Context) (V, error) is now the single GetOrSet contract across interface, providers, and tests"
  - "E2E closures use func(_ context.Context) to signal the ctx is intentionally ignored"

requirements-completed: [SF-02]

coverage:
  - id: D1
    description: "Cache interface GetOrSet accepts ctx-aware setter; all 5 providers satisfy the interface via compile-time assertions with error semantics preserved"
    requirement: SF-02
    verification:
      - kind: unit
        ref: "go build ./cache/... (compile-time assertions mem.go:130, memcache.go:178, postgres.go:177)"
        status: pass
      - kind: unit
        ref: "go test ./cache/mem/ ./cache/redis/ ./cache/valkey/ ./cache/memcache/ ./cache/postgres/ -count=1 — 71 passed"
        status: pass
    human_judgment: false
  - id: D2
    description: "Provider test closures and E2E suite compile and pass against the ctx-aware setter signature"
    requirement: SF-02
    verification:
      - kind: unit
        ref: "go test ./cache/mem/ ./cache/redis/ ./cache/valkey/ ./cache/memcache/ ./cache/postgres/ -count=1"
        status: pass
      - kind: integration
        ref: "go vet -tags e2e ./cache/ (compiles e2e-tagged GetOrSet closures)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Package docs document ErrCanceled, the ctx-aware GetOrSet setter, and the SingleflightGetOrSet helper"
    verification:
      - kind: other
        ref: "grep cache/doc.go for ErrCanceled, context.Context, SingleflightGetOrSet"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-08-06
status: complete
---

# Phase 05 Plan 03: D-03 ctx-aware setter ripple across Cache interface, 5 providers, and all GetOrSet call sites

**The `Cache` interface `GetOrSet` setter now takes `context.Context` — the breaking D-03 change rippled through all 5 provider methods (mem, redis, valkey, memcache, postgres) and every test/E2E call site, with error decoration, valkey's initErr guard, and `t.Parallel()` placement untouched; provider bodies stay hand-rolled (delegation to `SingleflightGetOrSet` is Phase 6).**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-06T14:15:38Z
- **Completed:** 2026-08-06T14:21:16Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments
- `cache.Cache.GetOrSet` setter signature changed to `func(context.Context) (V, error)` (D-03) with an updated doc comment; all 5 providers still satisfy the interface (compile-time assertions green)
- Every provider passes the caller's ctx to the setter exactly once (D-06); redis keeps `cache/redis: %w`, valkey keeps its initErr guard and `cache/valkey: %w`, mem/memcache/postgres keep raw setter errors (D-14)
- 5 provider test files adapted (7 closures total) plus the E2E suite's 3 closures — full module green: build, race test, vet, and `make coverage-quick` (total 78.4%, all thresholds PASS)
- `cache/doc.go` documents `ErrCanceled`, the ctx-aware GetOrSet contract, and the `SingleflightGetOrSet` helper

## Task Commits

Each task was committed atomically:

1. **Task 1: Change the Cache interface setter signature and ripple all 5 provider GetOrSet bodies** - `6ade60f` (feat)
2. **Task 2: Ripple the ctx setter through provider test closures and the E2E suite** - `2ac849c` (test)
3. **Task 3: Update package docs and run the full module gate** - `3c76ed6` (docs)

## Files Created/Modified
- `cache/cache.go` - GetOrSet interface setter gains `context.Context` (D-03); doc comment updated
- `cache/mem/mem.go` - GetOrSet signature + `setter(ctx)` call; raw setter error preserved
- `cache/redis/redis.go` - GetOrSet signature + `setter(ctx)`; `cache/redis: %w` prefix preserved
- `cache/valkey/valkey.go` - GetOrSet signature + `setter(ctx)` after initErr guard; `cache/valkey:` prefix preserved
- `cache/memcache/memcache.go` - GetOrSet signature + `setter(ctx)`; raw setter error preserved
- `cache/postgres/postgres.go` - GetOrSet signature + `setter(ctx)`; raw setter error preserved
- `cache/mem/mem_test.go` - 3 setter closures gain ctx param; `context` import added
- `cache/redis/redis_test.go` - 1 setter closure gains ctx param; `context` import added
- `cache/valkey/valkey_test.go` - 1 setter closure gains ctx param; `context` import added
- `cache/memcache/memcache_test.go` - 1 setter closure gains ctx param; `context` import added (t.Parallel NOT added — server-backed subtest)
- `cache/postgres/postgres_test.go` - 1 setter closure gains ctx param; `context` import added (t.Parallel NOT added — server-backed subtest)
- `cache/cache_e2e_test.go` - 3 GetOrSet closures use `func(_ context.Context)`; `context` import added
- `cache/doc.go` - ErrCanceled sentinel doc line, ctx-aware GetOrSet description, SingleflightGetOrSet mention

## Decisions Made
- Applied locked decisions D-03, D-06, D-14 exactly as specified — no new decisions required
- Provider `context` imports were added in each test file (required for the new closure signature to compile; implied by the plan's closure-ripple task)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-staged unrelated file swept into Task 1 commit**
- **Found during:** Task 1 (commit step)
- **Issue:** `.github/workflows/opencode.yml` was staged in the index before execution started (unrelated to this plan). The first Task 1 commit (`96399da`) captured it alongside the 6 task files (7 files total).
- **Fix:** Soft-reset the commit (`git reset --soft HEAD~1`), unstaged the unrelated file (`git restore --staged`), and recommitted only the 6 task files. The workflow file's content is preserved on disk, untracked, for the orchestrator/user to handle separately.
- **Files modified:** (commit scope fix only — no source change)
- **Verification:** `git show --stat HEAD` shows exactly the 6 planned files; `git status` shows the workflow file back to untracked.
- **Committed in:** `6ade60f` (Task 1 commit, corrected)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Single auto-fix kept the task commit scoped to plan files. No scope creep; no plan behavior change.

## Issues Encountered
- **Transient e2e vet failure (plan-anticipated):** `go vet -tags e2e ./cache/` failed once with `undefined: cache.SingleflightGetOrSet` because plan 05-01's `singleflight_test.go` had landed on disk before its `singleflight.go` implementation. Resolved by re-running after 05-01's implementation landed — vet then passed with no issues. Exactly the transient cross-plan condition documented in the plan's context note.
- **Task 2 verify note:** `go vet -tags e2e` compiled the whole cache test package, so it could not typecheck the E2E closure edits in isolation while 05-01 was mid-flight; the re-run after 05-01 landed confirmed the closures compile.

## User Setup Required

None - no external service configuration required. (Redis/valkey/memcache/postgres tests skip gracefully when no server is present; memcache/postgres server-backed subtests passed by skip.)

## Next Phase Readiness
- The `Cache` contract and all providers/tests are on the ctx-aware setter signature — Phase 6 can delegate the 5 hand-rolled GetOrSet bodies to `cache.SingleflightGetOrSet` with no further signature work
- `cache/singleflight.go` (plan 05-01) and `cache/singleflight_test.go` now exist on disk; `cache/errors.go` (`ErrCanceled`/`Panic`) is in flight from plan 05-02
- No blockers; the module gate is fully green at plan end

---
*Phase: 05-shared-singleflight-helper*
*Completed: 2026-08-06*

## Self-Check: PASSED

- All 13 modified files verified present on disk
- All 3 task commits verified in git log: `6ade60f` (feat), `2ac849c` (test), `3c76ed6` (docs)
- SUMMARY.md verified present at `.planning/phases/05-shared-singleflight-helper/05-03-SUMMARY.md`

