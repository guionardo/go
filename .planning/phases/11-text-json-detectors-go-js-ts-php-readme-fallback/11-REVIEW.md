---
phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
reviewed: 2026-09-29T16:45:00Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - project_probe/readme.go
  - project_probe/detect_go.go
  - project_probe/json.go
  - project_probe/detect_javascript.go
  - project_probe/detect_php.go
  - project_probe/manifest.go
  - project_probe/registry.go
  - project_probe/doc.go
  - project_probe/readme_test.go
  - project_probe/detect_go_test.go
  - project_probe/detect_javascript_test.go
  - project_probe/detect_php_test.go
  - project_probe/manifest_fifo_test.go
  - project_probe/manifest_fifo_windows_test.go
  - project_probe/manifest_test.go
  - project_probe/registry_test.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-09-29T16:45:00Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

The three detectors (Go@0, JS/TS@3, PHP@6), the shared `readJSONManifest` helper, the `readmeDescription`/`firstRealParagraph` fallback chain, the WR-01 FIFO regular-file gate (`O_NONBLOCK` + `f.Stat().Mode().IsRegular()`), and the 7-slot nil registry literal were reviewed against the locked contracts D-01..D-12 in 11-CONTEXT.md / 11-RESEARCH.md. Verification performed: `go test` (96 pass), `go test -race`, `go vet`, `golangci-lint`, and `GOOS=windows go build` — all green. `syscall.O_NONBLOCK` exists on Windows as `0x00800` and is a harmless no-op in `os.OpenFile`'s CreateFile flag mapping, so the WR-01 gate is cross-platform safe (the FIFO test row is correctly Unix-gated with a build-tagged `makeFIFO` stub).

The never-fail contract holds everywhere I traced: every failure path (missing/oversized/BOM'd/malformed manifest, FIFO, unreadable folder) degrades to a non-match or `""`, never an error or a hang. The go.mod parser correctly isolates `toolchain`/`gopher`/`modulex` lines, first-occurrence wins, and the JSON detectors correctly pin D-06 (no `Private` field), duplicate-key last-wins, and type-mismatch → non-match.

Two heuristic gaps in the README extraction were confirmed by executing the production logic against real-world inputs: the plain (unwrapped) image badge form `![alt](url)` is **not** recognized as a badge (only the wrapped `[![...](...)](...)` form strips to empty), and multi-line HTML comment preambles leak into the Description. Both produce exactly the noise D-09 exists to remove, on common README shapes. One carry-over note: the Phase 10 CR-01 (Windows errno mapping, `errors.go`) and WR-02 (panic logging) remain unfixed — documented in IN-03.

## Warnings

### WR-01: `isBadgeLine` misses the plain image badge form — raw badge markdown becomes the Description

**File:** `project_probe/readme.go:113-135` (key: line 134)

**Issue:** `isBadgeLine` only strips markdown link segments starting at the last `[` before `](`, which leaves the image marker `!` (and any trailing `)` residue) behind for the unwrapped form. Verified against the production logic (replica executed):

| Input | `isBadgeLine` |
|---|---|
| `[![logo](x)](y)` (wrapped — the only tested form) | `true` |
| `![logo](x)` (plain image) | `false` — remainder `"!"` |
| `![Build Status](https://travis-ci.org/foo/bar.svg?branch=master)` | `false` |
| `![npm](https://img.shields.io/npm/v/pkg.svg)` | `false` |

The plain `![alt](url)` form is the canonical markdown image and appears as the first line of a large share of real READMEs (shields.io status images, logos). Consequence: `readmeDescription` returns the raw badge markdown as `Description` — precisely the noise D-09's "image-only badge lines" skip rule exists to remove. The test matrix pins only the wrapped form (`badge_only` row), so the gap is invisible to the suite. The RESEARCH rule (`[^\]]*\]\([^)]*\)` segments strip to empty) does handle the plain form; the implementation drifted from it.

**Fix:** treat a remainder of only `!` characters as a badge line:

```go
return strings.TrimSpace(line) == "" || strings.Trim(line, "!") == ""
```

and add the missing test row (Pitfall 5 matrix):

```go
{"plain_image_badge", "![logo](x)\npara\n", "para"},
```

### WR-02: Multi-line HTML comment preambles leak into the Description

**File:** `project_probe/readme.go:73-74`

**Issue:** Only single-line `<!-- ... -->` lines are skipped. A multi-line comment block — common at the top of generated READMEs (banners, license headers, "NOTE" admonitions) — is not skipped: the opening `<!--` line fails the `HasSuffix("-->")` conjunct, so it (and the comment body and closing `-->`) are treated as prose. Verified against the production logic:

```
Input:  "<!--\nGenerated banner\n-->\nReal description\n"
Output: "<!-- Generated banner --> Real description"   (want: "Real description")
```

The description then starts with an HTML comment — same noise class WR-01 removes.

**Fix:** track a comment state across lines:

```go
case trimmed == "":
    if len(para) > 0 {
        return strings.Join(para, " ") // first block is complete
    }
    prev = ""
case strings.HasPrefix(trimmed, "<!--"):
    prev = "" // enter comment block — skip until the closing "-->"
case strings.HasSuffix(trimmed, "-->") && len(para) == 0 && prev == "":
    prev = "" // close the comment block; lines between are consumed by the
    // block state, not by para
```

(The precise shape is executor discretion — the contract is: no `<!--`-block content ever enters `para`. Add a matrix row `{"multi_line_html_comment", "<!--\nBanner\n-->\npara\n", "para"}`.)

## Info

### IN-01: Dead condition in the underline-removal branch

**File:** `project_probe/readme.go:69`

**Issue:** `para[len(para)-1] == prev` is always true whenever `prev != ""`: `prev` is assigned only in the `default` (append) branch, and every other branch resets it to `""`. The condition documents intent but can never be false — the guard `len(para) > 0` is the only meaningful check.

**Fix:** simplify to `if prev != "" && len(para) > 0 { para = para[:len(para)-1] }` (or drop `prev` entirely and pop unconditionally when `len(para) > 0` — the underline always refers to the last appended line).

### IN-02: `detect_go_test.go` asserts a stdlib constant inside the detector table loop

**File:** `project_probe/detect_go_test.go:251-253`

**Issue:** The `folder_base_nested` special case asserts `filepath.Base(filepath.Clean(".")) == "."` — a statement about Go's stdlib, not about any package code. It runs inside every iteration of the table loop (guarded by name), adds noise, and tests nothing the package owns.

**Fix:** remove the assertion (the row's real assertion is `wantName == filepath.Base(folder)` on the nested path, which already pins A7); the `"."` edge is stdlib behavior.

### IN-03: Phase 10 CR-01 (Windows errno mapping) and WR-02 (panic logging) remain open

**File:** `project_probe/errors.go:25-36` (unchanged) — carry-over from `.planning/phases/10-*/10-REVIEW.md`

**Issue:** The Phase 10 review classified CR-01 ("D-04 errno mapping broken on Windows — `ErrPermissionDenied` unreachable, missing-intermediate-directory misclassified as `ErrNotDirectory`") as must-fix-before-ship, and WR-02 ("recovered detector panics are silently discarded — no log trace") as a warning. Phase 11 fixed WR-01 (FIFO gate) but deliberately deferred both of these (CONTEXT: "advisory; consider WR-01 fix if touching readManifest" — only WR-01 was considered). With the registry now live, every probe reaches the detector path, so a panicking detector (Phase 12-13 parsers) is still fully invisible in production, and the Windows sentinel misclassification is unchanged. Noting here so the pipeline tracks them; they are Phase 10 scope, not Phase 11 regressions.

**Fix:** defer to the Phase 14 ROBT-05 audit / a dedicated fix phase; at minimum reference 10-REVIEW.md CR-01/WR-02 in the milestone's deferred-items log.

---

_Reviewed: 2026-09-29T16:45:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_