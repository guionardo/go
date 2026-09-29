# Deferred Items — Phase 10

Out-of-scope discoveries logged during execution (scope boundary rule: only
auto-fix issues directly caused by the current task's changes).

## Deferred Items

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| pre-existing | `make coverage-quick` fails on `release/update.go` (68.9% vs 70% file threshold) — pre-existing, unrelated to project_probe (which is at 100%); blocks the coverage gate for every commit until fixed | resolved | 2026-09-29 | v1.7 |

> **Resolved 2026-09-29 (Phase 14, plan 14-03):** `release/update.go` coverage closed to 95.9% (71/74 statements) via test-only additions (no-options module-derivation row + seven error-path rows). `make coverage-quick` now passes — file 70 / pkg 80 / total 75 thresholds satisfied (80.7% total). This deferred record is stale; the gate it blocked has been green since Phase 14.
