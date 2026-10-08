---
title: SQLite Cache Backend Design Decisions
date: 2026-10-08
context: Socratic exploration (gsd-explore) of a persistent infra-free cache provider — started as "a new cache type based on file system", pivoted to SQLite for concurrency control
---

# SQLite Cache Backend Design Decisions

## Idea

A 6th cache backend (`cache/sqlite`) for persistence without infrastructure — the durable local cache you reach for when you can't or don't want to run Redis/Postgres/Docker. It is embedded, not a service.

## Decisions

- **Storage: SQLite, not plain files.** Chosen over one-file-per-key after the multi-process requirement surfaced: SQLite provides transactional writes, WAL concurrency, TTL as an indexed DELETE, and batch operations as transactions — replacing hand-rolled file locking.
- **Driver: pure-Go `modernc.org/sqlite`.** No CGO; keeps Linux/macOS/Windows CI simple and cross-compilation painless. Accepted trade-off: heavier dependency tree and a modest CPU-bound gap vs C.
- **Multi-process access is in scope.** Concurrent CLI invocations may share the DB file; WAL + `busy_timeout` are the coordination mechanism.
- **Location: default + memory mode.** `WithName("myapp")` resolves under `os.UserCacheDir()`; empty path means `:memory:` (tests); an explicit path overrides both.
- **Expiry: lazy + sweep on open.** `Get` deletes/misses expired rows; the provider sweeps expired rows once when it opens. No background sweeper goroutine — it would fight the single-writer model across processes.
- **Integration: follow the existing provider pattern.** Implement `cacher[K, V]`, wrap with `NewConcreteCache`; JSON values and `fmt.Sprint` key formatting for parity with redis/postgres.

## Open Questions (for planning)

- Key encoding: raw `fmt.Sprint` vs a hash (filename/ID constraints no longer apply under SQLite, but key length/precedence still matter).
- Checkpoint/vacuum tuning: `wal_autocheckpoint`, `synchronous`, DB growth under delete-heavy TTL workloads.
- Error mapping parity (`ErrMiss`) and context cancellation behavior.
- Dependency approval: this would be the repo's first embedded DB dependency.
- Test strategy: real-file E2E vs `:memory:` unit tests; coverage-gate fit.

## Research Caveat

A research pass (2026-10-08) surfaced driver and WAL claims whose disposition was tier-floored — unresolved, not settled. They are recorded as unresolved in `.planning/research/questions.md` and must be verified (resolved-tier research pass or spike) before implementation.
