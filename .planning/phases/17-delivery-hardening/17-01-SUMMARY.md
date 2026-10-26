---
phase: 17-delivery-hardening
plan: 01
subsystem: infra
tags: [ci, github-actions, cgo, sqlite, modernc, race-detector, qual-01, workflow]

# Dependency graph
requires:
  - phase: 16-batch-surface-multi-process-hardening
    provides: the three-OS test job (matrix, two-process E2E step, `-race` step gated to `runner.os != 'Windows'`) and its reviewer-flagged Windows `-race` follow-up
  - phase: 15-provider-foundation-core-cache-semantics
    provides: the CGO-free `modernc.org/sqlite` v1.60.1 provider and the exact `modernc.org/libc` v1.77.1 pin in go.mod
provides:
  - "`.github/workflows/go.yml` `test` job: job-level `env: CGO_ENABLED: "0"` so `Build (CGO-free)`, `Run tests` and `SQLite two-process E2E` all execute without cgo on ubuntu/macos/windows"
  - "`Build (CGO-free)` step under `shell: bash`: `go env CGO_ENABLED` (visible `0`) -> `go build ./...` -> `go vet ./...` -> `go list -m modernc.org/sqlite modernc.org/libc` (pins printed in the log)"
  - "SQLite race detector: the single step-level `CGO_ENABLED: \"1\"` override, with a comment recording that the driver is pure Go but `-race` is a cgo build mode on non-darwin OSes"
affects: [17-02 (benchmarks run under the same job), 17-03 (docs), gsd-ship (CI evidence for QUAL-01), verify-work]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens / 1 task, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 472
  tasks: 1
  commits: 1
plan_head_before: 3772fe5e707e931c663132787a612d769ce9966d
plan_head_after: fa8c8545617d7705411ad9f60fb17b36d657d6b8

# Tech tracking
tech-stack:
  added: []            # no new dependencies, no new GitHub Action, go.mod untouched
  patterns:
    - "Cross-cutting build property at job level, single documented step-level exception: `env:` on the job expresses the property once, and exactly one step overrides it for a structural reason (D-01/D-02)"
    - "Visible-proof CI step: an env property is asserted by *printing* `go env <VAR>` in the log rather than by a grep guard, so the claim is auditable after the fact (D-03)"
    - "`shell: bash` on any multi-command `run:` block, because the default Windows shell (pwsh) propagates only the LAST exit code and fails open"

key-files:
  created: []
  modified:
    - .github/workflows/go.yml

key-decisions:
  - "D-01 followed exactly: `CGO_ENABLED: \"0\"` sits at job level (between `runs-on` and `steps`) and is NOT copied onto individual steps, so build, unit tests and the two-process E2E are all CGO-free by construction"
  - "D-03 extended per the CONTEXT discretion item: the `Build (CGO-free)` step runs `go env CGO_ENABLED` -> `go build ./...` -> `go vet ./...` -> `go list -m modernc.org/sqlite modernc.org/libc` and nothing after them — `go vet` is a free second gate and `go list -m` makes QUAL-01's pin half visible in the CI log (RESEARCH Open Question 2)"
  - "`shell: bash` chosen over splitting the step: RESEARCH Q2/Pitfall 2 proved a 3+ command pwsh block reports the last exit code only, so a failing `go build` would be masked by a succeeding `go vet`"
  - "The race step's comment was rewritten rather than annotated: `.planning/research/STACK.md`'s '-race still works (pure Go)' claim is right for macOS and wrong for Linux, and the step-level `CGO_ENABLED: \"1\"` is load-bearing (negative probe reproduces `go: -race requires cgo`, exit 2)"
  - "D-04 honoured: no Windows `-race` step and no mingw setup — the existing `if: runner.os != 'Windows'` gate and the follow-up note are preserved"

patterns-established:
  - "Property-at-job-level + one exception: express a build-wide invariant once (job `env`) and document the single override in place, rather than toggling it per step"

requirements-completed: [QUAL-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The three-OS `test` job enforces and visibly proves a CGO-free build: job-level `env: CGO_ENABLED: \"0\"`, a `Build (CGO-free)` step (`shell: bash`; `go env CGO_ENABLED` -> `go build ./...` -> `go vet ./...` -> `go list -m` pins) placed after `Set up Go` and before `Build swapper binaries`, and exactly one step-level `CGO_ENABLED: \"1\"` on the SQLite race detector whose Windows gate is untouched"
    requirement: "QUAL-01"
    verification:
      - kind: other
        ref: "ruby -ryaml structural gate over .github/workflows/go.yml (job env == \"0\", step name/shell/command order, race-step env + `if` gate, single `Build (CGO-free)`) — exit 0"
        status: pass
      - kind: unit
        ref: "CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1 (GOTOOLCHAIN=go1.26.4, 25 packages ok)"
        status: pass
      - kind: e2e
        ref: "CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s (TestTwoProcessContention + TestTwoProcessCrashRecovery both PASS)"
        status: pass
      - kind: unit
        ref: "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c -o /dev/null ./cache/sqlite/ -> 'go: -race requires cgo', exit 2 (proves the override is necessary); CGO_ENABLED=1 go test -race ./cache/sqlite/ -> ok"
        status: pass
      - kind: other
        ref: "go list -m modernc.org/sqlite modernc.org/libc -> v1.60.1 / v1.77.1; make coverage-quick -> file 70%, package 80%, total 75% all PASS (81.8%, 2105/2572)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The three GitHub-hosted runner legs behave as designed: ubuntu/macos/windows legs green, the `Build (CGO-free)` log contains a `0` line, ubuntu + macOS `SQLite race detector` green and the windows leg skipping that step"
    requirement: "QUAL-01"
    verification: []
    human_judgment: true
    rationale: "Requires a push/PR run on GitHub-hosted runners. No local Windows runtime exists, so the windows leg's runtime behaviour (CGO-free build/test/E2E) can only be observed in CI; the local cross-compiles are a proxy, not proof."

# Metrics
duration: 4min
completed: 2026-10-26
status: complete
---

# Phase 17 Plan 01: CGO-Free CI Proof Summary

**The three-OS `test` job now enforces `CGO_ENABLED=0` at job level and proves it in the log with a `shell: bash` `Build (CGO-free)` step (cgo flag, build, vet, both sqlite pins), while `-race` keeps running through exactly one documented step-level `CGO_ENABLED: "1"` exception.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-10-26T22:38Z
- **Completed:** 2026-10-26T22:42Z
- **Tasks:** 1 (tracer)
- **Files modified:** 1 (`.github/workflows/go.yml`, +21 / -3)

## Accomplishments

- Job-level `env: CGO_ENABLED: "0"` on the three-OS `test` job, so `Build (CGO-free)`, `Run tests` and `SQLite two-process E2E` all run without cgo — the headline promise (embedded SQLite, no C toolchain) is now enforced on every leg, and the E2E's children re-exec `os.Args[0]`, so the property is structural rather than environmental.
- New `Build (CGO-free)` step (`shell: bash`, placed between `Set up Go` and `Build swapper binaries`) printing `go env CGO_ENABLED` as the visible `0`, then `go build ./...`, `go vet ./...`, and `go list -m modernc.org/sqlite modernc.org/libc` — QUAL-01's pin half (`v1.60.1` / `v1.77.1`) now appears in the CI log with no extra job.
- Exactly one `CGO_ENABLED` exception: step-level `"1"` on `SQLite race detector`, whose `if: runner.os != 'Windows'` gate and `run:` line are byte-identical; its comment now records the supersession of the stale "`-race` still works (pure Go)" claim and keeps the Windows `-race` follow-up.
- Local reproduction of the whole job is green: `CGO_ENABLED=0` build + vet + full 25-package suite + both two-process E2E arms; `CGO_ENABLED=1 -race` clean; the negative probe (`CGO_ENABLED=0 GOOS=linux go test -race -c`) still refuses with `go: -race requires cgo` (exit 2), which is what makes the override load-bearing.
- `coverage` job, `Run tests`, `SQLite two-process E2E`, `Build swapper binaries`, the matrix list and the `name`/`permissions` blocks are unchanged (verified with a `diff` of the `coverage` job against `HEAD~`).

## Task Commits

Each task was committed atomically:

1. **Task 1: CGO-free `test` job — job env, visible build proof, single documented cgo exception** - `fa8c854` (feat)

**Plan metadata:** this SUMMARY (`docs: complete plan`) — committed separately.

## Files Created/Modified

- `.github/workflows/go.yml` — job-level `env: CGO_ENABLED: "0"`; new `Build (CGO-free)` step (`shell: bash`, four commands in order); step-level `env: CGO_ENABLED: "1"` on `SQLite race detector` plus the rewritten supersession comment. Everything else untouched.

## Decisions Made

Followed D-01..D-04 without deviation; the one discretion item the plan opened was resolved as planned:

- `go vet ./...` and `go list -m modernc.org/sqlite modernc.org/libc` joined the build step (RESEARCH Open Question 2). Justification: `go vet` is clean under `CGO_ENABLED=0` and costs ~2 s, and `go list -m` makes the pin half of QUAL-01 visible in the log without a separate job. Neither is a gate; the `shell: bash` requirement is what keeps them from masking a build failure.
- The race step's comment was **rewritten**, not merely extended, so the supersession lives in one place: the driver is pure Go, but `-race` is a cgo build mode on every non-darwin OS. Without that, a later agent could "simplify" the override away and turn the ubuntu leg red.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Both YAML-structural gates passed on the first attempt, and all eight `<verify>` commands passed without a fix-up cycle.

One environment note (not a deviation): local reproductions ran under `GOTOOLCHAIN=go1.26.4` while the host `go` is 1.27.0, matching the toolchain CI resolves from `go-version-file: go.mod`. The `-race`-requires-cgo rule was re-confirmed on this toolchain, so the D-02 rationale holds for the CI version too.

## Out-of-Scope Follow-ups (recorded, deliberately NOT fixed here)

- **`coverage` job still uses `actions/setup-go@v3` with no `go-version-file`**, so its Go version can drift past `go 1.26.4`. RESEARCH Open Question 5; the phase boundary excludes it and fixing it here would add an unrelated diff to a delivery-hardening PR.
- **Windows `-race` broadening** stays deferred (D-04) — needs a mingw setup step, no requirement behind it; the in-workflow comment still flags it.
- **Probe-fallback assumption (QUAL-01, unclassified — review manually):** the only Windows-runtime evidence is the CI windows leg plus cross-compiles. No local Windows runtime exists, so a Windows-only defect in the CGO-free build/test path would surface only as a red CI leg, never in local verification. This is why deliverable D2 is marked `human_judgment: true`.

## Next Phase Readiness

- QUAL-01 is satisfied at the workflow level and locally proven; the remaining evidence is the first CI run on the pushed PR (three green legs, a `0` line in the `Build (CGO-free)` log, race green on ubuntu/macOS and skipped on windows).
- `.github/workflows/go.yml` is now the single source of the job's build property, so plans 17-02/17-03 need no CI change; the job env applies to whatever `./cache/...` tests those plans add.
- `make coverage-quick` stays green on the committed tree: file 70% / package 80% / total 75% all PASS at 81.8% (2105/2572), `cache/sqlite` 93.2%.

---

## Self-Check: PASSED

- FOUND: `.planning/phases/17-delivery-hardening/17-01-SUMMARY.md`
- FOUND: `.github/workflows/go.yml`
- FOUND: commit `fa8c854` (Task 1)

*Phase: 17-delivery-hardening*
*Completed: 2026-10-26*
