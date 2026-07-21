# Roadmap: go

## Milestone: 1 — Core Packages

### Phase 1: Cache Package

**Requirements:** CACHE-01 through CACHE-11
**Goal:** A generic `Cache[K, V]` abstraction over multiple backends (in-memory, Redis, Memcache, Postgres, Valkey) with each provider in its own sub-package — pluggable without code changes.
**Status:** Planned
**Plans:** 3/3 plans complete

Plans:

- [x] 02-01-PLAN.md — Base cache interface + in-memory provider + go.mod dependencies
- [x] 02-02-PLAN.md — Redis + Valkey cache providers
- [x] 02-03-PLAN.md — Memcache + Postgres cache providers
