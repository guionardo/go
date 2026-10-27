---
phase: 17
review: 17-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "The \"Delete\" arm measures Set+Delete, so the reported number is not the delete cost"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "\"GetOrSet\" silently degrades to the warm hit path and the doc comment does not say so"
  - id: WR-03
    severity: warning
    disposition: fixed
    title: "The CGO-free step prints its evidence but never asserts it"
  - id: WR-04
    severity: warning
    disposition: fixed
    title: "The macOS race step forces cgo even though darwin is the one OS where `-race` does not need it"
  - id: IN-01
    severity: info
    disposition: fixed
    title: "`ExampleNew_withPath` leaks the first cache handle on every error path"
  - id: IN-02
    severity: info
    disposition: fixed
    title: "The canceled-MGet caveat is stated twice in the package comment"
  - id: IN-03
    severity: info
    disposition: fixed
    title: "MDel is described as \"idempotent\" next to an all-or-nothing MSet, but it is fail-fast and partial"
  - id: IN-04
    severity: info
    disposition: fixed
    title: "The `coverage` job was left out of the hardening"
  - id: IN-05
    severity: info
    disposition: fixed
    title: "Nothing in CI ever exercises the newly registered benchmarks"
  - id: IN-06
    severity: info
    disposition: fixed
    title: "The primitive arms bound key growth; the batch arms do not, so file-mode numbers drift with `b.N`"
open: 0
total: 11
recorded: 2026-10-26T23:10:00.000Z
---

# Phase 17: Code Review Disposition

> Ledger written by the `execute-phase` code-review gate. One row per finding, defaulting to
> `open`. Edit the Disposition cell (`open` | `fixed` | `skipped` | `deferred`) and record a
> reason in Source; the next run preserves what you write here.

| ID | Severity | Title | Disposition | Source |
|----|----------|-------|-------------|--------|
| CR-01 | critical | Primitive benchmark arms swallow every error, so a broken cache reports fast fake numbers | fixed | 55b0f72 — provider probe + b.Fatal on every primitive op in cache/bench_test.go |
| WR-01 | warning | The "Delete" arm measures Set+Delete, so the reported number is not the delete cost | fixed |40a7936 — MDel arms now time only the delete; seeding moved outside the timed region, names unchanged |
| WR-02 | warning | "GetOrSet" silently degrades to the warm hit path and the doc comment does not say so | fixed |1e6a9b0 — GetOrSet measures the miss path (per-iteration delete); doc comment states exactly what each arm times |
| WR-03 | warning | The CGO-free step prints its evidence but never asserts it | fixed | ad010d2 — step fails unless CGO_ENABLED=0 and both driver pins resolve (.github/workflows/go.yml) |
| WR-04 | warning | The macOS race step forces cgo even though darwin is the one OS where `-race` does not need it | fixed |55340f4 — CI reworked; the darwin exemption is documented in the workflow comment rather than re-plumbing the step |
| IN-01 | info | `ExampleNew_withPath` leaks the first cache handle on every error path | fixed |3aceac2 — ExampleNew_withPath closes its first handle on every path |
| IN-02 | info | The canceled-MGet caveat is stated twice in the package comment | fixed |51cc44f — duplicated canceled-MGet caveat reduced to one statement |
| IN-03 | info | MDel is described as "idempotent" next to an all-or-nothing MSet, but it is fail-fast and partial | fixed |51cc44f — MDel documented as idempotent but not atomic (fail-fast, partial) |
| IN-04 | info | The `coverage` job was left out of the hardening | fixed |55340f4 — coverage job pinned to checkout@v4 / setup-go@v5 with go-version-file |
| IN-05 | info | Nothing in CI ever exercises the newly registered benchmarks | fixed |55340f4 — Benchmark smoke step added to CI |
| IN-06 | info | The primitive arms bound key growth; the batch arms do not, so file-mode numbers drift with `b.N` | fixed |a06e207 — batch arms iterate a bounded 1024-key ring |
