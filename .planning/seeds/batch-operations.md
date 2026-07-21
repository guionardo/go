---
title: Cache Batch Operations
trigger_condition: When at least 2 downstream projects need batch cache operations (e.g., MGet, MSet, MDel)
planted_date: 2026-07-21
---

# Cache Batch Operations

Extend the cache interface with batch operations:

- `MGet(keys ...K) map[K]V` — multi-get
- `MSet(items map[K]V, ttl ...time.Duration)` — multi-set
- `MDel(keys ...K)` — multi-delete

Design consideration: batch ops should have sensible defaults for providers that lack native batching (e.g., in-memory can iterate; Memcache has `GetMulti`; Redis has pipelines).
