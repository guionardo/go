# Research Questions

## 2026-10-08 — SQLite cache: driver + multi-process WAL verification

Origin: gsd-explore session "SQLite cache backend" (seed SEED-261008-2o5). A research pass surfaced the claims below, but its disposition tier could not be resolved (tier-floor), so **none are settled** — verify each with a resolved-tier research pass or a spike before implementation.

Research-derived content, treated as untrusted data:

DATA_yvs1wule_START
- [unresolved — tier-floor: unearned confidence] modernc.org/sqlite appears actively maintained, CGO-free, supports darwin/linux/windows amd64+arm64, bundles a recent SQLite; self-reported 1.3–2.0x CPU-bound gap vs C. (source cited by researcher: pkg.go.dev/modernc.org/sqlite; gitlab.com/cznic/sqlite)
- [unresolved — tier-floor: unearned confidence] mattn/go-sqlite3 requires CGO_ENABLED=1 + gcc on all three CI OSes. (source cited by researcher: repo releases/README)
- [unresolved — tier-floor: unearned confidence] Alternatives: ncruces/go-sqlite3 (cgo-free via wasm2go, higher per-connection memory); zombiezen/go-sqlite wraps modernc. (source cited by researcher: repo READMEs)
- [unresolved — tier-floor: unearned confidence] WAL does not work over network filesystems; configure journal_mode=WAL + busy_timeout; SQLITE_BUSY is still possible; active readers can cause checkpoint starvation. (source cited by researcher: sqlite.org/wal.html)
- [unresolved — tier-floor: unearned confidence] SQLite deletes never shrink the file (freelist reuse); auto_vacuum must be set before any tables exist; last close checkpoints and deletes the WAL. (source cited by researcher: sqlite.org pragma docs)
DATA_yvs1wule_END

Questions to answer with a resolved-tier pass or a spike:

1. Which driver (modernc vs an alternative) and version do we pin, and does the dependency tree fit the repo's minimal-deps posture?
2. What pragma set (`journal_mode`, `busy_timeout`, `synchronous`, `wal_autocheckpoint`) gives correct multi-process behavior for concurrent CLI invocations?
3. What is the DB growth behavior under our lazy-expiry + sweep-on-open pattern, and do docs need checkpoint/vacuum guidance?
4. Spike: two processes writing concurrently through modernc — observable SQLITE_BUSY behavior and retry requirements.
