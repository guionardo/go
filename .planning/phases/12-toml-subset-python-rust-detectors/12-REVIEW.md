---
phase: 12-toml-subset-python-rust-detectors
reviewed: 2026-09-29T20:30:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - project_probe/toml.go
  - project_probe/toml_test.go
  - project_probe/detect_python.go
  - project_probe/detect_python_test.go
  - project_probe/detect_rust.go
  - project_probe/detect_rust_test.go
  - project_probe/registry.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - project_probe/detect_php_test.go
findings:
  critical: 1
  warning: 1
  info: 4
  total: 6
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-09-29T20:30:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

The TOML-subset reader (`readTOMLSection`), the Python detector (`detectPython`, `[project]` → `[tool.poetry]` whole-section fallback), the Rust detector (`detectRust`, `[package]` with dotted-key `version.workspace` degrade), and the registry fill (slots 1 and 4, 5-of-7 live) were reviewed against the locked contracts D-01..D-09 in 12-CONTEXT.md / 12-RESEARCH.md. Verification performed: full suite (172 tests pass), coverage profile of the new files (toml.go ~96% aggregate, detect_python.go/detect_rust.go 100%), and an executable replica of the production reader probed against adversarial inputs.

The detector-level contracts hold: presence-match never fails (D-09), `[project]` wins whole-section over `[tool.poetry]` (D-04/D-disc-3), dotted keys degrade to `""` (D-07), registry order and positions are correct (D-08, D-11), sub-table isolation via exact header equality is pinned and works, CRLF/BOM/comment/duplicate-key handling is correct. The reader never panics on pathological input (bracket floods, 1 MB jumbles, unterminated quotes).

One BLOCKER: the skip-state machine that implements the phase's central strict-degrade guarantee (D-02/SC4 — "multi-line string/array/inline-table bodies can never fabricate headers or keyvals") is not quote-aware. Closing-delimiter detection uses `strings.Contains`/`ContainsRune` on whole lines, so a delimiter run *inside* a construct body — a `""""""` (6-quote) line inside a multi-line string, a `]`/`}` inside a quoted string within a multi-line array/inline table, or a 4-quote run on the opening line — prematurely clears (or fails to enter) the skip state, after which keyval/header-looking lines inside the construct body are parsed and **stored**. I executed the production logic (faithful replica) against five triggering inputs: every one fabricated `name = "evil"` (or switched sections via a fake `[project]` header), overwriting the real metadata. All five inputs are malformed TOML (confirmed against CPython's `tomllib`) — precisely the input class D-09's presence-match design treats as first-class and the phase's own tests pin as never-fabricate.

Carry-overs: 11-REVIEW WR-01 (plain image badge `![alt](url)` not recognized as a badge) and WR-02 (multi-line HTML comment preambles leak into Description) remain unfixed — `readme.go` was touched only by gofmt alignment this phase. Phase 10 CR-01 (Windows errno mapping) and WR-02 (panic logging) also remain open; with 5/7 detectors live the panic-invisibility concern grows.

## Critical Issues

### CR-01: Skip-state clearing is not quote-aware — multi-line construct bodies fabricate name/version/description (SC4/D-02 contract violation)

**File:** `project_probe/toml.go:25-33` (skip-state clearing), `project_probe/toml.go:157-172` (`enterSkip` same-line-close detection)

**Issue:** The three global skip states exist so "string/array/inline-table bodies can never fabricate headers or keyvals (SC4, T-12-02)" — the phase's own blocker requirement. The implementation clears them with whole-line delimiter search — `strings.Contains(line, skipDelim)` (line 26) and `strings.ContainsRune(line, skipClose)` (line 31) — with no quoted-string awareness. A delimiter appearing *inside* a quoted string in the construct body looks like a close:

- A `""""""` (6-quote) line inside a multi-line string body — TOML parses this as close+reopen — clears the skip state; the reopened string's body then parses as keyvals.
- A `]` or `}` inside a quoted string within a multi-line array/inline table clears the bracket state.
- `enterSkip`'s same-line checks (`strings.Contains(value[3:], value[:3])` at line 160, `ContainsRune(value[1:], ']'/'}')` at lines 164/168) have the same flaw: an opening line ending in a 4-quote run (`description = """a""""` = close+reopen) never enters the skip state at all, and `authors = ["A ] B",` (first `]` inside a string) never enters the bracket state.

Verified against an executable replica of the production reader (identical logic, run in a scratch module):

| Input (all malformed TOML per CPython `tomllib`) | Reader result | Contract |
|---|---|---|
| `[package]` + `description = """` … body line `text """"""` … `name = "evil"` … `"""` | `map[name:evil]` — real name overwritten | want `map[name:acme]` |
| Same with `'''` (multi-line literal) | `map[name:evil]` | want `map[name:acme]` |
| `description = """` … body `text """"""` … `[project]` … `name = "evil"` … `"""`, read as `"project"` | `map[name:evil]` — fake header inside the string body switches the section | want `map[]` |
| `description = """a""""` (4-quote run on the opening line) then `name = "evil"` | `map[name:evil]` | want `map[]` |
| `authors = [` … `"Alice ] Bob",` … `name = "evil"` (no comma) … `]` | `map[name:evil]` | want `map[name:acme]` |
| `metadata = {` … `version = "0.0.1 }",` … `name = "evil"` … `}` | `map[name:evil]` | want `map[name:acme]` |

The existing matrix only pins the *simple* body shapes (TestReadTOMLSection_MultiLineStringSkip, TestReadTOMLSection_CrossSectionSkip use plain `"""`-delimited bodies), so the suite is green while the guarantee is false. Malformed manifests are a first-class input under the D-09 presence-match design ("a garbage pyproject.toml still claims LanguagePython"), so the fabricated fields flow straight into `ProjectData.Name/Version/Description` — the exact "partial data" D-02 forbids.

**Fix:** make the skip-state scanning quote-aware:

```go
// For skipDelim (multi-line string): scan the line for runs of the delimiter
// char. A run of exactly 3 closes the string (clear the state; drop the
// remainder as today — trailing content after a close is invalid TOML).
// A run > 3 closes AND reopens (legal TOML) → stay in the skip state.
// Runs of 1-2 are content.
func closesMultiLine(line, delim string) (closed, reopens bool) {
    // e.g. for delim `"""`: walk the line counting consecutive quote runs;
    // closed  = run length == 3
    // reopens = run length  > 3  (the string restarts on the next line)
}

// For skipClose (] / }): walk the line tracking whether we are inside a
// " or ' quoted string (honoring \ escapes for "); only a bracket outside
// a string clears the state. Track nesting depth for [ / { so a nested
// array/table inside a table does not clear the outer state.
func clearsBracket(line string, closeRune rune) bool
```

`enterSkip`'s same-line-close checks must use the same quote-aware scan (a `]`/`}` inside a string on the opening line does not close the construct; a 4+ quote run on the opening line means the multi-line string reopened and the skip state must be entered).

Add matrix rows pinning no-fabrication for each shape above (6-quote body line for `"""` and `'''`, 4-quote opening line, `]`/`}` inside quoted strings in multi-line arrays/inline tables).

## Warnings

### WR-01: `TestRunDetectors_EmptyRegistry` exercises the production registry against the test CWD — stale comment, fragile by location

**File:** `project_probe/registry_test.go:21-28`

**Issue:** The comment says "the injected slice below never matches the `""` folder … this test replaces the slice wholesale" — but no slice is injected anymore and nothing is replaced. The test runs the **production** 5-detector registry against `runDetectors("")`, which resolves `readManifest("", "go.mod")` etc. to relative paths in the test's working directory (the package dir). It passes only because `project_probe/` happens to contain none of the five manifest names. Adding any of `go.mod`/`pyproject.toml`/`package.json`/`Cargo.toml`/`composer.json` to the package directory for tooling would silently flip this test to a match with a confusing failure, and the wrong comment actively misleads the next reader about what the test does.

**Fix:** inject an empty (or non-matching) slice explicitly so the test does not depend on the package directory's contents, and correct the comment:

```go
original := detectors
detectors = []detectorFunc{} // explicit: no live detectors
defer func() { detectors = original }()

pd, ok := runDetectors("x")
assert.False(t, ok)
```

## Info

### IN-01: `TestDetectorPositions` doc comment is stale — Python and Rust are live

**File:** `project_probe/registry_test.go:65-69`

**Issue:** The comment reads "the future-phase slots (Python, C#/.NET, Rust, Java) are nil until their plans land" — Python (index 1) and Rust (index 4) are filled this phase and the assertions below correctly require `NotNil`. The comment describes the Phase 11 interim state.

**Fix:** "slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13; Python and Rust are live at 1 and 4."

### IN-02: `TestReadTOMLSection_Adversarial` asserts a trivially-true condition

**File:** `project_probe/toml_test.go:339`

**Issue:** `assert.NotNil(t, readTOMLSection(...))` can never fail — the function always returns a freshly-allocated non-nil map. The rows' real value is proving no-panic on pathological input; the assertion adds noise, not coverage.

**Fix:** drop the assertion and let the call itself be the check (or use `require.NotPanics` around a call).

### IN-03: The multi-line literal-string (`'''`) skip path is never exercised

**File:** `project_probe/toml_test.go` (matrix), `project_probe/toml.go:97,159`

**Issue:** No test opens a `'''` multi-line literal string. `enterSkip`/`parseKeyValue` report 100% coverage only because Go counts the `||` statement once and every test short-circuits on the `"""` operand. The `'''` branch is byte-identical in shape to `"""`, so risk is low — but it is the same untested territory where CR-01 lived.

**Fix:** add one row: `[project]` with a `'''`-opened multi-line string whose body contains a fake keyval → empty map.

### IN-04: Phase 11/10 review carry-overs remain open

**File:** `project_probe/readme.go:113-135,73-74` (unchanged this phase); `project_probe/errors.go:25-36` (unchanged)

**Issue:** 11-REVIEW WR-01 (plain image badge form `![alt](url)` not recognized by `isBadgeLine` — raw badge markdown becomes the Description) and WR-02 (multi-line `<!-- … -->` comment preambles leak into the Description) were not fixed in this phase — `readme.go` received only gofmt alignment. Both now also affect the Python/Rust detectors' DATA-04 description fallback. Phase 10 CR-01 (Windows errno mapping) and WR-02 (silently discarded detector panics) remain open; with 5/7 detectors live and a parser that processes arbitrary manifest content, the invisible-panic concern grows.

**Fix:** track in the milestone deferred-items log; prioritize a dedicated fix phase (the badge/comment fixes are ~10 lines each per 11-REVIEW).

---

_Reviewed: 2026-09-29T20:30:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_