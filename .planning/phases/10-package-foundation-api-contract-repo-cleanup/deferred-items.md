# Deferred Items — Phase 10

Out-of-scope discoveries logged during execution (scope boundary rule: only
auto-fix issues directly caused by the current task's changes).

## Deferred Items

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| pre-existing | `make coverage-quick` fails on `release/update.go` (68.9% vs 70% file threshold) — pre-existing, unrelated to project_probe (which is at 100%); blocks the coverage gate for every commit until fixed | open | 2026-09-29 | v1.7 |