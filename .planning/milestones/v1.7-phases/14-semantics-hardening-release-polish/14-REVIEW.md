---
phase: 14-semantics-hardening-release-polish
reviewed: 2026-09-29T00:00:00Z
depth: standard
files_reviewed: 34
files_reviewed_list:
  - project_probe/fuzz_test.go
  - project_probe/testdata/fuzz/FuzzJSONManifest/bom-prefixed.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/composer.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/duplicate-keys.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/malformed-truncated.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/package.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/version-null.json
  - project_probe/testdata/fuzz/FuzzTOMLManifest/cargo-real.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/cargo-workspace.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-dynamic.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-poetry.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-real.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/toml-6quote.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/toml-bracket-string.toml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-oldstyle.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-sdk.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-whitespace-only.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/pom-child.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/pom-pretty.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/xml-truncated.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/xml-whitespace-only.xml
  - project_probe/readme.go
  - project_probe/detect_csharp.go
  - project_probe/detect_java_kotlin.go
  - project_probe/registry_test.go
  - project_probe/readme_test.go
  - project_probe/detect_csharp_test.go
  - project_probe/detect_java_kotlin_test.go
  - project_probe/doc.go
  - release/update_test.go
  - release/update.go (context)
  - release/release.go (context)
  - README.md
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-09-29
**Depth:** standard
**Files Reviewed:** 34 (all Phase-14 touched files + supporting context files)
**Status:** issues_found

## Summary

Phase 14 (semantics, hardening, release polish) was reviewed against its locked contracts (D-01..D-10), the 14-AUDIT.md claims, and the 14-RESEARCH.md patterns. The core deliverables verify cleanly:

- **Fuzz targets (D-04/D-05):** all 20 corpus seeds are correctly `go test fuzz v1`-encoded and execute under plain `go test` (34 seed cases pass); a 5s live `-fuzz` run of each target completed with **zero crashes/panics** (JSON: 1324 execs, TOML: 5025 execs, XML: 780 execs). Fuzz bodies correctly never assert detector bools (Pitfall 3 compliance).
- **Fold-in fixes (D-06/D-07):** readme.go plain-badge + comment-block state, per-field XML `TrimSpace` in both XML detectors, `.csproj` exact-name guard, explicit empty-slice registry test — all present with the review-prescribed shapes and test rows; `gofmt`-level analysis confirms the removed `para[len(para)-1] == prev` condition was indeed dead (IN-01 cleanup is behavior-neutral).
- **Coverage gate (D-09):** reproduced — `make coverage-quick` PASS (80.7%, 2281/2828), matching the audit exactly. `release/update.go` untouched, test-only closure as contracted.
- **Anti-feature audit (D-03):** all six grep families re-run → zero matches, verified.
- **doc.go / README (D-02/D-10):** Version-semantics contract accurate against all 7 detector code paths; README cascade row matches the registry order (Go → Python → C# → JS → Rust → Java → PHP).

Issues found are correctness/quality gaps in the newly-hardened badge logic, test-reliability hazards in the new release tests, and a formatting-gate gap that contradicts the phase's own delta-zero evidence.

## Warnings

### WR-01: `isBadgeLine` misses multiple plain badges on one line — raw markdown leaks into Description

**File:** `project_probe/readme.go:149`
**Issue:** The WR-01 fix (`strings.Trim(line, "!") == ""`) only recognizes a *single* plain badge. A line with two plain badges — the common README pattern `![CI](url) ![Coverage](url2)` — strips to the remainder `! !`, which is not all-`!` and not whitespace, so the line is not a badge and the **raw markdown is emitted verbatim as the project Description**. Verified empirically:

```
isBadgeLine("![ci](a) ![coverage](b)") = false   // leaks "![ci](a) ![coverage](b)" into Description
isBadgeLine("![a](b) [![c](d)](e)")     = false   // mixed plain+wrapped also leaks
isBadgeLine("[![a](b)](c) [![d](e)](f)") = true   // wrapped-only form works
```

The wrapped form works in multiples because each `[...](...)` strips to empty; the plain form leaves one `!` per badge. This is the same defect class the phase was fixing (11 WR-01) and the exact feature under polish.

**Fix:** treat a remainder of only `!` characters and whitespace as a badge:

```go
return strings.TrimSpace(line) == "" || strings.TrimSpace(strings.ReplaceAll(line, "!", "")) == ""
```

`![a](b) Welcome` → `Welcome` (still not a badge, D-09 preserved); `![ci](a) ![coverage](b)` → `""` (badge). Add matrix rows `{"two_plain_badges", "![ci](a) ![cov](b)\npara\n", "para"}` and `{"plain_plus_wrapped", "![a](b) [![c](d)](e)\npara\n", "para"}` to `TestFirstRealParagraph`.

### WR-02: `require` called inside httptest handler goroutine — FailNow in a non-test goroutine

**File:** `release/update_test.go:256` (new test; pre-existing instance at line 195)
**Issue:** `TestCheckForUpdate_NoOptionsDerivesOwnerRepo` calls `require.Equal(t, ...)` inside the `httptest` handler, which runs on the server's own goroutine. On mismatch, testify invokes `t.FailNow()` → `runtime.Goexit()` on a **non-test goroutine** — explicitly forbidden by the `testing` package docs. The handler goroutine dies without writing a response, so the client-side `require.NoError(t, err)` then fails with a confusing network/EOF error that hides the actual path-mismatch; the intended assertion message is never shown. The new test replicates the pre-existing anti-pattern from `TestCheckForUpdate_WithToken` (line 195), so the defect is carried forward into the coverage-closure rows rather than corrected.

**Fix:** assert inside the handler with a goroutine-safe mechanism and make the client side fail meaningfully:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/repos/guionardo/go/releases/latest" {
        t.Errorf("handler path = %q, want %q", r.URL.Path, "/repos/guionardo/go/releases/latest")
        w.WriteHeader(http.StatusNotFound)
        return
    }
    _, _ = fmt.Fprint(w, `{"tag_name": "v2.0.0", "name": "v2.0.0", "assets": []}`)
}))
```

`t.Errorf` is safe from any goroutine (only `FailNow` is restricted). Consider applying the same treatment to the pre-existing line 195 instance.

### WR-03: Phase-14 files fail the repo's own formatter gate; the audit's "0 issues" delta evidence is vacuous

**File:** `project_probe/fuzz_test.go:18,75`; `project_probe/detect_csharp.go:119`; `project_probe/detect_java_kotlin.go:129`; `project_probe/detect_csharp_test.go:469`; `project_probe/detect_java_kotlin_test.go:406`; `project_probe/readme_test.go`
**Issue:** The D-10 contract ("pre-commit gofmt/goimports enforced on touched files", RESEARCH Pattern 5) is not met:

- `gofmt -l` flags **5 phase-touched files** on the committed tree: `fuzz_test.go` (NEW — missing final newline at :75 plus a misaligned trailing comment at :18), `detect_csharp.go`, `detect_java_kotlin.go`, `detect_csharp_test.go`, `detect_java_kotlin_test.go` (all end with `}` and no trailing newline).
- `GOTOOLCHAIN=go1.26.4 golangci-lint fmt --diff ./project_probe/... ./release/...` (the repo's configured formatters: gofmt + golines + goimports, `.golangci.yml`) requires rewrites in **6 phase-touched files** including the new `fuzz_test.go` and both phase-modified production files.
- The audit's delta-zero evidence ("`--new-from-rev=8461196` → 0 issues") reproduces in a fresh clone — but it is **vacuous**: the repo config sets `new-from-rev: HEAD` alongside `new-from-merge-base: main`, so on a clean tree the "new" diff is against HEAD itself and *nothing* can ever be reported. The 0-issue result is an artifact of the filter, not proof of cleanliness (the audit's own note about the dirty-tree fallback confirms the filter misbehaves both ways).

**Fix:** run `gofmt -w` (adds the missing final newlines and fixes the comment alignment) on the five files, and pass the phase's touched files through `golangci-lint fmt` before the tag. If `new-from-rev: HEAD` is intended as the delta base, the config should use the pre-phase commit instead so the gate measures something.

## Info

### IN-01: Comment line with trailing text after `-->` swallows the following paragraph

**File:** `project_probe/readme.go:70-72`
**Issue:** The multi-line comment opener case tests `!strings.HasSuffix(trimmed, "-->")` but not `strings.Contains(trimmed, "-->")`. A line where the comment closes mid-line with trailing content — `<!-- badge --> extra` — enters `inComment`, and because no later line contains `-->`, **the entire remainder of the README is consumed** and the description comes back empty. Verified empirically: `firstRealParagraph("<!-- badge --> extra\npara\n")` → `""` (expected `"para"`). Rare shape, but the state machine's self-closing detection should match how it closes (same `Contains` check used in the `inComment` case).

**Fix:**

```go
case strings.HasPrefix(trimmed, "<!--") && !strings.Contains(trimmed, "-->"):
    inComment = true // only enter when the comment is truly still open at EOL
    prev = ""
```

A line that both opens and contains `-->` then falls to the existing single-line skip case (consumed as one line), matching the documented "no comment content enters para" contract. Add a matrix row `{"comment_trailing_text", "<!-- note --> extra\npara\n", "para"}`.

### IN-02: Dead `mu.Lock()` + inaccurate `//nolint:paralleltest` rationale on three DownloadUpdate error rows

**File:** `release/update_test.go:401-420, 423-446, 449-483`
**Issue:** `TestDownloadUpdate_MkdirAllError`, `TestDownloadUpdate_CreateError`, and `TestDownloadUpdate_DigestMismatch` take `mu.Lock()` and carry the comment `//nolint:paralleltest // global state mutation: githubAPIBase`, but none of them reads or writes `githubAPIBase` — the lock and the rationale are dead weight (harmless, consistent with the file's discipline, but misleading for the next reader; `t.Parallel()` alone would be safe for these three).

**Fix:** drop `mu.Lock()/defer mu.Unlock()` from the three rows (or keep the lock only where `githubAPIBase` is actually touched) and change the nolint rationale to match, e.g. `//nolint:paralleltest // file-local httptest fixtures only`.

---

## Verified claims (not findings)

- Fuzz corpus encoding: all 20 seeds well-formed (`go test fuzz v1` header + single-line `[]byte(...)`); no raw manifests (Pitfall 1 avoided); readable filenames (A7).
- No-panic invariant: live `-fuzz` runs of all three targets — no crashes; seed mode on CI-safe.
- Coverage gate: `make coverage-quick` PASS 80.7% (2281/2828) — audit exact.
- Anti-feature greps: all six families zero matches.
- `go vet ./...` clean; `GOOS=windows go vet` claim consistent with the code (no OS-specific constructs added this phase).
- Test assertions match `update.go` error strings ("invalid character", "failed to get latest release", "failed deserialization", "parsing release version", "download failed") — all pass on go1.27.0; ENOTDIR triggers are cross-platform by construction.
- `TestCheckForUpdate_NoOptionsDerivesOwnerRepo` module derivation works in test binaries (`debug.ReadBuildInfo().Main.Path`).
- doc.go Version-semantics contract matches all 7 detector assignments; README cascade row matches registry order.
- Deferred dispositions ledger (D-08) recorded with rationale; 14-AUDIT.md section 2 accurate.

---

_Reviewed: 2026-09-29_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_