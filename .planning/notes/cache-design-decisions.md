---
title: Cache Package Design Decisions
date: 2026-07-21
context: Socratic exploration of cache abstraction for go monorepo
---

# Cache Package Design Decisions

## Interface Design

- Generic `Cache[K, V]` with methods: `Get`, `Set`, `Delete`, `GetOrSet`
- Every method accepts `context.Context` for cancellation and tracing
- `Set` accepts optional per-key TTL with provider-level default fallback
- Batch operations deferred to future (seed idea tracked)

## Error Handling

- All provider errors are wrapped and returned to callers
- No silent swallowing — errors from connection failures, timeouts, etc. propagate

## Provider Architecture

- Each provider in its own sub-package (`cache/mem`, `cache/redis`, etc.)
- Independently importable; no required provider dependency in the base package

## Providers (all 5 from day one)

- In-memory, Redis, Memcache, Postgres, Valkey
