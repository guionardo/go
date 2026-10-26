---
phase: 17
review: 17-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: open
    title: "Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers"
  - id: WR-01
    severity: warning
    disposition: open
    title: "The \"Delete\" arm measures Set+Delete, so the reported number is not the delete cost"
  - id: WR-02
    severity: warning
    disposition: open
    title: "\"GetOrSet\" silently degrades to the warm hit path and the doc comment does not say so"
  - id: WR-03
    severity: warning
    disposition: open
    title: "The CGO-free step prints its evidence but never asserts it"
  - id: WR-04
    severity: warning
    disposition: open
    title: "The macOS race step forces cgo even though darwin is the one OS where `-race` does not need it"
  - id: IN-01
    severity: info
    disposition: open
    title: "`ExampleNew_withPath` leaks the first cache handle on every error path"
  - id: IN-02
    severity: info
    disposition: open
    title: "The canceled-MGet caveat is stated twice in the package comment"
  - id: IN-03
    severity: info
    disposition: open
    title: "MDel is described as \"idempotent\" next to an all-or-nothing MSet, but it is fail-fast and partial"
  - id: IN-04
    severity: info
    disposition: open
    title: "The `coverage` job was left out of the hardening"
  - id: IN-05
    severity: info
    disposition: open
    title: "Nothing in CI ever exercises the newly registered benchmarks"
  - id: IN-06
    severity: info
    disposition: open
    title: "The primitive arms bound key growth; the batch arms do not, so file-mode numbers drift with `b.N`"
open: 11
total: 11
recorded: 2026-10-26T23:10:00.000Z
---

# Phase 17: Code Review Disposition

> Ledger written by the `execute-phase` code-review gate. One row per finding, defaulting to
> `open`. Edit the Disposition cell (`open` | `fixed` | `skipped` | `deferred`) and record a
> reason in Source; the next run preserves what you write here.

| ID | Severity | Title | Disposition | Source |
|----|----------|-------|-------------|--------|
| CR-01 | critical | Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers | open | |
| WR-01 | warning | The "Delete" arm measures Set+Delete, so the reported number is not the delete cost | open | |
| WR-02 | warning | "GetOrSet" silently degrades to the warm hit path and the doc comment does not say so | open | |
| WR-03 | warning | The CGO-free step prints its evidence but never asserts it | open | |
| WR-04 | warning | The macOS race step forces cgo even though darwin is the one OS where `-race` does not need it | open | |
| IN-01 | info | `ExampleNew_withPath` leaks the first cache handle on every error path | open | |
| IN-02 | info | The canceled-MGet caveat is stated twice in the package comment | open | |
| IN-03 | info | MDel is described as "idempotent" next to an all-or-nothing MSet, but it is fail-fast and partial | open | |
| IN-04 | info | The `coverage` job was left out of the hardening | open | |
| IN-05 | info | Nothing in CI ever exercises the newly registered benchmarks | open | |
| IN-06 | info | The primitive arms bound key growth; the batch arms do not, so file-mode numbers drift with `b.N` | open | |
