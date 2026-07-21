# Roadmap: go

## Milestone: 1 — Core Packages

### Phase 1: Slices Package
**Requirements:** SLICE-01 through SLICE-10
**Status:** Pending

### Phase 2: Cache Package
**Requirements:** CACHE-01 through CACHE-11
**Goal:** A generic `Cache[K, V]` abstraction over multiple backends (in-memory, Redis, Memcache, Postgres, Valkey) with each provider in its own sub-package — pluggable without code changes.
**Status:** Planned
**Plans:** 3 plans

Plans:
- [ ] 02-01-PLAN.md — Base cache interface + in-memory provider + go.mod dependencies
- [ ] 02-02-PLAN.md — Redis + Valkey cache providers
- [ ] 02-03-PLAN.md — Memcache + Postgres cache providers
