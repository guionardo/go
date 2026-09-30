# Phase 12: TOML Subset + Python/Rust Detectors - Research

**Researched:** 2026-09-29
**Domain:** Section-aware TOML-subset parsing, PEP 621 / Poetry pyproject.toml metadata, Cargo.toml `[package]` metadata, registry-slot extension
**Confidence:** HIGH

## Summary

Phase 12 fills registry indices 1 (Python) and 4 (Rust) in the Phase 11 nil-slot literal, backed by a new unexported section-aware TOML-subset reader (`readTOMLSection`, ROBT-03). The reader is the only genuinely new runtime code: a line-based state machine over `readManifest` output with three skip states (multi-line string, array, inline table) plus a strict value classifier. The locked grammar is a strict subset of TOML v1.0.0 — bare/`"quoted"` keys, `=` separator, `"`/`'` quoted scalars, `#` comments, `[name]` headers (including dotted `[tool.poetry]`) — and anything legal-but-unsupported (multiline strings, dotted keys, arrays, inline tables, unquoted values) degrades per-key to an absent entry (read as `""`), never a panic, never partial data. The design was validated against the official TOML v1.0.0 spec [VERIFIED: toml.io/en/v1.0.0] and the ecosystem specs for PEP 621 [CITED: packaging.python.org/specifications/pyproject-toml], Poetry `[tool.poetry]` [CITED: python-poetry.org/docs/1.8/pyproject], and Cargo `[package]` [CITED: doc.rust-lang.org/cargo/reference/manifest.html].

Three verified ecosystem facts drive the detector shapes. (1) **PEP 621**: `name` is the only statically-required key in `[project]`; `version` is required-but-may-be-`dynamic` (i.e. absent, or `dynamic = ["version"]`); `description` is an optional one-line string — so a valid `[project]` virtually always has a Name, and `dynamic` versions correctly yield `""`. (2) **Poetry**: `[tool.poetry]` name/version/description are plain strings; Poetry 2.x migrated to `[project]` and deprecated the legacy fields — so both-present files are rare transition states and whole-section precedence (`[project]` wins) is the ecosystem-correct rule. (3) **Cargo**: `name` is the only field Cargo requires; workspace members use dotted keys `version.workspace = true` / `description.workspace = true` (MSRV 1.64+) which the reader classifies as unsupported → `""` (D-07) with the README fallback still applying to description; a virtual workspace manifest (`[workspace]` without `[package]`) still matches on Cargo.toml presence (D-09) with folder-base Name.

**Primary recommendation:** Three plans mirroring Phase 11 — P01 = `toml.go` reader + pure-content test matrix (ROBT-03), P02 = `detect_python.go` + registry slot 1 (DETC-05), P03 = `detect_rust.go` + registry slot 4 + `TestDetectorPositions` final state + `doc.go` refresh (DETC-06). Both detectors match on manifest **presence** (D-09, like detectGo) — a garbage pyproject.toml/Cargo.toml still yields Language=Python/Rust with empty fields and fallbacks. Test fixtures stay inline `t.TempDir()` + `os.WriteFile` (repo convention, no `testdata/`); the reader tests themselves are pure `[]byte` tables and may `t.Parallel()`. Inherited operational facts: `make coverage-quick` stays known-red on `release/update.go` 68.9% (assert project_probe rows + total only); `golangci-lint` is broken under the default go1.27.0 toolchain and is **not enforced by CI or locally** (pre-commit hooks not installed — verified this session), so lint is advisory, not a gate; the WR-01 FIFO gate already landed in `readManifest` (O_NONBLOCK + regular-file check [VERIFIED: project_probe/manifest.go:29-43]), so the Phase 11 OQ-1 is closed.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### TOML-Subset Reader (ROBT-03)
- **D-01:** Unexported section-aware reader (~80 lines), stdlib-only. Supported grammar: section headers `[name]`, bare or `"quoted"` keys, `=` separator, `"`/`'` quoted scalar values, `#` comments, blank lines.
- **D-02:** **Strict degrade-to-empty** on legal-but-unsupported TOML: multiline strings, dotted keys, inline tables, arrays, unquoted values → empty fields for that read, never a panic, never partial data. (Phase 12 blocker: TOML strict-degrade needs fixture-driven validation.)
- **D-03:** Reader API shape: `readTOMLSection(content []byte, section string) map[string]string` (or equivalent unexported helper) — consumes `readManifest` output; section missing → empty map (not error).

#### Python Detector (DETC-05)
- **D-04:** `detectPython(folder) (ProjectData, bool)` — pyproject.toml via readManifest; primary fields from `[project]` (PEP 621: name/version/description); legacy fallback to `[tool.poetry]` name/version/description when `[project]` absent.
- **D-05:** Description → README fallback via `readmeDescription` (DATA-04); name → folder-base fallback (DATA-02) when both sections lack a name.

#### Rust Detector (DETC-06)
- **D-06:** `detectRust(folder) (ProjectData, bool)` — Cargo.toml via readManifest; `[package]` name/version/description.
- **D-07:** `version.workspace = true` → empty Version — never fabricated, never resolved from a workspace root (Phase 12 SC3). Other dynamic/unsupported version forms degrade to empty.

#### Registry Wiring
- **D-08:** `detectPython` at registry index 1, `detectRust` at index 4 — completes 5 of 7 live slots (C#/.NET index 2, Java/Kotlin index 5 remain for Phase 13). Order preserved: Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP.
- **D-09:** Both detectors follow the never-fail contract: `readManifest` failure → `(ProjectData{}, false)`; malformed TOML → strict empty → detector still matches on manifest presence (like detectGo D-disc-1) with empty/fallback fields.

### the agent's Discretion
- Exact reader implementation (scanner vs line-based), fixture layout for TOML edge cases, test row organization. Follow repo conventions (CONVENTIONS.md) and the Phase 10/11 package patterns (PATTERNS.md).

### Deferred Ideas (OUT OF SCOPE)
- Full TOML dependency (`pelletier/go-toml/v2`) — v2 backlog (FRAM-02), stdlib-only constraint holds for v1.7
- Workspace-root version resolution for Cargo — not in scope; `version.workspace` degrades to empty
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DETC-05 | Python detector — pyproject.toml `[project]` + legacy `[tool.poetry]`, PEP 621 fields | PEP 621 verified [CITED: packaging.python.org/specifications/pyproject-toml]: `name` statically required, `version` static-string-or-`dynamic`, `description` optional one-line string; Poetry legacy fields verified [CITED: python-poetry.org/docs/1.8/pyproject]: plain strings, required in package mode; Poetry 2.x deprecation of `[tool.poetry]` fields → whole-section precedence recommendation (D-disc-3); `LanguagePython Language = "Python"` [VERIFIED: project_probe/project.go:32-33] |
| DETC-06 | Rust detector — Cargo.toml `[package]` fields | Cargo manifest verified [CITED: doc.rust-lang.org/cargo/reference/manifest.html]: name only required field; version/description plain strings; workspace inheritance via dotted keys `version.workspace = true` [CITED: doc.rust-lang.org/cargo/reference/workspaces.html] → classified unsupported → `""` (D-07); virtual manifest (`[workspace]` without `[package]`) matches on presence with folder-base Name (D-09); `LanguageRust Language = "Rust"` [VERIFIED: project_probe/project.go:41-42] |
| ROBT-03 | Unexported section-aware TOML-subset reader (~80 lines); strict degrade-to-empty | TOML v1.0.0 grammar verified [VERIFIED: toml.io/en/v1.0.0]: header/keyval grammar, dotted keys, string forms, arrays, inline tables, comments, CRLF; strict-degrade classification matrix (Pattern 1) with fixture rows pinning every degrade case (STATE.md blocker: "needs fixture-driven validation") |
</phase_requirements>

## Architectural Responsibility Map

Single-package stdlib library — no browser/SSR/CDN/database tiers exist. Every capability belongs to the package core; the map prevents the planner from pushing parsing into tests, docs, or other packages.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TOML-subset parsing | API/Backend (package core) | — | `toml.go` — unexported `readTOMLSection` (ROBT-03); must live in the package, not `cmd/` or tests |
| Python metadata extraction | API/Backend (package core) | — | `detect_python.go` — D-04 section precedence + D-05 fallback chains |
| Rust metadata extraction | API/Backend (package core) | — | `detect_rust.go` — D-06 `[package]` fields + D-07 workspace-inheritance degrade |
| Registry position ownership | API/Backend (package core) | — | `registry.go` literal — fill indices 1 and 4 in place (D-08); no reordering |
| Manifest I/O safety | API/Backend (package core) | — | Existing `readManifest` (1 MB cap, BOM strip, WR-01 FIFO gate) [VERIFIED: project_probe/manifest.go:13,15,29-43] — consumed unchanged |
| Description fallback | API/Backend (package core) | — | Existing `readmeDescription` [VERIFIED: project_probe/readme.go:29-48] — consumed unchanged |
| Docs (detector list) | Docs | — | `doc.go` paragraph refresh to name Python@1 and Rust@4 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `strings` | 1.26.4 (go.mod) | `Split`, `TrimSpace`, `HasPrefix`, `Index`, `Contains`, `Trim` for the line-state machine | Hand-rolled subset parsing (ROBT-03 mandate); verified grammar needs no regex |
| Go stdlib `bytes` | 1.26.4 | Split content into lines (`bytes.Split`) | `readManifest` returns `[]byte` (Phase 10); avoids a string copy — Phase 11 precedent (parseGoMod uses `strings.Split(string(content), …)`; either is fine, pick one) |
| Go stdlib `path/filepath` | 1.26.4 | `Base` (folder-name fallback, DATA-02) | ROBT-04 — never `path`; Phase 11 precedent |
| `github.com/stretchr/testify` | v1.11.1 (test-only, existing) | `assert`/`require` | AGENTS.md mandate; already in go.mod [VERIFIED: go.mod:12] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| none | — | — | No supporting libraries needed — everything else (BOM, cap, FIFO gate, README extraction, panic recover) already exists |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled line-based reader | `pelletier/go-toml/v2` | Locked out — FRAM-02 defers the full TOML dep to v2 (CONTEXT Deferred); stdlib-only mandate (ROBT-04) governs v1.7. Also overkill: we read 3 keys per section |
| Line-based state machine | `bufio.Scanner` | `bufio.Scanner` has a default 64 KB token cap — would need `Buffer()` bumps; `bytes.Split` on a 1 MB-capped input is simpler and matches Phase 11 (parseGoMod). Line-based chosen over a rune scanner because the grammar is line-oriented (keyval must be on one line) [VERIFIED: toml.io/en/v1.0.0 §Key/Value Pair] |
| Presence-based match for Python/Rust | Parse-success match (JS/PHP D-04 style) | Locked — D-09 explicitly says "like detectGo D-disc-1": manifest presence = match, content failure = empty fields. A garbage pyproject.toml claims Python — ecosystem-correct (the file is the marker) |
| Reading both `[project]` and `[tool.poetry]` per-field | Whole-section precedence | Locked by D-04's "when `[project]` absent"; per-field mixing across sections is partial data and would fabricate hybrid metadata for migrated Poetry 2.x files (D-disc-3) |

**Installation:** none — stdlib only; testify already present.
**Version verification:** `go.mod` declares `go 1.26.4` [VERIFIED: go.mod:3]; `github.com/stretchr/testify v1.11.1` [VERIFIED: go.mod:12]. Local toolchain go1.27.0 (darwin/arm64) — forward-compatible; CI pins via `go-version-file: go.mod`.

## Package Legitimacy Audit

> Gate outcome: **N/A — no external packages installed by this phase.** All runtime code is Go stdlib (`strings`, `bytes`, `path/filepath`); `testify` is an established module dependency used test-only. The planner must not add any `go get` task — same disposition as Phase 11 (go ecosystem not covered by the package-legitimacy seam; gate satisfied by zero installs).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|----------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/stretchr/testify` v1.11.1 | Go modules | ~10 yrs | widely used | stretchr/testify | OK (existing dep) | Approved — already in go.mod, no install action |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                        ┌──────────────────────────────────────────────┐
                        │          projectprobe.Probe(folder)           │
                        └───────────────────────┬──────────────────────┘
                                                │ clean-once → stat → readdir → hasContent (Phase 10, unchanged)
                                                ▼
                                   ┌──────────────────────────────────────────┐
                                   │   runDetectors(clean)  [registry.go]     │
                                   │   detectors = []detectorFunc{             │
                                   │       detectGo,     // idx 0 (Phase 11)   │
                                   │       detectPython, // idx 1 (THIS PHASE) │
                                   │       nil,          // idx 2 C#/.NET (13) │
                                   │       detectJS,     // idx 3 (Phase 11)   │
                                   │       detectRust,   // idx 4 (THIS PHASE) │
                                   │       nil,          // idx 5 Java (13)    │
                                   │       detectPHP,    // idx 6 (Phase 11)   │
                                   │   }                                      │
                                   │   nil-skip + callDetector recover (both   │
                                   │   already in place — unchanged)           │
                                   └──────┬───────────┬───────────┬────────────┘
                                          ▼           ▼           ▼
                                    detectPython   detectRust   (existing Go/JS/PHP)
                                          │           │
                                          ▼           ▼
                        readManifest(folder,"pyproject.toml")  readManifest(folder,"Cargo.toml")
                          (1 MB cap + BOM strip + FIFO gate,     (same)
                           Phase 10 — WR-01 landed Phase 11)
                                          │           │
                                          ▼           ▼
                        readTOMLSection(content,"project")   readTOMLSection(content,"package")
                        → empty map? → readTOMLSection(       │
                          content,"tool.poetry") [D-04]        ▼
                                          │           readTOMLSection returns map[string]string;
                                          ▼           version.workspace = true → dotted key →
                        name → filepath.Base(folder) [DATA-02]  NOT stored → "" [D-07]
                        version raw or "" [DATA-03, never       name "" → filepath.Base(folder)
                          fabricated — dynamic/unsupported → ""] description "" → readmeDescription
                        description "" → readmeDescription(folder) [DATA-04]
                                          │
                                          └──────────► ProjectData{Language, Name, Version, Description}
                                                       merged into Probe's defaults — first match wins
```

Entry point: `Probe` (unchanged). Processing: existing pipeline → registry (two slots filled) → two new detectors → shared reader → shared fallbacks. Branching: each detector matches on manifest presence (D-09) or falls through; strict-degrade means empty fields, never errors/panics. External dependencies: filesystem only (ROBT-05 anti-features preserved).

### Recommended Project Structure

```
project_probe/
├── registry.go              # MODIFIED: detectPython at idx 1, detectRust at idx 4 (D-08); comment refresh
├── toml.go                  # NEW: readTOMLSection(content []byte, section string) map[string]string (ROBT-03)
├── detect_python.go         # NEW: detectPython — pyproject.toml presence-match, [project] → [tool.poetry] (D-04/D-05)
├── detect_rust.go           # NEW: detectRust — Cargo.toml presence-match, [package] fields (D-06/D-07)
├── doc.go                   # MODIFIED: detector-list paragraph names Python@1 + Rust@4
├── toml_test.go             # NEW: pure-content reader matrix (may t.Parallel — no global state)
├── detect_python_test.go    # NEW: probe-level rows (inline t.TempDir + os.WriteFile)
├── detect_rust_test.go      # NEW: probe-level rows incl. version.workspace + virtual manifest
├── registry_test.go         # MODIFIED: TestDetectorPositions — indices 1 and 4 become NotNil
├── detect_php_test.go       # MODIFIED: TestProbe_CascadePrecedence — add Python/Rust precedence rows
└── (everything else)        # UNCHANGED — readManifest, readme, json, probe, ignore, errors, example
```

Conventions honored: snake_case files, internal `package projectprobe` tests for unexported access (toml_test.go must be internal to reach `readTOMLSection`), decorder order (type → const → var → func), no `t.Parallel()` in registry-mutating tests [VERIFIED: project_probe/registry_test.go:12-14]. `_python`/`_rust` suffixes verified safe: neither appears in Go's GOOS list (`aix android darwin dragonfly freebsd illumos ios js linux netbsd openbsd plan9 solaris wasip1 windows`) nor GOARCH list (`386 amd64 arm arm64 loong64 mips mips64 mips64le mipsle ppc64 ppc64le riscv64 s390x wasm`) — verified via `go tool dist list` this session, unlike the Phase 11 `_js` GOOS collision.

### Pattern 1: Section-aware TOML-subset reader with strict degrade (ROBT-03, D-01/D-02/D-03)

**What:** A line-based state machine over `readManifest` output. Grammar supported (D-01): `[name]` section headers (including dotted `[tool.poetry]`), bare or `"quoted"` keys, `=` separator, `"`/`'` quoted scalar values, `#` comments, blank lines. Everything else legal-but-unsupported degrades per-key to **not stored** (read as `""`) — never a panic, never partial data (D-02). Design points verified against the TOML v1.0.0 spec [VERIFIED: toml.io/en/v1.0.0]:

1. **Line-oriented**: "The key, equals sign, and value must be on the same line"; headers are "square brackets on a line by themselves"; "You can tell headers apart from arrays because arrays are only ever values" — a trimmed line starting with `[` is a header. CRLF is legal ("Newline means LF or CRLF") — `TrimSpace` handles `\r`. BOM was stripped upstream by `readManifest` (Phase 10).
2. **Comment rule**: "A hash symbol marks the rest of the line as a comment, except when inside a string" — so `#` handling must be quote-aware; the design extracts quoted values *first* and ignores the remainder, which makes `#` inside quotes content automatically. For headers, take the content between the first `[` and the first `]` — a trailing `# comment` after `]` is then ignored by construction (leniency; spec grammar `std-table = %x5B ws key ws %x5D` has no trailing-comment slot, but accepting it harms nothing — A2).
3. **Header matching is exact-string**: `[project]` ≠ `[project.optional-dependencies]`; `[tool.poetry]` ≠ `[tool.poetry.dependencies]` (PEP 621 allows `[project.scripts]`, `[project.optional-dependencies]` etc.; Poetry files always carry `[tool.poetry.dependencies]` — keys in those sub-tables must never leak into the parent read). Whitespace inside a header is legal TOML (`[ d.e.f ]` = `[d.e.f]`) — TrimSpace the inner content. Quoted header segments (`[tool."poetry"]`) are legal TOML and equivalent to `[tool.poetry]`, but pathological — documented leniency: no match (A3).
4. **Key parsing**: split at the first `=`. Key side: trim; accept bare keys `[A-Za-z0-9_-]+` (spec: "Bare keys may only contain ASCII letters, ASCII digits, underscores, and dashes") or quoted keys `"…"`/`'…'` (spec: "Quoted keys follow the exact same rules as either basic strings or literal strings"). A bare key containing `.` is a dotted key → unsupported → skip (this is how `version.workspace = true` degrades, D-07). Only the three target field names are stored; all other keys are ignored (never stored).
5. **Value classification** (strict degrade matrix, in order):
   - `"""…` or `'''…` → multi-line string → **enter multi-line skip state** (see below), key not stored.
   - `[` → array (may span lines per spec) → **enter bracket-skip state** if no `]` on this line; key not stored.
   - `{` → inline table (single-line in TOML 1.0; TOML 1.1 allows newlines — skip state covers both) → **enter bracket-skip state** if no `}` on this line; key not stored.
   - `"…"` (basic string) → scan to closing `"` honoring `\` escapes (spec: `\"` and `\\` are escapes; skip the char after a backslash); content = raw between quotes, **no unescaping** (A4 — Phase 11 precedent: strip outer quotes only, never unescape interior). After the closing quote, the remainder must be whitespace or a `#` comment — otherwise the line is malformed and the key is **not stored** ("never partial data").
   - `'…'` (literal string) → same, no escape handling; `'` cannot appear inside (spec: "no way to write a single quote inside a literal string").
   - anything else (unquoted: integers, floats, booleans, datetimes, barewords — e.g. `publish = false`, `dynamic = ["version"]` is the array case) → unquoted value → **not stored** (D-02 lists unquoted values as a degrade case).
   - `key =` with no value → invalid per spec ("Unspecified values are invalid") → not stored.
6. **Multi-line / bracket skip states are GLOBAL, not section-scoped**: a multi-line string in `[build-system]` may *contain* lines that look like headers or keyvals (e.g. `[project]` or `name = "evil"` inside a `"""…"""` block); parsing them would fabricate data. While in a skip state, no line is parsed as header or keyval; the state clears when a line contains the closing delimiter (`"""`/`'''`/`]`/`}`). This is the difference between "degrades" and "fabricates" — the SC4 contract (STATE.md blocker) demands the fixture rows proving it.
7. **Duplicate keys** are invalid TOML but harmless to us: last occurrence wins (mirrors the Phase 11 `encoding/json` duplicate-keys last-wins behavior [VERIFIED: STATE.md Phase 11 decision]). A5.
8. **Never panics**: all parsing is bounds-checked string scanning on 1 MB-capped input; `callDetector` recover remains the outer net [VERIFIED: project_probe/registry.go:44-53].

Skeleton (shape only — executor discretion on helper split; keep `readTOMLSection` + 2-3 tiny helpers under ~110 lines total with comments; the ~80-line target is soft, correctness first):

```go
// readTOMLSection returns the key/values of the named TOML section.
// Supported grammar (D-01): [name] headers (dotted names like
// "tool.poetry" match exactly), bare or "quoted" keys, "="/'=' separator,
// "…"/'…' quoted scalar values, # comments, blank lines. Legal-but-
// unsupported TOML — multi-line strings, dotted keys, arrays, inline
// tables, unquoted values — is never stored, so the caller reads "" for
// those keys (D-02 strict degrade: no panic, no partial data). A missing
// section returns an empty map, never an error (D-03).
func readTOMLSection(content []byte, section string) map[string]string {
	out := make(map[string]string)
	var current string
	var skipDelim string // multi-line string delimiter when non-empty
	var skipClose rune   // ']' or '}' when in bracket-skip state
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case skipDelim != "":
			if strings.Contains(line, skipDelim) {
				skipDelim = ""
			}
			continue
		case skipClose != 0:
			if strings.ContainsRune(line, skipClose) {
				skipClose = 0
			}
			continue
		case strings.HasPrefix(line, "[["): // array-of-tables header
			current = headerName(line[2:], "]]")
		case strings.HasPrefix(line, "["): // table header
			current = headerName(line[1:], "]")
		case current != section || !strings.Contains(line, "="):
			continue
		default:
			key, value, ok := parseKeyValue(line) // quote-aware
			if !ok || key != "name" && key != "version" && key != "description" {
				continue
			}
			out[key] = value
			skipDelim, skipClose = enterSkip(value) // ""/"""…, [→], {→}
		}
	}
	return out
}
```

*(Behavior contract is the matrix in Common Pitfalls P1-P6 + the edge-case list below; exact branch structure is executor discretion.)*

**When to use:** ROBT-03 only — consumed exclusively by `detectPython`/`detectRust` (CONTEXT: "unexported, like readManifest"). `readTOMLSection` may `t.Parallel()` in tests (no global state).

### Pattern 2: Python detector — `[project]` primary, `[tool.poetry]` legacy (DETC-05, D-04/D-05)

**What:** Presence-match on pyproject.toml (D-09). Read `[project]`; when it yields nothing, read `[tool.poetry]` (D-04 "when `[project]` absent"). Then the standard DATA-02/DATA-04 chains.

```go
// detectPython reports a Python project when folder contains a pyproject.toml
// (D-09: presence-based match — malformed TOML still matches, fields degrade
// to empty). Metadata comes from [project] (PEP 621) with [tool.poetry] as
// the legacy fallback when [project] yields nothing (D-04). Content flows
// exclusively through readManifest (1 MB cap + BOM strip + FIFO gate); any
// read failure degrades to a non-match (D-09).
func detectPython(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "pyproject.toml")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	fields := readTOMLSection(content, "project")
	if len(fields) == 0 {
		fields = readTOMLSection(content, "tool.poetry") // D-04 legacy fallback
	}
	data := ProjectData{Language: LanguagePython} // "Python" [VERIFIED: project_probe/project.go:32-33]
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = fields["version"] // raw string or "" (DATA-03; dynamic/unsupported → "", never fabricated)
	data.Description = fields["description"]
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 chain
	}
	return data, true
}
```

Verified ecosystem facts this shape relies on:
- PEP 621: `name` is the only statically-required key ("The only keys required to be statically defined are: `name`") — a valid `[project]` almost always yields Name, so folder-base fires mostly for malformed files [CITED: packaging.python.org/specifications/pyproject-toml].
- `version` may be static string or `dynamic = ["version"]` (absent) — either way Version is the raw string or `""`, never fabricated (DATA-03, D-disc in Phase 11) [CITED: peps.python.org/pep-0621].
- `description` is the one-line Summary string (the full README lives in `readme`, which we do not read) [CITED: packaging.python.org/specifications/pyproject-toml].
- Poetry 1.x: `[tool.poetry]` name/version/description are plain strings, required in package mode [CITED: python-poetry.org/docs/1.8/pyproject].
- Poetry 2.x (2024+) supports `[project]` and deprecates the legacy fields — both-present files are transition states; `[project]` wins (D-disc-3) [CITED: python-poetry.org CHANGELOG #9135].

**When to use:** DETC-05. Note the `len(fields) == 0` fallback covers both "section absent" and "section present but everything unsupported" — deterministic and testable (D-disc-3).

### Pattern 3: Rust detector — `[package]` with workspace-inheritance degrade (DETC-06, D-06/D-07)

**What:** Presence-match on Cargo.toml (D-09). Read `[package]`; `version.workspace = true` is a dotted key → not stored → `""` (D-07). Same chains.

```go
// detectRust reports a Rust project when folder contains a Cargo.toml
// (D-09: presence-based match). Metadata comes from [package]: name,
// version, description (D-06). Workspace-inherited fields — version.workspace
// = true, description.workspace = true — are dotted keys the subset reader
// classifies as unsupported, so they degrade to "" (D-07: never fabricated,
// never resolved from a workspace root). A virtual workspace manifest
// ([workspace] without [package]) still matches on presence; Name falls back
// to the folder base.
func detectRust(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "Cargo.toml")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	fields := readTOMLSection(content, "package")
	data := ProjectData{Language: LanguageRust} // "Rust" [VERIFIED: project_probe/project.go:41-42]
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02
	}
	data.Version = fields["version"] // "" for version.workspace = true (D-07)
	data.Description = fields["description"]
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 — also for description.workspace = true
	}
	return data, true
}
```

Verified ecosystem facts:
- Cargo requires only `name` in `[package]` ("The only field required by Cargo is name") — a `[package]` without version is legal → Version `""` [CITED: doc.rust-lang.org/cargo/reference/manifest.html].
- `[package]` legitimately contains ARRAYS (`authors`, `keywords`, `categories`) and unquoted booleans (`publish = false`) — these must degrade per-key without disturbing the section state (P4).
- Workspace inheritance: `version.workspace = true` / `description.workspace = true` / `edition.workspace = true` inherit from `[workspace.package]` at the workspace root (MSRV 1.64+) [CITED: doc.rust-lang.org/cargo/reference/workspaces.html]. We never read the workspace root (D-07 — root-scoped, anti-feature ROBT-05).
- Virtual manifest: `[workspace]` without `[package]` is legal ("This is called a virtual manifest") — matches on presence with folder-base Name [CITED: doc.rust-lang.org/cargo/reference/workspaces.html].
- `description.workspace = true` → Description `""` → README fallback fires (same chain as any absent description).

**When to use:** DETC-06.

### Pattern 4: Registry fill + test/doc refresh (D-08)

**What:** Surgical edits to three existing files.

```go
// registry.go — fill indices 1 and 4 in place (D-08); nil-skip already exists
var detectors = []detectorFunc{
	detectGo,      // index 0 — Go (Phase 11)
	detectPython,  // index 1 — Python (Phase 12)
	nil,           // index 2 — C#/.NET (Phase 13)
	detectJS,      // index 3 — JavaScript/TypeScript (Phase 11)
	detectRust,    // index 4 — Rust (Phase 12)
	nil,           // index 5 — Java/Kotlin (Phase 13)
	detectPHP,     // index 6 — PHP (Phase 11)
}
```

`TestDetectorPositions` [VERIFIED: project_probe/registry_test.go:70-79] flips two assertions per plan (interim after P02: `detectors[1]` NotNil, `detectors[4]` still Nil; final after P03: both NotNil). `doc.go` paragraph [VERIFIED: project_probe/doc.go:35-46] refresh: "live detectors at positions 0 (Go), 1 (Python), 3 (JS/TS), 4 (Rust), and 6 (PHP); slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13." Cascade-precedence rows in `TestProbe_CascadePrecedence` [VERIFIED: project_probe/detect_php_test.go:167-172]: pyproject.toml + Cargo.toml → `LanguagePython` (index 1 beats 4); broken package.json + valid pyproject.toml → `LanguagePython` (index 1 beats 3 — Python matches on presence while JS needs parse success).

**When to use:** D-08 wiring. No `runDetectors` change needed — the nil-skip [VERIFIED: project_probe/registry.go:28-39] already handles empty slots; `callDetector` recover is unchanged.

### Discretion Decisions Taken (CONTEXT "the agent's Discretion" — locked recommendations)

| # | Decision | Rationale |
|---|----------|-----------|
| D-disc-1 | **Line-based reader** (`strings.Split` on `\n`) over a rune scanner | TOML keyvals are single-line by spec; Phase 11 parseGoMod precedent; no `bufio.Scanner` token-cap dance |
| D-disc-2 | **Unsupported values are NOT stored** (map simply lacks the key) rather than stored as `""` | Identical read result (`""` via missing-key lookup), simpler code, and "never partial data" is structurally enforced — a stored `""` would be indistinguishable from a legitimately-empty string anyway |
| D-disc-3 | **Whole-section precedence for Python**: `[project]` when its map is non-empty, else `[tool.poetry]`; no per-field mixing | D-04's "when `[project]` absent" is the contract; per-field mixing across sections would fabricate hybrid metadata in migrated Poetry 2.x files (which carry BOTH sections, legacy fields deprecated). Flagged as A1 for user confirmation |
| D-disc-4 | **Multi-line/bracket skip states are global** (tracked even outside the target section) | A `"""…"""` block in any section can contain lines that look like `[project]` headers or `name = "evil"` keyvals; parsing them fabricates data — the strict-degrade contract (SC4) requires the skip |
| D-disc-5 | **Raw extraction, no unescaping** of basic strings (outer quotes stripped only) | Phase 11 A4 precedent ("strip only matched outer quotes, never unescape interior"); TOML `\"`/`\\` sequences in name/version/description are pathological |
| D-disc-6 | **Remainder-after-close-quote must be whitespace/comment**, else the key is not stored | "Never partial data": `name = "foo" garbage` would otherwise silently yield `foo` from an invalid line |
| D-disc-7 | **Inline fixtures** via `t.TempDir()` + `os.WriteFile` for detector tests; pure `[]byte` string tables for reader tests | Repo convention (TESTING.md); reader tests need no files at all — denser matrix, maximum parallelism (no global state) |
| D-disc-8 | **Header name = content between first `[` and first `]`** (first `[[`/`]]` for array-of-tables) | Handles `[project] # comment` and `[ project ]` without explicit comment stripping; quoted segments and inner-dot-space forms are documented non-matches (A3) |

### Anti-Patterns to Avoid

- **Section-prefix matching** (`strings.HasPrefix(current, section)`): `[project.optional-dependencies]` and `[tool.poetry.dependencies]` would leak keys into `[project]`/`[tool.poetry]` reads — exact equality only.
- **Stripping `#` comments before parsing values**: `description = "hello # world"` would truncate to `hello`; extract the quoted value first, ignore the remainder (the spec's "except when inside a string" rule).
- **`strings.SplitN(line, "=", 2)` without quote-awareness on the key side**: quoted keys may contain `=` (`"a=b" = 1`); since only bare `name`/`version`/`description` matter, validate the key form first and reject quoted keys that aren't exactly the target names.
- **Treating multi-line strings as a single-line degrade**: without the skip state, `"""…"""` blocks containing fake keyvals/headers fabricate data — the SC4 "no partial data" violation.
- **Reading `[workspace.package]` to resolve `version.workspace`**: D-07 forbids it; ROBT-05 anti-features (root-scoped only); empty Version is the locked contract.
- **A `dynamic` field or `{file = …}` version workaround**: `dynamic = ["version"]` is an array (unsupported → version absent → `""`) and any table-valued version is unsupported → `""`. No code branches for these — the reader degrades them; test rows pin the behavior.
- **`t.Parallel()` in tests that mutate `detectors`**: AGENTS.md global-state rule; reader tests (pure) and detector tests (t.TempDir only) MAY parallelize; registry-touching tests must not [VERIFIED: project_probe/registry_test.go:12-14].
- **Fabricating a Name when `[project]` lacks one**: PEP 621 requires static name, but malformed files exist — folder-base fallback (DATA-02) is the contract, never a synthesized name.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| BOM removal | A leading-byte state machine | `readManifest`'s `bytes.TrimPrefix(content, utf8BOM)` [VERIFIED: project_probe/manifest.go:15,50] | Phase 10 built it; the reader inherits it — a BOM'd pyproject.toml must parse (SC1) |
| File-size caps / FIFO hang protection | Per-detector read loops / re-opening with plain `os.Open` | `readManifest` (1 MB LimitReader + O_NONBLOCK + `Mode().IsRegular()` gate) [VERIFIED: project_probe/manifest.go:29-43] | Phase 10/11 built it; the WR-01 FIFO DoS is already closed — do not regress it |
| README description fallback | Re-extracting first paragraphs | `readmeDescription(folder)` [VERIFIED: project_probe/readme.go:29-48] | Phase 11 built it (badges/TOC/underline heuristics); detectors call it when description is empty |
| Folder-name fallback | Hand-rolled last-segment extraction | `filepath.Base` | Stdlib; Windows-safe (ROBT-04) |
| Panic containment | Trusting the parser not to panic | `callDetector` recover → non-match [VERIFIED: project_probe/registry.go:44-53] | Phase 10 built it; the reader is designed never to panic, and the net stays behind it |
| Test assertions | Hand-rolled compare helpers | `testify` `assert`/`require` | Repo mandate (AGENTS.md); `testifylint` enforced |
| TOML parsing (the one thing that IS hand-rolled) | Promoting `pelletier/go-toml/v2` | The ~100-line subset reader | FRAM-02 defers the dep to v2; stdlib-only (ROBT-04); we read 3 keys per section — a full parser is overkill |

**Key insight:** everything "hard" in this phase — BOM, caps, FIFO DoS, README heuristics, panic safety, cross-platform paths — was already solved in Phases 10-11. The genuinely new code is one ~100-line reader and two ~25-line detectors; every risk lives in the reader's strict-degrade classification, which is why the reader test matrix (P1-P6 below) is the densest table in the phase.

## Common Pitfalls

### Pitfall 1: `#` inside quoted values truncates the value
**What goes wrong:** `description = "hello # world"` yields `hello` (or the line is misparsed) when comments are stripped before value extraction.
**Why it happens:** TOML's comment rule is "except when inside a string" [VERIFIED: toml.io/en/v1.0.0 §Comment] — a naive `strings.Index(line, "#")` strip violates it.
**How to avoid:** Extract the quoted value first (scan to the closing quote), then ignore everything after it. Pin with rows: `description = "hello # world"` → `hello # world`; `name = "x" # trailing` → `x`.
**Warning signs:** A probe returns a Description truncated at `#`.

### Pitfall 2: Sub-table key leakage into the parent section
**What goes wrong:** `[project.optional-dependencies]` keys (e.g. `test = ["pytest"]`) or `[tool.poetry.dependencies]` entries (`requests = "^2.13.0"`) appear in the `[project]`/`[tool.poetry]` read.
**Why it happens:** Prefix/suffix matching instead of exact header equality; or failing to switch the current section when a sub-table header appears.
**How to avoid:** Exact string equality on the header name; ANY `[`-leading line (including `[[…]]`) switches the section. Pin rows: `[project]` followed by `[project.scripts]` with `name = "evil"` inside → `name` stays the `[project]` value.
**Warning signs:** A pyproject with `[project.optional-dependencies]` reports a wrong Version.

### Pitfall 3: Multi-line strings / arrays / inline tables fabricate fake keyvals
**What goes wrong:** A `"""…"""` description (or a multi-line array/inline table) containing a line like `name = "evil"` or `[project]` overwrites real fields — "partial data" the SC4 contract forbids.
**Why it happens:** The reader treats every non-header line as a keyval; it doesn't know it is inside a string/array/table body.
**How to avoid:** The global skip states (Pattern 1 point 6): on `"""`/`'''`/`[`/`{` values, skip lines until the closing delimiter appears. Pin rows: multi-line string containing `name = "evil"` → name unchanged/empty; multi-line `authors = [` block in `[package]` → no fabricated fields.
**Warning signs:** SC4's "degrades strictly to empty fields" fails on a fixture with a multi-line string.

### Pitfall 4: Arrays and unquoted values inside the TARGET section break state
**What goes wrong:** `[package]` legitimately contains `authors = ["Alice"]`, `keywords = ["a","b"]`, `publish = false` — naive parsers either error, panic, or misparse subsequent lines.
**Why it happens:** Arrays are common in `[package]` (and `[tool.poetry]` has `packages = [...]`, `scripts = {...}`); they are legal-but-unsupported per D-02.
**How to avoid:** The value classifier handles `[`/`{`/unquoted per-key (not stored) and the one-line array case needs no state (closing bracket on the same line). Pin rows: `[package]` with `authors = ["a", "b"]` then `version = "1.2.3"` → Version intact, authors absent.
**Warning signs:** A Cargo.toml with authors fails to report version.

### Pitfall 5: CRLF and BOM regression
**What goes wrong:** Windows-edited manifests with `\r\n` or a UTF-8 BOM yield empty fields or non-matches.
**Why it happens:** CRLF is legal TOML ("Newline means LF or CRLF"); the BOM is stripped by `readManifest` but only if the reader consumes its output.
**How to avoid:** `TrimSpace` per line (handles `\r`); always route content through `readManifest` (never `os.ReadFile` in detectors). Pin rows: CRLF file parses; `\xEF\xBB\xBF`-prefixed pyproject.toml parses (mirror Phase 11 BOM rows).
**Warning signs:** A probe of a CRLF fixture returns LanguageUnknown.

### Pitfall 6: `version.workspace = true` handled instead of degraded
**What goes wrong:** A workspace member's version gets resolved from the workspace root, or `true` is reported as the Version, or a dotted-key parse panics.
**Why it happens:** Workspace inheritance is the modern Cargo norm (MSRV 1.64+) [CITED: doc.rust-lang.org/cargo/reference/workspaces.html]; the dotted key `version.workspace` tempts special handling.
**How to avoid:** The reader classifies dotted keys as unsupported (Pattern 1 point 4) → `""`; D-07 forbids workspace-root resolution. Pin rows: `version.workspace = true` → Version `""`; `description.workspace = true` → `""` → README fallback fires.
**Warning signs:** SC3 fails — Version is `true`, `0.0.0`, or resolved from a sibling Cargo.toml.

### Pitfall 7: `make coverage-quick` red before the first commit (pre-existing)
**What goes wrong:** The mandatory pre-commit gate fails on `release/update.go 68.9%` vs 70% file threshold — verified live this session.
**Why it happens:** Pre-existing, unrelated to project_probe (logged in deferred-items); CI's coverage job uses `.testcoverage.yml` with identical thresholds [VERIFIED: .testcoverage.yml].
**How to avoid:** Run `make coverage-quick`, assert the **project_probe rows pass** (pkg ≥80% / file ≥70% — new files toml.go/detect_python.go/detect_rust.go must each clear 70%) and total ≥75%, and treat the `release/update.go` row as the documented known-red. Do NOT fix release/update.go (scope boundary).
**Warning signs:** A plan verification step fails with no remediation path — encode the known-red exception explicitly.

### Pitfall 8: Lint is not a gate — and golangci-lint is broken under the default toolchain
**What goes wrong:** An executor runs `golangci-lint run` and hits a typecheck panic on the go1.27.0 stdlib (`math/rand/v2` — "method must have no type parameters"), or runs `GOTOOLCHAIN=go1.26.4 golangci-lint run` and sees 86 pre-existing issues in Phase 11 files (gofmt/golines diffs from go1.26-formatters) — either way the "gate" fails and blocks commits.
**Why it happens:** golangci-lint v2.12.2 is built with go1.26.3 and cannot analyze go1.27.0's stdlib; the reverse toolchain pins the formatters to go1.26 rules that disagree with the go1.27 `gofmt` hook that formatted Phase 11 files.
**How to avoid:** Verified this session: CI (`.github/workflows/go.yml`) runs **no lint job** — only `go test -v ./...` + coverage; pre-commit hooks are **not installed** (no `.git/hooks/pre-commit`). Lint is advisory: write lint-clean code per CONVENTIONS (cyclop ≤10, wsl_v5, testifylint) but never treat golangci-lint output as a commit blocker. If a contributor wants a sanity check, `GOTOOLCHAIN=go1.26.4 golangci-lint run --new-from-rev=HEAD~1 ./project_probe/...` scopes to new issues only.
**Warning signs:** A plan's verification section demands a green `golangci-lint run` — that is unachievable in this repo today.

### Pitfall 9: Registry position assertions drift mid-phase
**What goes wrong:** `TestDetectorPositions` [VERIFIED: project_probe/registry_test.go:70-79] asserts `detectors[1] == nil` and `detectors[4] == nil` — both flips are required, in the right plans, or the suite breaks.
**Why it happens:** The test pins the Phase-11 interim state; filling slot 1 (P02) without updating the test fails CI; updating both flips in P02 breaks P03's RED step.
**How to avoid:** Interim update in P02 (`detectors[1]` NotNil, `detectors[4]` still Nil), final in P03 (both NotNil). Each plan's RED test is the flip itself.
**Warning signs:** A plan's verification says "all tests pass" while the registry is mid-fill.

## Code Examples

Verified patterns from official sources plus this session's probes:

### TOML value classification — the strict-degrade decision table (Pattern 1 mechanics)
```
Input line (trimmed) in target section        Result
name = "acme"                                 name → acme
name = 'acme'                                 name → acme
name = "acme" # comment                       name → acme
name = "a=b"                                  name → a=b        (= inside value)
description = "a \"b\" c"                     description → a \"b\" c   (raw, no unescape, D-disc-5)
name = "foo" garbage                          NOT stored         (D-disc-6 never-partial)
version.workspace = true                      NOT stored         (dotted key → "", D-07)
version = """multi                            NOT stored + skip state
version = { file = "VERSION" }                NOT stored         (inline table)
dynamic = ["version"]                         NOT stored         (array)
publish = false                               NOT stored         (unquoted value)
key =                                         NOT stored         (unspecified value, invalid)
```

### Registry position test — final state (after P03)
```go
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
	assert.Len(t, detectors, 7)
	assert.NotNil(t, detectors[0]) // Go
	assert.NotNil(t, detectors[1]) // Python — Phase 12 (was Nil)
	assert.Nil(t, detectors[2])    // C#/.NET — Phase 13
	assert.NotNil(t, detectors[3]) // JS/TS
	assert.NotNil(t, detectors[4]) // Rust — Phase 12 (was Nil)
	assert.Nil(t, detectors[5])    // Java/Kotlin — Phase 13
	assert.NotNil(t, detectors[6]) // PHP
}
```

### Cascade precedence rows to add (probe-level)
```go
// In TestProbe_CascadePrecedence [VERIFIED: project_probe/detect_php_test.go:167-172] or a new test:
// 1. pyproject.toml + Cargo.toml → LanguagePython  (index 1 beats index 4, D-08 order)
// 2. broken package.json + valid pyproject.toml → LanguagePython
//    (index 1 beats 3 — Python matches on presence, D-09; JS needs parse success, D-04)
```

### Reader edge-case matrix (toml_test.go rows — pure []byte, no files)
- `[project]` full → name/version/description all extracted
- section missing → empty map (D-03)
- `[tool.poetry]` keys not visible to a `project` read
- `[project]` then `[project.optional-dependencies]` then keyvals → sub-table keys excluded (P2)
- `[project] # comment` header with trailing comment → matches (D-disc-8)
- `[ project ]` padded header → matches
- `[[tool.poetry.source]]` → array-of-tables header does NOT match `tool.poetry` read
- root-table keyvals before any header → excluded
- CRLF file → parses; BOM-prefixed content → parses (P5)
- multiline string containing `name = "evil"` + `[project]` → no fabrication (P3)
- `authors = [` multi-line array in `[package]` → later `version = "1.2.3"` still read (P4)
- `version = {` newline inline table (TOML 1.1 form) → skip state (P3)
- `name = "foo" garbage` → not stored (D-disc-6)
- duplicate `name` keys → last wins (A5)
- empty file, comment-only file, whitespace-only file → empty map
- 200-key section → all returned (no limit)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Poetry 1.x `[tool.poetry]` metadata only | PEP 621 `[project]` (Poetry 2.x, Sept 2024) with legacy fields deprecated | 2024 (Poetry 2.0) | Both-present files exist in the wild during migration; whole-section precedence (`[project]` wins) is the ecosystem-correct rule [CITED: python-poetry.org CHANGELOG #9135] |
| Cargo members duplicating version/description | Workspace inheritance (`version.workspace = true`, MSRV 1.64+) | Rust 1.64 (2022) | Modern workspace members have NO literal version/description — empty Version is the norm for them, not an anomaly [CITED: doc.rust-lang.org/cargo/reference/workspaces.html] |
| TOML 1.0.0 (2021) | TOML 1.1.0 (2025) — newlines in inline tables, hex floats, etc. | 2025 | Our subset reads only single-line quoted scalars; the 1.1 newline-in-inline-table form is handled by the bracket-skip state (P3) — no behavior change needed |
| Detectors 3/7 live (Phase 11) | 5/7 live (this phase) | Phase 12 | Python beats JS/TS and Rust in mixed-repo cascades; C#/.NET and Java slots remain for Phase 13 |

**Deprecated/outdated:**
- `[tool.poetry]` name/version/description — deprecated by Poetry 2.x in favor of `[project]`; we keep the legacy read as the D-04 fallback (still millions of Poetry 1.x repos).
- Hand-resolving workspace-inherited versions — never done (D-07); empty is correct.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Whole-section precedence (`[project]` non-empty map wins; `[tool.poetry]` only when `[project]` yields nothing) is the correct D-04/D-05 reading; "when both sections lack a name" = after the section pick, per-field fallbacks | Pattern 2 / D-disc-3 | If the user intended per-field cross-section fallback (name from poetry when project lacks it), behavior differs only for files where `[project]` exists without a name — rare (PEP 621 requires static name) and easy to adjust (one test row) |
| A2 | A trailing `# comment` after a section header is accepted as a match (leniency beyond the spec grammar) | Pattern 1 | Harmless: accepting a form a strict parser rejects violates no contract; files with header comments are common in the wild |
| A3 | Quoted header segments (`[tool."poetry"]`) and inner-dot whitespace (`[tool . poetry]`) are documented non-matches | Pattern 1 | Frequency ~0; such files degrade to empty fields with presence-match still firing (Language correct, Name = folder base) |
| A4 | Basic-string values are extracted raw (outer quotes stripped, interior escapes kept verbatim) | Pattern 1 / D-disc-5 | A `\"` inside a description stays `\"` — cosmetic; matches Phase 11's go.mod quote leniency precedent |
| A5 | Duplicate keys last-wins (invalid TOML, harmless leniency) | Pattern 1 | Invalid files only; mirrors Phase 11 JSON duplicate-keys behavior |
| A6 | `version = { file = "…" }` (table-valued version) degrades to `""` — non-spec form handled by the inline-table rule | Pattern 1 / Code Examples | PEP 621 says version is a string, but table-valued versions appear in the wild; degrade is the safe reading of D-02 |
| A7 | Multi-line string / array / inline-table skip states must be global (not section-scoped) | Pattern 1 / D-disc-4 | If the user wanted the 80-line budget over correctness, fake-keyval fabrication inside string bodies would be accepted — this is the SC4 "no partial data" reading |
| A8 | `detect_python.go`/`detect_rust.go`/`toml.go` filename suffixes are build-safe | Project Structure | Verified via `go tool dist list` this session — `python`/`rust` appear in neither GOOS nor GOARCH lists (unlike `_js`) |
| A9 | Lint (golangci-lint) is advisory, not a gate: CI runs no lint job; pre-commit hooks are not installed | Pitfall 8 | If a contributor installs pre-commit hooks and CI gains a lint job, Phase 11's files already fail it — a future cleanup task, not this phase's blocker |

## Open Questions (RESOLVED)

1. **Section precedence interpretation (A1) — confirm whole-section, not per-field, fallback.**
   - What we know: D-04 says legacy fallback "when `[project]` absent"; D-05 says "name → folder-base fallback (DATA-02) when both sections lack a name". The reader returns a map per section; `len(fields) == 0` cleanly covers absent-and-empty.
   - What's unclear: whether "both sections lack a name" implies name may be looked up in `[tool.poetry]` even when `[project]` exists but lacks the name.
   - Recommendation: whole-section precedence (D-disc-3) — deterministic, matches D-04's wording, and Poetry 2.x migration files carry both sections with deprecated legacy fields that must not mix. The planner should note this choice in the plan; no user confirmation strictly required (within discretion), but it is the one interpretation risk worth recording.
   - **RESOLVED — adopted by plan 12-02** (whole-section precedence D-disc-3; pinned by TestProbe_PythonProjectWins + TestProbe_PythonEmptyProjectFallsToPoetry).

2. **`[workspace]`-only virtual manifests — confirm folder-base Name is desired.**
   - What we know: D-09 locks presence-match for detectRust; a virtual manifest has no `[package]`, so Name = folder base, Version = "", Description = README fallback.
   - What's unclear: nothing technically — this is the natural consequence of the locked decisions; recorded for visibility.
   - Recommendation: no action; pin with a test row so the behavior is explicit.
   - **RESOLVED — adopted by plan 12-03** (TestProbe_RustVirtualManifest row; D-09 presence-match yields folder-base Name).

3. **Reader size budget (~80 lines) vs the skip states.**
   - What we know: the strict-degrade contract (SC4) needs global skip states (P3); with doc comments the reader lands ~100-110 lines.
   - What's unclear: whether the CONTEXT's "~80 lines" is a hard budget.
   - Recommendation: treat as soft — correctness first (A7); the skip states are ~15 lines. Note the deviation in the plan if it matters.
   - **RESOLVED — adopted by plan 12-01** (soft-budget flagged assumption in the prohibitions; ~100-110 lines with doc comments documented).

4. **golangci-lint enforcement (A9) — confirm advisory status.**
   - What we know: verified this session — CI (`go.yml`) has no lint job; `.git/hooks/pre-commit` does not exist; `golangci-lint run` typecheck-panics under the default go1.27.0 toolchain.
   - What's unclear: whether the user wants lint enforced via some other path (e.g. a Makefile target) in this phase.
   - Recommendation: keep advisory; the plan's verification uses `go test ./...` + `make coverage-quick` (project_probe rows) as gates, matching Phase 11's effective practice.
   - **RESOLVED — adopted by plans 12-01/12-02/12-03** (lint stays advisory; verification gates are `go test ./...` + `make coverage-quick` project_probe rows).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All code/tests | ✓ | go1.27.0 local (module declares 1.26.4; CI pins via go-version-file) | Toolchain auto-download per go.mod |
| `make` | `make coverage-quick` gate | ✓ | GNU Make 3.81 | — |
| `go-test-coverage` | Coverage gate | ✓ | installed (~/go/bin) | `make check-go-test-coverage` auto-installs |
| `golangci-lint` | (advisory only) | ⚠️ broken under default toolchain | v2.12.2 (built go1.26.3) | Not a gate — CI runs no lint job; pre-commit hooks not installed (Pitfall 8) |
| `pre-commit` | Commit hooks | ✗ not installed | — | Hooks don't run locally; CI enforces tests + coverage only |
| Coverage gate health | Every commit | ⚠️ **known-red** | `make coverage-quick` fails on `release/update.go` 68.9% vs 70% (pre-existing, deferred) | Assert project_probe rows (pkg ≥80%, file ≥70% incl. the 3 new files) + total ≥75% only; document the known-red (Pitfall 7) |
| 3-OS CI matrix | Cross-platform guarantees | ✓ (GitHub Actions) | — | Local darwin; no new platform-specific code in this phase (stdlib only, no syscalls) |

**Missing dependencies with no fallback:** none — stdlib-only phase; lint is advisory (Pitfall 8).
**Missing dependencies with fallback:** pre-commit hooks (not installed) — CI covers tests/coverage; lint is advisory.

## Validation Architecture

> **SKIPPED** — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly; per the research contract this section is omitted. The repo's standard gates (`go test ./...`, `make coverage-quick`, `testify` assertions) apply as documented in Project Constraints.

## Security Domain

> Required — `workflow.security_enforcement: true` in .planning/config.json. ASVS level 1; block on HIGH.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no users/sessions) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (OS permissions surface via existing sentinels) |
| V5 Input Validation | yes | All manifest content is untrusted input: `readManifest` 1 MB cap + BOM strip + FIFO gate [VERIFIED: project_probe/manifest.go:13,29-43]; `readTOMLSection` is a bounds-checked string scanner that never panics and never stores partial data (D-02); `callDetector` recover remains the outer net [VERIFIED: project_probe/registry.go:44-53] |
| V6 Cryptography | no | — |

### Known Threat Patterns for {stdlib TOML-subset parsing}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| FIFO/special file at a manifest path hangs the probe | DoS | Already closed — `readManifest` uses O_NONBLOCK + `Mode().IsRegular()` gate (WR-01 landed Phase 11) [VERIFIED: project_probe/manifest.go:29-43] |
| Pathological TOML content causing parser panics (e.g. 1 MB of `[`/quotes) | DoS | Bounds-checked scanning; skip states; `callDetector` recover → non-match [VERIFIED: project_probe/registry.go:44-53]; reader tests include adversarial rows (unterminated strings, bracket floods) |
| Oversized manifest memory exhaustion | DoS | `readManifest` LimitReader cap inherited by all manifest reads |
| Malformed manifest spoofing a different language (broken pyproject.toml + valid Cargo.toml) | Spoofing | Presence-match for Python/Rust (D-09) — a garbage pyproject.toml claims Python at index 1 before Rust at 4; the ecosystem-correct reading (the file IS the marker), pinned by cascade rows |
| Workspace-root traversal to resolve `version.workspace` | Tampering | Forbidden by D-07; root-scoped only (ROBT-05) — no parent-directory reads anywhere in the reader/detectors |
| Symlink-following / network / build-tool execution | Tampering | ROBT-05 anti-features — none introduced; stdlib-only imports; no `EvalSymlinks`, no `os/exec`, no `net/http` (Phase 10 prohibitions stand) |

## Project Constraints (from AGENTS.md)

Directives the planner must honor (verbatim intent from `./AGENTS.md`):

- **Milestone branches:** every milestone gets its own git branch for the PR flow; work on the milestone branch, then open a PR to `main`. Branch name: `gsd/v1.7-{slug}` (current branch: `gsd/v1.7-project-probe`).
- **Before every commit:** run `make coverage-quick` — enforces thresholds in `.testcoverage-quick.yml` (packages ≥80%, files ≥70%, total ≥75%). **Known-red exception:** the gate currently fails on `release/update.go` (68.9% vs 70%, pre-existing, logged in deferred-items.md) — assert project_probe rows + total instead and document the known-red (Pitfall 7). Do not commit if *project_probe* fails.
- **Before pushing a tag:** regenerate and stage the quality report (`make quality-report`, `git add quality-report.md`).
- **Cross-platform testing (CI runs Linux/macOS/Windows):** tests modifying global state (the detector registry slice) must NOT use `t.Parallel()`; the new reader/detector tests use only `t.TempDir()` folders or pure `[]byte` and MAY parallelize; no OS-specific file-mode assertions.
- **Code style:** standard Go idioms; focused single-purpose functions; `testify` for assertions; unit + edge-case tests; Go doc comments for all exported symbols; Conventional Commits; minimal external dependencies — prefer stdlib.
- **Spike findings:** consult `Skill("spike-findings-go")` before committing (auto-loaded during implementation; its panic-recovery precedent is the `callDetector` net this phase's parser sits behind — the reader must be written so the net is never needed).

## Sources

### Primary (HIGH confidence)
- [VERIFIED: toml.io/en/v1.0.0 (fetched directly this session)] — TOML 1.0.0 grammar: whitespace/newline rules, comment rule ("except when inside a string"), bare/quoted/dotted keys, basic/literal/multi-line strings and escapes, arrays (multi-line, trailing commas), table headers (`[a.b.c]`, quoted segments, whitespace around keys), array-of-tables `[[…]]`, inline tables (single-line in 1.0), "arrays are only ever values", unspecified values invalid, keyval same-line rule
- [VERIFIED (in-repo): project_probe/registry.go:3-53] — `detectorFunc` contract, nil-slot literal with Python@1/Rust@4 placeholders, nil-skip, `callDetector` recover
- [VERIFIED (in-repo): project_probe/manifest.go:13,15,29-43] — `maxManifestSize = 1 << 20`, `utf8BOM`, O_NONBLOCK + `IsRegular()` WR-01 gate
- [VERIFIED (in-repo): project_probe/project.go:32-33,41-42] — `LanguagePython Language = "Python"`, `LanguageRust Language = "Rust"`
- [VERIFIED (in-repo): project_probe/readme.go:29-48] — `readmeDescription` exact-case candidate logic
- [VERIFIED (in-repo): project_probe/json.go:11-17] — `readJSONManifest` shape (readManifest wrapper precedent)
- [VERIFIED (in-repo): project_probe/registry_test.go:70-79] — `TestDetectorPositions` current asserts (indices 1/4 Nil — must flip)
- [VERIFIED (in-repo): project_probe/detect_php_test.go:167-172] — `TestProbe_CascadePrecedence` location for new Python/Rust rows
- [VERIFIED (in-repo): project_probe/doc.go:35-46] — stale detector-list paragraph to refresh
- [VERIFIED (in-repo): .github/workflows/go.yml] — CI runs `go test -v ./...` + coverage only; **no lint job**
- [VERIFIED (in-repo): .git/hooks/] — pre-commit hooks not installed (no pre-commit file)
- [VERIFIED (in-repo): .testcoverage-quick.yml + .testcoverage.yml] — identical thresholds (file 70 / package 80 / total 75)
- [VERIFIED (live this session)] — `go tool dist list`: no `python`/`rust` in GOOS or GOARCH lists (filename safety); `make coverage-quick` red on `release/update.go` 68.9% only; 96 tests pass in project_probe; golangci-lint 2.12.2 typecheck-panics on go1.27 stdlib / flags Phase 11 files under GOTOOLCHAIN=go1.26.4

### Secondary (MEDIUM confidence)
- [CITED: packaging.python.org/specifications/pyproject-toml] — `[project]` spec: name statically required; version static-or-dynamic; description = one-line Summary string; allowed keys list
- [CITED: peps.python.org/pep-0621] — PEP 621 historical spec: table-name rule, name/version/description formats, dynamic escape hatch
- [CITED: python-poetry.org/docs/1.8/pyproject] — `[tool.poetry]` name/version/description plain strings, required in package mode; dependencies tables
- [CITED: python-poetry.org CHANGELOG (#9135) + poetry-plugin-migrate docs] — Poetry 2.x `[project]` support and `[tool.poetry]` field deprecation; migration mapping tool.poetry.name → project.name etc.
- [CITED: doc.rust-lang.org/cargo/reference/manifest.html] — `[package]` section: name only required field; version/description semantics; first-section convention
- [CITED: doc.rust-lang.org/cargo/reference/workspaces.html] — workspace inheritance `{key}.workspace = true` (MSRV 1.64+), `[workspace.package]`, virtual manifests without `[package]`

### Tertiary (LOW confidence)
- [ASSUMED] — TOML 1.1.0 release timing (2025) and inline-table newline change (bracket-skip state covers it regardless; cited only for State of the Art)
- [ASSUMED] — prevalence of table-valued `version = {file = …}` in the wild (degrade behavior is D-02-mandated either way; A6)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib-only mandate; zero new dependencies; every consumed helper verified in-repo with line citations
- Architecture: HIGH — reader design grounded in the official TOML 1.0.0 spec (fetched this session); detector shapes grounded in official ecosystem specs; registry edit is a literal slot fill; two interpretation assumptions (A1, A7) flagged honestly
- Pitfalls: HIGH — coverage known-red and lint breakage reproduced live this session; strict-degrade matrix derived from the spec's grammar rules

**Research date:** 2026-09-29
**Valid until:** 2026-10-29 (TOML 1.0.0 grammar and PEP 621/Cargo specs stable; the lint/coverage environment facts are ours to re-verify)