---
phase: 16
review: 16-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "`retryBusyWithin` can overrun its budget by a full extra `fn` call"
  - id: WR-02
    severity: warning
    disposition: open
    title: "`chunksOf` hangs forever (infinite loop, unbounded allocation) on `size <= 0`"
  - id: IN-01
    severity: info
    disposition: open
    title: "MSet marshal failure error does not identify the failing key"
  - id: IN-02
    severity: info
    disposition: open
    title: "MGet logs a Warn for every call against a closed provider"
  - id: IN-03
    severity: info
    disposition: open
    title: "Double `cache/sqlite:` prefix persists on the batch surface (Phase 15 IN-01, still open)"
  - id: IN-04
    severity: info
    disposition: open
    title: "MSet does not dedup colliding stringified keys, so the stored value is nondeterministic"
  - id: IN-05
    severity: info
    disposition: open
    title: "`coverage` job pins unpinned third-party actions at old majors (pre-existing)"
open: 7
total: 7
recorded: 2026-10-08T23:20:43.845Z
---

# Phase 16: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
