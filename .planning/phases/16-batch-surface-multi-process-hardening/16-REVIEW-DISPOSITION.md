---
phase: 16
review: 16-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "Unsanitized `asset.Name` enables path traversal / arbitrary file write"
  - id: IN-01
    severity: info
    disposition: open
    title: "Redundant double-close of the file handle on the error path"
  - id: IN-02
    severity: info
    disposition: open
    title: "`ctx` parameter is unused in `DownloadUpdate`"
  - id: IN-03
    severity: info
    disposition: open
    title: "`removeFile` test seam is a mutable package global without the shared mutex"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "`chunksOf` hangs forever (infinite loop, unbounded allocation) on `size <= 0`"
  - id: IN-04
    severity: info
    disposition: open
    title: "MSet does not dedup colliding stringified keys, so the stored value is nondeterministic"
  - id: IN-05
    severity: info
    disposition: open
    title: "`coverage` job pins unpinned third-party actions at old majors (pre-existing)"
open: 6
total: 7
recorded: 2026-10-26T14:20:34.037Z
---

# Phase 16: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| WR-02 | warning | fixed | 16-REVIEW-FIX.md (not in the current review) |
| IN-04 | info | open | - (not in the current review) |
| IN-05 | info | open | - (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
