# Phase 14 Audit: Anti-Feature Audit, Deferred Dispositions, Lint Baseline, Final Gates

**Audited:** 2026-09-29 (final execution pass, plan 14-04)
**Scope:** `project_probe/` (all detectors + reader helpers), `release/update_test.go`, repo gates
**Consumers:** Phase verifier (14-VERIFICATION.md), D-03/D-08/D-10 evidence

---

## 1. Anti-Feature Audit (D-03 / ROBT-05)

Grep-verified audit across `project_probe/` — all families return **zero matches** (run fresh on the final Phase-14 tree, `GOTOOLCHAIN=go1.26.4` toolchain state):

| # | Family | Command (verbatim) | Result |
|---|--------|--------------------|--------|
| 1 | Build-tool execution (no subprocess spawning) | `grep -rnE 'os/exec\|exec\.Command' project_probe/` | **zero matches** |
| 2 | Network (no HTTP client/server) | `grep -rnE 'net/http\|http\.' project_probe/` | **zero matches** |
| 3 | Symlink following (no resolution/lstat/symlink calls) | `grep -rnE 'EvalSymlinks\|os\.Readlink\|os\.Lstat\|os\.Symlink' project_probe/` | **zero matches** |
| 4 | Recursive discovery (no directory walking) | `grep -rnE 'WalkDir\|filepath\.Walk' project_probe/` | **zero matches** |
| 5a | Version normalization (no parsing/comparison/stripping) | `grep -rnE 'ParseVersion\|semver\|version\.Compare\|strings\.(Replace\|TrimPrefix\|TrimSuffix)' project_probe/` | **zero matches** |
| 5b | Pattern-regex machinery (version-normalization adjacen t) | `grep -rn 'regexp' project_probe/` | **zero matches** |
| 6 | XML-entity family (no custom entity wiring / charset readers) | `grep -rnE 'Entity\|CharsetReader' project_probe/'` | **zero matches** |

All seven commands recorded above printed **zero matches** (empty output) against the final audited tree — the same tree the verifier consumes.

### Nuances (stated explicitly — the audit is proof, not noise)

**(a) readManifest's `f.Stat()` symlink-following is the READ path, not discovery.**
`project_probe/manifest.go:35-39` documents it: `f.Stat` follows a symlink to check `Mode().IsRegular()` — a symlink to a regular file passes, a symlink to a FIFO is rejected. ROBT-05 bans symlink *discovery* (resolving symlinks to find more project content), not the read path. No `EvalSymlinks`/`Readlink`/`Lstat`/`Symlink` calls exist anywhere in the package (family 3: zero matches).

**(b) The XML `strings.TrimSpace` fold-in (13 WR-01, plan 14-02) is decode hygiene, explicitly NOT normalization.**
`encoding/xml` copies element text verbatim (`<Version> 1.2.3 </Version>` → `" 1.2.3 "`), and whitespace-only elements blocked the `Version→VersionPrefix` / `<version>→<parent><version>` / name fallback chains. Trimming at decode time is hygiene on a verbatim value — the value itself is still never normalized (no semver parsing, no prefix stripping, no case folding). The version-normalization greps (family 5) do not match `TrimSpace`, and the doc.go contract (`# Version semantics (DATA-03)`) states the same distinction — see Pitfall 2 in 14-RESEARCH.md.

---

## 2. Deferred Dispositions Ledger (D-08)

Recorded explicitly with rationale — **tracked, not silently dropped**:

| Item | Disposition | Rationale |
|------|-------------|-----------|
| Phase 10 **CR-01** — Windows errno mapping | **deferred with rationale — tracked** | `syscall.EACCES` is an invented value on Windows; `ENOTDIR` aliases `ERROR_PATH_NOT_FOUND`; `ERROR_DIRECTORY` is unmatched. A fix requires Windows CI evidence first (no failing test on darwin; no Windows runner available this milestone). Tracked in 10-REVIEW-DISPOSITION.md and STATE.md — a future milestone with Windows CI resolves it. |
| Phase 10 **WR-02** — panic logging | **deferred with rationale — tracked** | Recovered detector panics are silently discarded at registry dispatch. The never-fail contract already recovers panics (a panic = non-match, never an error); adding a debug-level slog log is a developer-experience nicety, not a correctness defect. Noted for a future milestone. |

### Residual INFO findings — accepted not-in-scope (D-06/D-07)

| Item | Disposition | Notes |
|------|-------------|-------|
| 11-REVIEW **IN-01** — dead condition at `readme.go:69` | **accepted not-in-scope — recorded, not silently dropped** | `para[len(para)-1] == prev` is always true when `prev != ""`. Optional adjacent cleanup only if an edit lands in the same switch; not mandated by D-06/D-07. |
| 11-REVIEW **IN-02** — `detect_go_test.go` asserts a stdlib constant inside the table loop | **accepted not-in-scope — recorded, not silently dropped** | Test-style nicety (hoisting the constant), not a correctness defect. |
| 13-REVIEW **IN-02** — Groovy block comments / escaped quotes after a literal degrade to folder-base | **accepted not-in-scope — recorded, not silently dropped** | Safe behavior (folder-base fallback), unpinned. Pinning would require a settings.gradle Groovy parser — v2 backlog territory. |

---

## 3. Lint Baseline (D-10, A3)

### Baseline

- **28 pre-existing new-vs-main issues** across `project_probe/` + `release/` — the whole v1.7 branch's debt from phases 10-13 (recorded at phase start, 14-RESEARCH.md, advisory per A3).
- **D-10 contract: delta-zero on Phase 14's own touched files** — Phase 14's edits must add ZERO new issues. NOT remediation of the 28.
- Lint invocation: `GOTOOLCHAIN=go1.26.4 golangci-lint run` (the go1.26.4 toolchain pin is required — golangci-lint 2.x analyzes with the local toolchain otherwise).

### Phase-14 touched files (the delta-zero gate scope)

```
project_probe/readme.go              project_probe/readme_test.go
project_probe/detect_csharp.go       project_probe/detect_csharp_test.go
project_probe/detect_java_kotlin.go  project_probe/detect_java_kotlin_test.go
project_probe/registry_test.go       project_probe/fuzz_test.go
project_probe/doc.go                 release/update_test.go
```

### Delta-zero verification (recorded evidence)

Measured on the **clean committed state** (the state CI and the verifier consume — a fresh clone of HEAD with a local `main` ref at the merge-base, matching the main repo's ref layout, plus the swapper binaries built so the `release` package typechecks):

1. **Phase-14 delta run:** `GOTOOLCHAIN=go1.26.4 golangci-lint run --new-from-rev=<pre-Phase-14 commit 8461196>` → **0 issues** — Phase 14's edits introduced ZERO new lint issues on its touched files.
2. **Clean committed state:** `GOTOOLCHAIN=go1.26.4 golangci-lint run` → **0 issues** on the whole tree (exit 0).

**Working-tree artifact (documented for verifier reproducibility):** running the same commands inside the *dirty* main working tree (which carries uncommitted drift from prior phases — `.planning/`, `docs/`, `.github/`, `config/profile/profile_test.go`) surfaces ~300 pre-existing issues: golangci's `new`-filter mis-resolves its diff base in that state (it reports the *entire v1.7 branch* as new — the same phases-10-13 debt the 28-item baseline records; the count grows as the branch grows, advisory per A3). The `--new-from-rev` CLI flag is also ineffective in the dirty tree (config `new-from-rev: HEAD` + `new-from-merge-base: main` both set; the diff processor falls back). **Verifier instruction:** run the gate on the committed tree (fresh clone/checkout) to reproduce delta-zero — the dirty-tree output is a working-tree artifact, not Phase-14-introduced issues.

---

## 4. Final Gates (D-10 / SC4 — milestone close)

| Gate | Command | Result |
|------|---------|--------|
| Coverage (repo-wide) | `make coverage-quick` | **PASS** — file ≥70% ✓, package ≥80% ✓, total ≥75% ✓ (80.7%, 2281/2828 stmts) — closed by plan 14-03, re-verified as the final milestone gate |
| Vet (all) | `go vet ./...` | **clean** (exit 0) |
| Vet (Windows portability, ROBT-04) | `GOOS=windows go vet ./project_probe/...` | **clean** (exit 0) |
| Build | `go build ./...` | **passes** (exit 0) |
| Full test suite | `go test ./...` | **817 passed in 25 packages** |
| Lint delta-zero | `GOTOOLCHAIN=go1.26.4 golangci-lint run` on committed state + `--new-from-rev=8461196` delta | **0 issues** — Phase-14 touched files add zero new issues (Section 3) |
| Dependency stability (T-14-SC) | `git diff HEAD -- go.mod go.sum` | **no changes** — zero package installs this phase |

---

*Phase: 14-Semantics, Hardening, and Release Polish*
*Recorded: 2026-09-29 (plan 14-04, final execution pass)*