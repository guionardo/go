---
phase: 10
phase_name: "Package Foundation — API Contract + Repo Cleanup"
project: "github.com/guionardo/go"
generated: "2026-09-30"
counts:
  decisions: 7
  lessons: 4
  patterns: 6
  surprises: 4
missing_artifacts:
  - "10-UAT.md"
---

# Phase 10 Learnings: Package Foundation — API Contract + Repo Cleanup

## Decisions

### Never-fail API contract
`Probe(folder) (ProjectData, error)` reserves errors for hard folder-level I/O (3 sentinels: ErrFolderNotFound/ErrNotDirectory/ErrPermissionDenied, always wrapped with the cleaned path); every content outcome — empty, ignored-only, unrecognized — returns `LanguageUnknown` + nil error.

**Rationale:** A probe is best-effort detection; content outcomes are not failures. The contract is documented in doc.go (FND-02) and propagated to every helper signature `([]byte, bool)` / `(ProjectData, bool)`.
**Source:** 10-02-SUMMARY.md

### syscall errno mapping with errors.Is, never fs.ErrNotExist
`mapFolderError` matches `syscall.ENOENT/ENOTDIR/EACCES` + `EPERM` via `errors.Is` — `fs.ErrNotExist` conflates ENOENT+ENOTDIR and would misclassify a file path as a missing folder.

**Rationale:** D-04 contract requires distinct sentinels for missing folder vs not-a-directory; the errno mapping is the only portable way to distinguish them.
**Source:** 10-02-SUMMARY.md

### TDD RED adaptation: verified-but-uncommitted, evidence in commit body
Tests are written first and verified failing (exit 1), then committed together with the implementation in the `feat` commit whose message body records the RED evidence.

**Rationale:** The repo's pre-commit go-test hook runs `go test ./...` — a RED-only commit would break CI, so it cannot land. User-approved (STATE.md decision), carried through phases 11-14.
**Source:** 10-02-SUMMARY.md

### No empty git commit for untracked-file deletion
Deleting the untracked `project_detector/` sample produces no git diff; FND-01's "own commit" clause is satisfied by task isolation + `go build ./...` verification instead.

**Rationale:** Empty commits are forbidden by repo rule; the untracked reality is documented in the next real commit's message body (RESEARCH OQ-2 resolution).
**Source:** 10-01-SUMMARY.md

### Size cap via limit+1 truncation probe
`io.LimitReader(f, maxManifestSize+1)` + ReadAll + `len > cap` → non-match — exactly 1 MB accepted, over-cap rejected, never silently truncated, never OOM.

**Rationale:** The +1 probe is exact and race-free; truncation would silently corrupt manifests. `maxManifestSize = 1 << 20` is the single source of truth consumed by the probe and the tests.
**Source:** 10-03-SUMMARY.md

### GOTOOLCHAIN pin for golangci-lint
`GOTOOLCHAIN=go1.26.4` is required for the pre-commit lint hook — the installed golangci-lint v2.12.2 (built with go1.26.3) panics under the go1.27.0 system toolchain.

**Rationale:** Pre-existing environment issue affecting every package in the repo; worked around per-commit rather than fixing the tool (permanent fix: `make install-golangci`).
**Source:** 10-02-SUMMARY.md

### Named returns `(content []byte, ok bool)` helper signature
`readManifest` and the registry's `detectFunc` both use the bool-degrade shape — no error channel anywhere in the package's read paths.

**Rationale:** D-03 never-fail contract consistency: missing/unreadable/dir/over-cap all degrade to `ok=false` and callers fall to Unknown.
**Source:** 10-03-SUMMARY.md

---

## Lessons

### RED failure shape differs from plan prediction
The plan predicted "no non-test Go files in" for the RED; the actual failure was `undefined: ...` compile errors — an internal test package compiles its test files into the test binary, so the missing symbols surface instead.

**Context:** Task 1 RED verification (plan 10-02). Same RED intent either way (target tests cannot pass without the API); the compile-fail form is the strongest RED.
**Source:** 10-02-SUMMARY.md

### Deletion of untracked files leaves no git trace
`rm -rf project_detector/` produced zero git diff — `git rm` fails with "pathspec did not match any files" on untracked paths. The deletion is only provable via filesystem checks (`test ! -d`) and greps.

**Context:** Plan 10-01, FND-01. Verification must rely on filesystem assertions, not git history.
**Source:** 10-01-SUMMARY.md

### Coverage gate failures can be pre-existing and out of scope
`make coverage-quick` exited 2 on `release/update.go` at 68.9% vs 70% file threshold — reproducible before any project_probe change, unrelated to the phase (project_probe at 100%).

**Context:** Scope-boundary rule: documented in deferred-items.md, not fixed here. (Later closed in Phase 14.)
**Source:** 10-02-SUMMARY.md

### First lint run surfaces a wall of issues
63 lint issues across 8 new files on the first golangci-lint run: goconst (7), gosec (8), testifylint (2), wsl_v5 (22), formatting (24).

**Context:** GREEN verification (plan 10-02). Deduplicated test literals, lowered modes to 0o600/0o700, switched to require.NotErrorIs, ran `golangci-lint fmt`.
**Source:** 10-02-SUMMARY.md

---

## Patterns

### Panic-recovery dispatch: a panicking detector is a non-match
Named-return + `defer recover()` in `callDetector` — recovered panics return the zero `(ProjectData{}, false)`, never an error, never a re-panic (cache/singleflight `callSetter` precedent).

**When to use:** Any registry/dispatch over user-extensible or untrusted code paths where one bad member must not poison the whole probe.
**Source:** 10-02-SUMMARY.md

### Internal test package for unexported-surface tests
Tests use `package projectprobe` (not `projectprobe_test`) to reach unexported symbols (`mapFolderError`, `hasContent`, `runDetectors`).

**When to use:** When the contract under test is unexported internals; config package precedent.
**Source:** 10-02-SUMMARY.md

### Save/restore package-level state in registry tests — never t.Parallel
Registry tests snapshot/restore the package-level `detectors` slice; `t.Parallel()` is forbidden (AGENTS.md global-state rule — parallel subtests race on the shared slice).

**When to use:** Any test mutating package globals; verified by grep (`0 non-comment t.Parallel occurrences`) in every phase's self-check.
**Source:** 10-02-SUMMARY.md

### Clean-once folder path
`filepath.Clean(as-given)` computed once and reused for stat, readdir, error wrapping, and the `Folder` field — relative input stays relative; `Probe("")` → ErrFolderNotFound via the OQ-1 guard, never probing the cwd.

**When to use:** APIs returning the input path as data — canonicalize once, never twice.
**Source:** 10-02-SUMMARY.md

### Limit+1 truncation probe as the repo's capped-read idiom
`io.LimitReader(f, cap+1)` + ReadAll + length check (httptest_mock/request.go:213 precedent).

**When to use:** Any bounded file/stream read where over-cap must be rejected, not truncated.
**Source:** 10-03-SUMMARY.md

### UTF-8 BOM strip via bytes.TrimPrefix
`bytes.TrimPrefix(content, utf8BOM)` — three bytes, one call, no state machine; BOM-only files are valid with empty content.

**When to use:** Any text reader that must tolerate a leading BOM; reuse the shared `utf8BOM` constant (single source of truth).
**Source:** 10-03-SUMMARY.md

---

## Surprises

### Pre-commit hooks make a literal RED commit impossible
The go-test hook runs `go test ./...` on every commit — a failing tree can never be committed, so the canonical TDD "RED commit" cannot exist in this repo.

**Impact:** Required a user-approved adaptation (verified-but-uncommitted + evidence in commit body) that became the project's TDD contract for all later phases.
**Source:** 10-02-SUMMARY.md

### golangci-lint built with an older Go panics under the newer toolchain
The installed binary (built with go1.26.3) crashed with "file requires newer Go version go1.27" — not a lint finding, a driver incompatibility.

**Impact:** Every Go commit needed `GOTOOLCHAIN=go1.26.4`; documented as a pre-existing environment issue with a `make install-golangci` permanent fix.
**Source:** 10-02-SUMMARY.md

### A deprecated untracked sample can silently break the module build
`project_detector/` (12 untracked files) failed `go build ./...` with 3 errors from go.sum entries that were never added — untracked files are invisible to git but not to the build.

**Impact:** FND-01 became the phase's wave-1 precondition; go.mod/go.sum verified untouched (`git diff --stat` empty) — no cleanup existed to perform.
**Source:** 10-01-SUMMARY.md

### Project_probe hit 100% coverage on first measurement
All 10 files at 100% statement coverage in plan 10-02 — the internal-package test strategy plus probe-level fixtures made the coverage gate trivial for this package (later phases diluted it to 93-96% as detectors grew).

**Impact:** Coverage thresholds (pkg ≥80%, file ≥70%) held without overrides; the total only passed because project_probe offset the known-red.
**Source:** 10-02-SUMMARY.md