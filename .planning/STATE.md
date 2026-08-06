---
gsd_state_version: 1.0
milestone: v1.6
milestone_name: Cache Dedup
current_phase: 5
current_phase_name: Shared singleflight helper
status: planning
stopped_at: Phase 5 context gathered
last_updated: "2026-08-06T14:11:33.491Z"
last_activity: 2026-08-06
last_activity_desc: v1.6 roadmap created; 19/19 requirements mapped
progress:
  total_phases: 4
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# State

**Updated:** 2026-08-06

## Current Session

- **Phase:** 5 — Shared singleflight helper
- **Status:** v1.6 roadmap created (Phases 5-8), ready to plan Phase 5

## Current Position

Phase: 5 of 8 (Shared singleflight helper)
Plan: —
Status: Ready to execute
Last activity: 2026-08-06 — v1.6 roadmap created; 19/19 requirements mapped

Progress: [░░░░░░░░░░] 0%

## Accumulated Context

### Decisions

- Shared `singleflightGetOrSet[K, V]` helper in the cache package wraps only the setter — fast-path Get stays outside the group (validated spike 005)
- DoChan + select for cancelable waiters; wrapper recovers panicking setters and returns error to all waiters (spikes 008, 003)
- Get double-check inside the fn guards against clobbering a concurrent direct Set (spike 005)
- Delete-during-flight guarded by generation-tombstone re-check before Set — `Forget` alone is NOT sufficient (spike 007, verified failure)
- SF-09 formally maps to Phase 6 (per-provider TTL parity only verifiable post-integration); Phase 5 covers the helper-level race core

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-08-06T13:09:04.983Z
Stopped at: Phase 5 context gathered
Resume file: .planning/phases/05-shared-singleflight-helper/05-CONTEXT.md
