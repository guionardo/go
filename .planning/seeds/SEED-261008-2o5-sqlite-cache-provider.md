---
id: SEED-261008-2o5
status: dormant
planted: 2026-10-08
planted_during: v1.7 (between milestones)
trigger_when: when planning the next milestone (/gsd-new-milestone)
scope: one phase
---

# SEED-261008-2o5: SQLite-backed cache provider (cache/sqlite)

Add a 6th cache backend: an embedded, persistent, SQLite-backed provider.

## Why This Matters

Persistence without infrastructure: a durable local cache for CLIs and small services that survives process restarts with no Redis/Postgres/Docker to run. Shaped like the postgres provider but embedded — it implements the existing `cacher[K, V]` primitive (GetFunc/SetFunc/DeleteFunc/CloseFunc + MGetFunc/MSetFunc/MDelFunc), is wrapped by `cache.NewConcreteCache`, and returns `cache.BatchCache[K, V]`.

## Agreed Scope (from the 2026-10-08 exploration)

- Driver: pure-Go `modernc.org/sqlite` (CGO-free; keeps Linux/macOS/Windows CI simple)
- Multi-process in scope: WAL + busy_timeout; concurrent CLI runs share one DB file
- Location: default path via `os.UserCacheDir` + cache name; empty path = `:memory:` (tests); explicit path overrides
- Expiry: lazy delete on read + sweep of expired rows on open
- Values: JSON (parity with other providers); keys: `fmt.Sprint` parity (final encoding TBD)
- Repo conventions: tests + coverage gates + doc.go + example

## When to Surface

**Trigger:** when planning the next milestone (`/gsd-new-milestone`).

## Scope Estimate

**One phase** (medium) — provider + tests + docs, comparable to the postgres provider build-out.

## Breadcrumbs

- `cache/cache.go` — Cache + BatchCache interfaces
- `cache/concrete_cache.go` — cacher primitive + NewConcreteCache wrapper
- `cache/postgres/` — closest provider analog (SQL-backed)
- `cache/redis/redis.go` — GetFunc/SetFunc/MGetFunc/MSetFunc/MDelFunc reference pattern
- `.planning/research/questions.md` — unresolved driver/WAL claims to verify with a resolved-tier pass
- `.planning/notes/sqlite-cache-backend.md` — design decisions

## Notes

The driver/WAL research pass on 2026-10-08 was tier-floored (unresolved) — verify before implementation. Open details for planning: key encoding, checkpoint/vacuum tuning, DB growth, error mapping parity, dependency approval (first embedded DB dependency).
