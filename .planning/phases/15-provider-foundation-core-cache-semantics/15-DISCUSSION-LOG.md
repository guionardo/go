# Phase 15: Provider Foundation + Core Cache Semantics - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-08
**Phase:** 15-provider-foundation-core-cache-semantics
**Areas discussed:** Constructor & location API, Optimize/checkpoint surface, Periodic sweep mechanics, Error & open-time semantics

---

## Constructor & Location API

### How should `:memory:` mode be selected?

| Option | Description | Selected |
|--------|-------------|----------|
| Both | `WithMemory()` explicit option + empty path stays the documented zero value (STOR-02) | ✓ |
| Empty path only | One way to select memory — the zero value; no `WithMemory()` | |
| `WithMemory()` only | Empty path becomes an error (would amend STOR-02) | |

**User's choice:** Both (Recommended)
**Notes:** Explicit option for discoverability in tests/services; zero value remains memory per the locked requirement.

### How should contradictory location options be handled?

| Option | Description | Selected |
|--------|-------------|----------|
| Documented precedence | memory > path > name > (zero value = memory); deterministic, no signature change | ✓ |
| Panic on conflict | Matches config's constructor-panic precedent; harsh for a cache handle | |
| Return an error | `New` returns `(BatchCache, error)` — breaks provider parity | |

**User's choice:** Documented precedence (Recommended)
**Notes:** `New` keeps the five-provider signature (no error return).

### What filename does the name-based default directory use?

| Option | Description | Selected |
|--------|-------------|----------|
| cache.db | `os.UserCacheDir()/<name>/cache.db` — directory carries identity | ✓ |
| <name>.db | Repeats the name; renaming orphans the file | |
| sqlite.db | Mirrors the package name | |

**User's choice:** cache.db (Recommended)

---

## Optimize / Checkpoint Surface

### How should explicit checkpoint/optimize controls (STOR-07) be exposed?

| Option | Description | Selected |
|--------|-------------|----------|
| Optional interface | Exported `sqlite.Optimizable`; callers type-assert the returned `BatchCache` | ✓ |
| Concrete return type | `New` returns `*sqlite.Cache[K,V]` — diverges from all five providers | |
| Options-only tuning | No explicit method; weakest — STOR-07 requires an explicit trigger | |

**User's choice:** Optional interface (Recommended)

### Which operations should the optional interface expose?

| Option | Description | Selected |
|--------|-------------|----------|
| Checkpoint + Vacuum | `Checkpoint(ctx)` = TRUNCATE checkpoint; `Vacuum(ctx)` = full VACUUM; separate cost classes | ✓ |
| Single Optimize method | Only a safe TRUNCATE checkpoint; no VACUUM path | |
| One method + flag | `Optimize(ctx, vacuum bool)` | |

**User's choice:** Checkpoint + Vacuum (Recommended)

### What shape should the `wal_autocheckpoint` tuning option take?

| Option | Description | Selected |
|--------|-------------|----------|
| pages > 0 sets | `WithAutoCheckpoint(pages)`; zero value keeps SQLite default (1000) | ✓ |
| 0 disables | Zero-value collision with "unset"; needs documentation of the trap | |
| You decide / defer | Keep default fixed; only Checkpoint/Vacuum exposed | |

**User's choice:** pages > 0 sets (Recommended)

---

## Periodic Sweep Mechanics

### How should the periodic sweep (TTL-04) work?

| Option | Description | Selected |
|--------|-------------|----------|
| Timer goroutine | `WithSweepInterval(d)` opt-in ticker; `Close` stops it; mem/postgres parity | ✓ |
| Opportunistic on writes | No goroutine; sweep every N writes; less predictable timing | |
| Manual Sweep only | Sweep on open + `Sweep(ctx)` on the interface; amends TTL-04 | |

**User's choice:** Timer goroutine (Recommended)
**Notes:** Research's "no background goroutine" guidance was superseded by the user's explicit TTL-04 requirement; goroutine is opt-in only.

### What happens when a sweep fails?

| Option | Description | Selected |
|--------|-------------|----------|
| Best-effort both | Open sweep failure swallowed; periodic skips tick and retries next interval | ✓ |
| Surface on next op | Remember error, return on next Get/Set/Delete — couples unrelated ops | |
| Stop sweeper on error | Silent permanent stop — indistinguishable from running | |

**User's choice:** Best-effort both (Recommended)

### When exactly does the open-time sweep run?

| Option | Description | Selected |
|--------|-------------|----------|
| Synchronous in New | Runs before `New` returns; worst case bounded by `busy_timeout` | ✓ |
| Lazy on first op | `New` returns fast; sweep on first operation via sync.Once | |

**User's choice:** Synchronous in New (Recommended)

---

## Error & Open-Time Semantics

### If the database cannot be opened or the path is invalid, what should happen?

| Option | Description | Selected |
|--------|-------------|----------|
| Deferred error | `New` records the error; every operation returns it wrapped (redis lazy-connect precedent) | ✓ |
| Panic in New | Fail fast; unlike all five providers | |
| Retry-until-open | Self-heal on later ops; more complexity | |

**User's choice:** Deferred error (Recommended)

### What sentinel errors should the package export (beyond cache.ErrMiss)?

| Option | Description | Selected |
|--------|-------------|----------|
| ErrClosed + ErrInvalidPath | Ops after Close; `?`/`#` path rejection; `cache/sqlite:` prefix; `errors.Is` support | ✓ |
| ErrClosed only | Invalid-path errors stay plain wrapped errors | |
| No new sentinels | Only wrapped stdlib/sql errors | |

**User's choice:** ErrClosed + ErrInvalidPath (Recommended)

---

## the agent's Discretion

- Context-cancellation shape (ctx-aware SQL, wrapped `ctx.Err()` on cancel).
- `WithSweepInterval` clamping for tiny test intervals.
- doc.go structure, runnable example shape, README index row wording.
- Internal test seams (clock injection; helper-process pattern is Phase 16).

## Deferred Ideas

- None — discussion stayed within phase scope.
- Reviewed todo (not folded): WAL two-process spike — kept for Phase 16 (`resolves_phase: 16`).

---
*Discussion log: 2026-10-08*
