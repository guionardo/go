# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback - Research

**Researched:** 2026-09-29
**Domain:** Line-oriented go.mod parsing, JSON manifest decoding (package.json / composer.json), README first-paragraph extraction heuristics, ordered-registry extension
**Confidence:** HIGH

## Summary

Phase 11 fills three of the seven fixed positions in the Phase 10 `projectprobe` registry — Go (index 0), JS/TS (index 3), PHP (index 6) — plus the DATA-02 name chain (manifest → folder base) and the DATA-04 description chain (manifest → README first paragraph → empty). All parsing is stdlib-only hand-rolled code (`bytes`, `strings`, `encoding/json`), matching the ROBT-03 TOML-subset-reader precedent: no `golang.org/x/mod/modfile`, no TOML dependency, nothing new in go.mod. The detectors are pure functions with the locked `(ProjectData, bool)` contract; `readManifest` (1 MB cap + BOM strip) does all I/O.

Three verified facts drive the design. (1) The **go.mod grammar** (go.dev/ref/mod) is line-oriented with `//`-only comments; `module` values may be **double-quoted** (verified live: `module "example.com/dq"` parses; backtick form is rejected by the go tool); `go` directive versions may be `1.21`, `1.21rc1`, or `1.23.0` (all verified live) but **quoted** `go "1.22.5"` is rejected; trailing `//` comments on directive lines are legal (verified live); a UTF-8 BOM is **rejected by the go tool** (verified live: `unexpected input character '\ufeff'`) — so `readManifest`'s BOM strip makes our detector *more lenient* than the go tool, exactly as success criterion 1 requires. (2) **`encoding/json` behavior** (verified live on go1.27.0): `Unmarshal` rejects a leading UTF-8 BOM (`invalid character '\ufeff' looking for beginning of value`) — the BOM strip is *load-bearing* for JSON manifests too; malformed JSON and type mismatches return errors (detector → `false`, cascade continues per D-04); unknown fields are ignored; duplicate keys last-wins; `"private": true` without `version` yields `""` (D-06 needs no `Private` field — a test row pins it); `"description": null` yields `""` and correctly triggers the README fallback. (3) **Composer schema** (getcomposer.org/doc/04-schema.md): `name` is the full `vendor/package` string (D-07), `version` is *usually absent* (Packagist infers it from VCS tags) — empty Version is the common case, never fabricated; `description` is required for published libraries but absent in many project packages — README fallback is the common case.

**Primary recommendation:** Registry = literal with `nil` placeholders at the research-locked positions (`detectGo, nil, nil, detectJS, nil, nil, detectPHP`) plus a 2-line nil-skip in `runDetectors` — positions stay visible and Phase 12/13 fill slots 1/4 and 2/5 without reordering. Go detector matches on **go.mod presence** (D-01/D-02 frame both fields as independently optional; DATA-02's fallback exists for the missing-name case); JS/PHP match on **parse success** (D-04 is explicit for JS; PHP mirrors it). README fallback is one shared helper (`readmeDescription(folder) string`) consumed by all three detectors only when the manifest description is empty. Test fixtures are inline `t.TempDir()` + `os.WriteFile` (repo convention — no `testdata/` dirs exist in this repo). Plan structure mirrors Phase 10: P01 = README helper, P02 = Go detector + registry slots, P03 = JS/PHP detectors — each plan RED-tests-first (verified failing, uncommitted; RED evidence in the feat commit message). Two inherited operational facts the planner MUST handle: `make coverage-quick` is red on `release/update.go` 68.9% (pre-existing, logged in deferred-items.md — verify only project_probe rows + total), and `doc.go`'s "registry is empty in this phase" paragraph goes stale and needs a one-line update.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Go Detector (DETC-02)
- **D-01:** Name = go.mod `module` line verbatim (full module path, e.g. `github.com/guionardo/go` — not the last path segment). Quoted module lines parse correctly (strip quotes); BOM already stripped by `readManifest`.
- **D-02:** Version = `go` directive raw string (e.g. `1.26.4`), documented as **toolchain floor, not release version** — matches RESEARCH decision carried from STATE.md; empty when no `go` directive. Never fabricated, never normalized (DATA-03).
- **D-03:** go.mod has no description field → description comes from README fallback (DATA-04) or empty.

#### JS/TS Detector (DETC-03)
- **D-04:** package.json name/version/description via `encoding/json`; malformed JSON → detector returns false (next detector in cascade; per D-03 never-fail, degrade-to-Unknown).
- **D-05:** Language is ALWAYS `LanguageJavaScript` for package.json — the distinct `typescript` value is v2 backlog (REFN-01), NOT this phase.
- **D-06:** `private: true` with no version → empty Version, never fabricated.

#### PHP Detector (DETC-04)
- **D-07:** composer.json name/version/description; Name = full `vendor/package` string (ecosystem identifier — consistent with Go module-path reporting, not the last segment).

#### README Fallback (DATA-04)
- **D-08:** Ordered deterministic candidate list: `README.md` → `README.rst` → `README` (exact-case, first existing wins). Only when the manifest has no description.
- **D-09:** "First real paragraph" extraction skips: image-only badge lines (`[![...](...)`), TOC-style link lists (leading `- [` or numbered link lines at the top), rst underline headings (`===`/`---` underlines), and blank lines — then takes the first consecutive non-blank text block.
- **D-10:** No README → empty description (never an error).

#### Registry Integration
- **D-11:** Detectors register in the research-locked order (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) — this phase fills Go, JS/TS, PHP entries at their fixed positions; Python/C#/.NET/Rust/Java slots stay empty until phases 12-13.
- **D-12:** Each detector calls `readManifest(folder, name)` for its manifest; failure degrades to `(ProjectData{}, false)` — never-fail contract (D-03 from Phase 10).

### the agent's Discretion
- Exact go.mod line-parsing mechanics (strings within `module`/`go` lines), README content scoring thresholds (what counts as "badge line" vs text), test fixture layout. Follow repo conventions (CONVENTIONS.md) and the Phase 10 package patterns (PATTERNS.md).

### Deferred Ideas (OUT OF SCOPE)
- Distinct `typescript` Language value — v2 backlog (REFN-01), NOT this phase
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DETC-02 | Go detector — go.mod module → name, `go` directive → Version (documented as toolchain floor) | go.mod grammar verified against go.dev/ref/mod + live `go mod edit` probes: line-oriented, `//`-only comments, double-quoted module values legal (backtick rejected), `go 1.21`/`1.21rc1`/`1.23.0` all legal, trailing `//` comments legal, BOM rejected by the tool but stripped by readManifest; match rule = presence-based (see Discretion Decisions D-disc-1); parser = strings.Fields + comment strip + quote strip on the module value only (Pattern 2) |
| DETC-03 | JS/TS detector — package.json name/version/description | `encoding/json` verified live: Unmarshal rejects BOM/malformed/type-mismatch (→ `false`, D-04), ignores unknown fields, last-wins duplicates; `private:true` + no version → `""` (D-06); Language always `LanguageJavaScript` = `"JavaScript"` [VERIFIED: project_probe/project.go:36-37] |
| DETC-04 | PHP detector — composer.json name/version/description | Composer schema verified (getcomposer.org/doc/04-schema.md): name = `vendor/package` (regex `^[a-z0-9]([_.-]?[a-z0-9]+)*/[a-z0-9](([_.]|-{1,2})?[a-z0-9]+)*$`), version usually absent (Packagist infers from tags → empty Version is the norm), description required for libraries but absent in project packages → README fallback common; Language `LanguagePHP` = `"PHP"` [VERIFIED: project_probe/project.go:48-49] |
| DATA-02 | Name chain: manifest name → folder base | Fallback = `filepath.Base(folder)` when the manifest yields no name; applies to all three detectors (Go: missing/unparseable module line; JS/PHP: missing or empty-string name). Fold in the folder-Base edge rows (`sub/dir` → `dir`) |
| DATA-04 | Description: manifest → README first-paragraph → empty | Shared `readmeDescription(folder) string` helper (Pattern 4): candidates `README.md` → `README.rst` → `README` exact-case (D-08), first-real-paragraph extraction (D-09) with RST-verified underline rules; empty when no README (D-10). Heuristic thresholds per D-disc-3/D-disc-4 |

Note: DATA-03 (version semantics — the "toolchain floor" decision record) and ROBT-05 (anti-features audit) belong to Phase 14 per REQUIREMENTS.md traceability — Phase 11 implements the *behavior* (raw strings, empty when absent) but the decision record lands in Phase 14.
</phase_requirements>

## Architectural Responsibility Map

This is a single-package stdlib library — no browser/SSR/CDN/database tiers exist. Every capability belongs to the package core; the map prevents the planner from pushing detection logic into tests, docs, or other packages.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| go.mod line parsing | API/Backend (package core) | — | `detect_go.go` — stdlib-only hand-rolled parser (ROBT-03 precedent); must live in the package, not in `cmd/` or tests |
| JSON manifest decoding | API/Backend (package core) | — | `detect_js.go` / `detect_php.go` + shared `readJSONManifest` — D-04/D-12 never-fail semantics |
| README first-paragraph extraction | API/Backend (package core) | — | `readme.go` shared helper — consumed by all three detectors (D-08..D-10) |
| Registry position ownership | API/Backend (package core) | — | `registry.go` literal edit — nil placeholders at locked positions (D-11); Phase 12/13 fill them |
| Manifest I/O safety | API/Backend (package core) | — | Existing `readManifest` [VERIFIED: project_probe/manifest.go:19-35] — optional WR-01 regular-file gate (see Open Questions OQ-1) |
| Docs (never-fail contract) | Docs | — | `doc.go` stale "registry is empty" paragraph must be updated to name the three live detectors |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `strings` | 1.26.4 (go.mod) | Line splitting, `TrimSpace`, `HasPrefix`, `Fields` for go.mod directives | Hand-rolled line parsing (ROBT-03 pattern); verified grammar needs no regex |
| Go stdlib `encoding/json` | 1.26.4 | package.json / composer.json decode | D-04 locks it; behavior verified live (BOM/malformed/unknown-field/duplicate-key table) |
| Go stdlib `bytes` | 1.26.4 | Line iteration over `[]byte` content (`bytes.Split`) | readManifest returns `[]byte`; avoids a string copy |
| Go stdlib `path/filepath` | 1.26.4 | `Base` (folder-name fallback, DATA-02) | ROBT-04 — never `path` |
| Go stdlib `regexp` | 1.26.4 | TOC numbered-link detection in README extraction | Optional — only if the `^\d+\. \[` rule needs a regex; `strings` may suffice (see Pattern 4) |
| `github.com/stretchr/testify` | v1.11.1 (test-only, existing) | `assert`/`require` | AGENTS.md mandate; already in go.mod [VERIFIED: go.mod:12] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `path_tools` (`DirExists`, `FileExists`) | in-repo | Existence pre-checks | NOT needed — `readManifest` already distinguishes missing-file from other failures via `(nil, false)` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled go.mod line parser | `golang.org/x/mod/modfile` (already an indirect dep) | modfile is the *authoritative* parser, but the stdlib-only mandate (ROBT-04 + STATE.md "minimal external dependencies") governs v1.7; promoting an indirect dep to direct for ~25 lines of parsing violates the constraint. modfile also *rejects* BOM'd files, which SC1 requires us to accept |
| One detector file per language | Single `detectors.go` | Repo convention is single-concern files (CONVENTIONS.md); three small files + shared helpers match the Phase 10 layout |
| README extraction via a markdown library | Heuristic line scanner | No stdlib markdown parser exists; external parsers (goldmark etc.) violate stdlib-only; the D-09 skip-list is precisely scoped — a full parser is overkill |
| Dense detector slice without nil slots | `[]detectorFunc{detectGo, detectJS, detectPHP}` | Loses the research-locked positions (D-11): Phase 12 would have to re-insert Python between Go and JS — reordering risk and churn. The nil-slot literal documents the full order in one place |

**Installation:** none — stdlib only; testify already present.
**Version verification:** `go.mod` declares `go 1.26.4` [VERIFIED: go.mod:3]; `github.com/stretchr/testify v1.11.1` [VERIFIED: go.mod:12]. Local toolchain go1.27.0 (darwin/arm64) — forward-compatible; CI pins via `go-version-file: go.mod`.

## Package Legitimacy Audit

> Gate outcome: **N/A — no external packages installed by this phase.** All runtime code is Go stdlib (`strings`, `bytes`, `encoding/json`, `regexp`, `path/filepath`); `testify` is an established module dependency used test-only. The planner must not add any `go get` task — the go ecosystem is not covered by the package-legitimacy seam, and the gate is satisfied by zero installs (Phase 10 precedent).

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
                                   │       detectGo,   // idx 0 (this phase)   │
                                   │       nil,        // idx 1 Python (Ph 12) │
                                   │       nil,        // idx 2 C#/.NET (Ph 13)│
                                   │       detectJS,   // idx 3 (this phase)   │
                                   │       nil,        // idx 4 Rust (Ph 12)   │
                                   │       nil,        // idx 5 Java (Ph 13)   │
                                   │       detectPHP,  // idx 6 (this phase)   │
                                   │   }                                      │
                                   │   nil entries skipped; callDetector      │
                                   │   recovers panics → non-match (D-03)     │
                                   └──────┬───────────┬───────────┬───────────┘
                                          ▼           ▼           ▼
                                    detectGo     detectJS     detectPHP
                                          │           │           │
                    ┌─────────────────────┘           │           └─────────────────────┐
                    ▼                                 ▼                                 ▼
        readManifest(folder,"go.mod")    readJSONManifest(folder,"package.json")  readJSONManifest(folder,"composer.json")
                    │  (1 MB cap + BOM    │  (readManifest + json.Unmarshal;      │  (same; malformed → false)
                    │   strip — Phase 10) │   malformed → false, D-04)            │
                    ▼                     ▼                                       ▼
        parseGoMod(lines):            name/version/description from JSON      name = full vendor/package (D-07)
        module line (quotes        │   Name empty → filepath.Base(folder)   │   Name empty → filepath.Base(folder)
        stripped, D-01)            │   (DATA-02)                            │   (DATA-02)
        go directive raw (D-02)    │   Version "" when absent (D-06)        │   Version "" when absent (common!)
        Description: none (D-03)   │   Description empty →                  │   Description empty →
                    │              │   readmeDescription(folder)            │   readmeDescription(folder)
                    │              └───────────────────┬────────────────────┘
                    │                                  ▼
                    │                      readmeDescription(folder)  [readme.go]
                    │                      candidates: README.md → README.rst → README (exact-case, D-08)
                    │                      firstRealParagraph(content): skip blank / badge / TOC /
                    │                      underline-heading pairs → first non-blank block (D-09)
                    │                      no README → "" (D-10)
                    └──────────────────────────────────►  ProjectData{Language, Name, Version, Description} merged
                                                          into Probe's defaults (A7 merge rule) — first match wins
```

Entry point: `Probe` (unchanged). Processing: existing Phase 10 pipeline → registry with nil slots → three new detectors → shared README helper. Branching: each detector either matches (merge) or falls through; malformed JSON / unreadable manifest / missing README all degrade, never error. External dependencies: filesystem only (ROBT-05 anti-features preserved — no network, no build tools, no symlink walks).

### Recommended Project Structure

```
project_probe/
├── registry.go        # MODIFIED: nil-slot literal (D-11) + nil-skip in runDetectors; comments updated
├── detect_go.go       # detectGo(folder) (ProjectData, bool) — presence-based go.mod match + parseGoMod
├── detect_js.go       # detectJS — readJSONManifest("package.json"), LanguageJavaScript (D-05)
├── detect_php.go      # detectPHP — readJSONManifest("composer.json"), LanguagePHP (D-07)
├── json.go            # readJSONManifest(folder, name, v) bool — readManifest + json.Unmarshal (shared)
├── readme.go          # readmeDescription(folder) string + firstRealParagraph(content) — D-08..D-10
├── doc.go             # MODIFIED: last paragraph — registry no longer empty; name the three live detectors
├── detect_go_test.go  # table-driven go.mod fixtures (inline t.TempDir + os.WriteFile)
├── detect_js_test.go  # JSON rows incl. private-no-version, scoped name, null description, BOM'd file
├── detect_php_test.go # vendor/package name rows, version-absent row
├── readme_test.go     # extraction matrix: badges, TOC, rst underlines, setext, multi-line paragraph, none
├── registry_test.go   # EXISTING — still passes (tests replace the slice wholesale); update the
│                      #   empty-registry test comment (registry is no longer empty in production)
├── manifest_test.go   # UNCHANGED unless OQ-1 (WR-01 regular-file gate) lands — then add FIFO row
└── example_test.go    # UNCHANGED
```

Conventions honored: snake_case files, `package projectprobe` internal tests for unexported access, `example_test.go` present, decorder order (type → const → var → func), no `t.Parallel()` in registry tests [VERIFIED: project_probe/registry_test.go:12-14].

### Pattern 1: Nil-slot registry literal + skip (D-11)

**What:** The Phase 10 literal `var detectors = []detectorFunc{}` [VERIFIED: project_probe/registry.go:11] becomes a fixed-length literal with `nil` placeholders so the research-locked order is visible in one place and Phase 12/13 fill slots without reordering:

```go
var detectors = []detectorFunc{
	detectGo,  // index 0 — Go (Phase 11)
	nil,       // index 1 — Python (Phase 12)
	nil,       // index 2 — C#/.NET (Phase 13)
	detectJS,  // index 3 — JavaScript/TypeScript (Phase 11)
	nil,       // index 4 — Rust (Phase 12)
	nil,       // index 5 — Java/Kotlin (Phase 13)
	detectPHP, // index 6 — PHP (Phase 11)
}
```

`runDetectors` [VERIFIED: project_probe/registry.go:15-23] gains a 2-line nil skip before `callDetector` — a nil entry must never reach the panic-recover path (it would panic on `fn(folder)`, be swallowed as a non-match, and spam the WR-02 log if that fix lands):

```go
for _, fn := range detectors {
	if fn == nil { // empty slot (Phase 12/13) — skip
		continue
	}
	if pd, ok := callDetector(fn, folder); ok {
		return pd, true
	}
}
```

**When to use:** This phase only — the positions are the contract (D-11). The existing registry tests survive unchanged because they replace the whole slice (save/restore pattern [VERIFIED: project_probe/registry_test.go:19-20,31-32,49-50,66-67]) — but `TestRunDetectors_EmptyRegistry`'s comment ("pins the Phase-10 default state") goes stale and should be updated; the test itself still passes (no real folder matches an empty cascade input `""`).

### Pattern 2: Hand-rolled go.mod line parser (DETC-02, D-01/D-02)

**What:** A minimal line scanner over the BOM-stripped content. Verified grammar rules it implements (go.dev/ref/mod §Lexical elements + live probes):

- Lines are `module`/`go`/… keyword + arguments; `//` comments run to end of line; `/* */` banned (go tool: *"mod files must use // comments"*).
- Value tokenization via `strings.Fields` (handles tabs, multiple spaces; quoted values come through as single tokens).
- **Module value may be double-quoted** (verified: `module "example.com/dq"` → Path `example.com/dq`); backtick form is rejected by the go tool but stripping both quote kinds is harmless leniency (D-01: "Quoted module lines parse correctly (strip quotes)").
- **Go directive must NOT be quoted** (verified: `go "1.22.5"` rejected — *"invalid go version '\"1.22.5\"': must match format 1.23.0"*); valid forms verified: `go 1.21`, `go 1.21rc1`, `go 1.21.4`. Report the token verbatim (D-02 raw string).
- **Trailing comments legal** (verified: `module example.com/qux // my comment` → Path `example.com/qux`) — strip everything from the first `//` before tokenizing. Safe for module paths: `//` cannot occur inside a valid path (empty path element is illegal).
- **BOM rejected by the go tool** (verified: `unexpected input character '\ufeff'`) but stripped by `readManifest` before the parser — SC1's "BOM … parse correctly" is satisfied by construction; add a test row with a literal `\xEF\xBB\xBF` prefix.

```go
// parseGoMod extracts the module path and go directive from go.mod content.
// Returns ("", "") when neither directive is present (invalid go.mod, still
// a Go match — see D-disc-1).
func parseGoMod(content []byte) (modulePath, goVersion string) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i]) // trailing comment
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "module":
			if modulePath == "" {
				modulePath = strings.Trim(fields[1], `"`+"`")
			}
		case "go":
			if goVersion == "" {
				goVersion = fields[1] // raw string, never normalized (D-02/DATA-03)
			}
		}
	}
	return modulePath, goVersion
}
```

**When to use:** DETC-02. Known limitation (documented, not handled): the grammar's block form `module ( path )` (never emitted by the go tool) is not parsed — Name falls back to the folder base. `toolchain` lines (`toolchain go1.26.4`) are intentionally ignored — D-02 reports only the `go` directive.

### Pattern 3: JSON detectors + shared readJSONManifest (DETC-03/DETC-04)

**What:** JS and PHP share a 5-line helper; the decode struct is the minimal field set (no `Private` — D-06 needs no code branch, see verified behavior below).

```go
// readJSONManifest decodes <folder>/<name> as JSON into v. Malformed JSON,
// oversized files, and missing files all return false — callers degrade to a
// non-match (D-04, D-12). BOM stripping happens in readManifest and is
// REQUIRED: encoding/json rejects a leading BOM.
func readJSONManifest(folder, name string, v any) bool {
	content, ok := readManifest(folder, name)
	if !ok {
		return false
	}
	return json.Unmarshal(content, v) == nil
}
```

Verified `encoding/json` facts (live probe, go1.27.0) that the test rows should pin:
- BOM prefix → error `invalid character '\ufeff' looking for beginning of value` → `false`. *(readManifest's strip is the reason a BOM'd package.json still matches — SC2 requires it.)*
- Malformed (trailing comma, truncated) / type mismatch (`"name": 123`) → error → `false` (D-04 cascade continues).
- Unknown fields (`license`, `scripts`, …) → silently ignored, no error.
- Duplicate keys → last wins, no error.
- `"private": true` without `version` → Version `""` — **the D-06 test row, not a code branch**.
- `"name": "@scope/pkg"` → preserved verbatim.
- `"description": null` → `""` → triggers README fallback (pin this row).

Detector bodies (shared shape; JS shown):

```go
func detectJS(folder string) (ProjectData, bool) {
	var m struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if !readJSONManifest(folder, "package.json", &m) {
		return ProjectData{}, false // D-04: malformed → cascade continues
	}
	data := ProjectData{Language: LanguageJavaScript} // D-05: always JavaScript
	data.Name = m.Name
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain
	}
	data.Version = m.Version // raw; "" when absent (D-06)
	data.Description = m.Description
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 chain
	}
	return data, true
}
```

**When to use:** DETC-03/DETC-04. PHP is identical with `"composer.json"`, `LanguagePHP`, and the full `vendor/package` name (D-07). Note the **match-rule asymmetry is locked**: JS/PHP match only on parse success (D-04); Go matches on file presence (D-disc-1) — a folder with a *broken* package.json but a valid composer.json correctly resolves to PHP.

### Pattern 4: README first-real-paragraph extraction (D-08..D-10)

**What:** One shared helper; detectors call it only when their manifest description is empty. Candidate order `README.md` → `README.rst` → `README`, exact-case (D-08), each read via `readManifest` (cap + BOM strip); first readable candidate wins; none → `""` (D-10).

Extraction rules (D-09 + RST-spec grounding, verified against docutils "Sections"/"Paragraphs"):

1. **Blank lines** — skip (RST: "Paragraphs consist of blocks of left-aligned text… Blank lines separate paragraphs").
2. **Image-only badge lines** — a line containing `![` whose markdown link segments `[...](...)` (and wrapped `[![...](...)](...)` forms) strip to empty. Rule: repeatedly remove `[^\]]*\]\([^)]*\)` segments; if the remainder is empty, the line is image-only.
3. **TOC link lines** — trimmed line starts with `- [` / `* [` (markdown bullet link) or matches `^\d+[.)] \[` (numbered link).
4. **Underline-heading pairs** — a line of ≥3 identical non-alphanumeric printable ASCII chars (`=`, `-`, `~`, `^`, `_`, `*`, `+`, `#`, `'`, `"`, backtick) is an rst underline **and** a markdown setext heading underline (verified: RST "a single repeated punctuation character that begins in column 1 and forms a line extending at least as far as the right edge of the title text"). When found, discard BOTH it and the immediately preceding non-blank line (the title). Exclude `.` and `:` from the char set (ellipsis `...` and `::` appear in prose).
5. **First consecutive non-blank block** after the skips = the paragraph. Multi-line blocks are joined with a single space (D-disc-4).

**When to use:** DATA-04 in all three detectors. The `README.md`-vs-`README.rst`-vs-`README` candidate list means the same scanner serves markdown, rst, and plain text — the skip rules are a superset, which is exactly what D-09 describes.

### Discretion Decisions Taken (CONTEXT "the agent's Discretion" — locked recommendations)

| # | Decision | Rationale |
|---|----------|-----------|
| D-disc-1 | **Go detector match rule: go.mod presence → match**; unparseable module line → Name = folder base; no `go` directive → Version = `""`. JS/PHP: parse-success only (D-04) | D-01/D-02 frame both go.mod fields as independently optional; DATA-02's fallback chain exists precisely for the missing-name case; go.mod presence is the canonical Go marker (the tool itself assumes `go 1.16` when the directive is absent — we report `""` per D-02 instead of fabricating). SC1's "folder with go.mod → Language=Go" holds even for a go.mod with zero parseable directives |
| D-disc-2 | **Registry representation: nil-slot literal + skip** (Pattern 1) | D-11's fixed positions must be visible in one place; dense slices would force Phase 12/13 reorders |
| D-disc-3 | **Underline char set** excludes `.` and `:`; minimum length 3 | `...` (ellipsis) and `::` (rst literal-block introducer) appear in real prose; RST requires the underline to reach the title's right edge but title length is unknown at scan time — 3 is the pragmatic floor |
| D-disc-4 | **Multi-line paragraphs joined with a single space**; inline markdown markers (`**bold**`, `` `code` ``) preserved verbatim | A Description is a one-line field (composer: "Usually this is one line long"; package.json convention); markdown stripping is normalization beyond DATA-03's raw-string spirit and is a v2 idea |
| D-disc-5 | **ATX headings (`# Title`) are skipped** in addition to D-09's literal list | A heading is not prose; the rst-underline rule (which D-09 names) skips rst headings, and setext headings fall under the same underline rule — ATX is the markdown-only heading form and would otherwise make Description = `# Title`. Extends the letter of D-09; flagged in Open Questions OQ-2 for user confirmation |
| D-disc-6 | **Inline fixtures via `t.TempDir()` + `os.WriteFile`**; README content as inline string constants in test tables | Repo convention (TESTING.md: "Test data: Defined inline as struct fields or local variables", "File I/O — use t.TempDir() with real files"); no `testdata/` dirs exist anywhere in the repo (verified this session); inline strings make the extraction matrix readable |

### Anti-Patterns to Avoid

- **Promoting `golang.org/x/mod/modfile` to a direct dependency:** it is the authoritative parser but violates the stdlib-only v1.7 mandate; it also rejects BOM'd files that SC1 requires; ~25 lines of verified line-parsing replaces it.
- **`json.Unmarshal` without the `readManifest` BOM strip:** verified — Unmarshal errors on a leading BOM; the strip is load-bearing for SC2/SC3, not just SC1.
- **A `Private bool` field in the JS decode struct:** D-06 is a never-fabricate rule — the zero value already yields `""`; a field would be dead code (pin via test row instead).
- **Fabricating composer versions:** `version` is absent in the majority of real composer.json files (Packagist infers from tags) — empty Version is correct, not a bug.
- **Dense detector slice with ordering comments:** loses D-11 position guarantees; Phase 12/13 would need reorders.
- **README extraction that treats `...`/`::`/`---`-between-blank-lines as headings:** over-triggers on prose and markdown thematic breaks; the char-set + ≥3 rule (D-disc-3) is the calibrated threshold.
- **`t.Parallel()` in any test that mutates `detectors`:** AGENTS.md global-state rule; the save/restore pattern from Phase 10 stands [VERIFIED: project_probe/registry_test.go:12-14].

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON decoding | A hand-rolled JSON parser or `json.RawMessage` gymnastics | `encoding/json` `Unmarshal` into a struct | Stdlib, panic-safe on our field types, verified behavior table above; D-04 locks it |
| BOM removal | A leading-byte state machine | `readManifest`'s `bytes.TrimPrefix(content, utf8BOM)` [VERIFIED: project_probe/manifest.go:14,34] | Phase 10 built it; Phase 11 consumes it |
| File-size caps | Per-detector read loops | `readManifest` (1 MB LimitReader probe) [VERIFIED: project_probe/manifest.go:12,29-31] | Phase 10 built it; every detector inherits the cap for free |
| Panic containment | Trusting parsers not to panic | `callDetector` recover → non-match [VERIFIED: project_probe/registry.go:28-37] | Phase 10 built it; nil-skip keeps panics from being triggered by empty slots |
| Folder-name fallback | Hand-rolled last-segment extraction | `filepath.Base` | Stdlib; Windows-safe (ROBT-04) |
| Test assertions | Hand-rolled compare helpers | `testify` `assert`/`require` | Repo mandate (AGENTS.md); `testifylint` enforced |

**Key insight:** everything "hard" in this phase — BOM, caps, panic safety, JSON errors, cross-platform paths — was already solved in Phase 10 or is stdlib behavior verified this session. The genuinely new code is two ~30-line parsers and one ~40-line heuristic; every risk lives in the heuristic thresholds, which is why the README extraction matrix gets the densest test table.

## Common Pitfalls

### Pitfall 1: `json.Unmarshal` fails on a BOM'd package.json/composer.json
**What goes wrong:** SC2/SC3 fixtures written with a UTF-8 BOM return `false` and the detector never matches — or worse, tests pass locally (no BOM) and CI or real-world BOM'd files silently degrade to Unknown.
**Why it happens:** Verified: `Unmarshal` returns `invalid character '\ufeff' looking for beginning of value` on BOM-prefixed input; go.mod has the same behavior (verified with the go tool).
**How to avoid:** Rely on `readManifest`'s strip (already there) and **pin it with a test row**: write `"\xEF\xBB\xBF{...}"` to package.json and assert the detector matches. Never call `json.Unmarshal` on raw `os.ReadFile` output in detectors.
**Warning signs:** A BOM'd package.json folder probes as `LanguageUnknown`.

### Pitfall 2: `make coverage-quick` is red before the first commit (pre-existing)
**What goes wrong:** The mandatory pre-commit gate fails on `release/update.go 68.9%` vs 70% file threshold — verified live this session. Every Phase 11 commit hits it.
**Why it happens:** Pre-existing, unrelated to project_probe (logged in `.planning/phases/10-*/deferred-items.md`); not fixed by Phase 10 (out of scope, deferred).
**How to avoid:** Run `make coverage-quick`, assert the **project_probe rows pass** (pkg ≥80% / file ≥70% — current package coverage is 100% per deferred-items) and total stays ≥75%, and treat the `release/update.go` row as the documented known-red. Do NOT fix release/update.go in this phase (scope boundary rule). Note the known-red in the phase's commit messages or plan verification sections.
**Warning signs:** A plan's verification step fails with no remediation path — the planner must encode the known-red exception explicitly.

### Pitfall 3: Registry position drift when Phase 12/13 arrive
**What goes wrong:** If Phase 11 ships a dense slice (`detectGo, detectJS, detectPHP`), Phase 12 must splice Python between Go and JS — the D-11 order contract breaks under editing churn, and a wrong insert silently changes cascade precedence for mixed-repo folders.
**Why it happens:** The cascade order is only visible as slice position; dense slices hide the empty slots.
**How to avoid:** The nil-slot literal (Pattern 1) makes all seven positions permanent. Test the positions directly: `assert.Equal(t, 7, len(detectors))` and check `detectors[0] != nil`, `detectors[3] != nil`, `detectors[6] != nil`, `detectors[1] == nil` etc. — cheap, pins D-11.
**Warning signs:** A review diff shows detectors reordered rather than slots filled.

### Pitfall 4: Registry tests silently stop testing what they claim
**What goes wrong:** `TestRunDetectors_EmptyRegistry` keeps passing (its injected call on `""` yields no match even with real detectors) but its comment "pins the Phase-10 default state" is now false — a future reader trusts a stale claim.
**Why it happens:** Tests replace the whole slice wholesale; production semantics change without the test noticing.
**How to avoid:** Update the stale comments in `registry_test.go`; add the position-assertion test from Pitfall 3 so the real production literal is exercised, not just fakes.
**Warning signs:** Comments describing "empty registry in Phase 10" surviving into Phase 11 code.

### Pitfall 5: README heuristic over/under-matching
**What goes wrong:** Under-matching: `# Title` becomes the Description (D-disc-5 rejected this). Over-matching: an ellipsis line `...` or a markdown `---` thematic break swallows a real paragraph, or a badge line with trailing text (`![logo](x) Welcome!`) is wrongly skipped.
**Why it happens:** Thresholds are judgment calls (CONTEXT grants discretion); regex-free string checks drift from intent.
**How to avoid:** The extraction matrix test must include: badge-only line, badge+text line (NOT skipped), `- [TOC](#x)` list, `1. [TOC](#x)`, rst `Title\n=======`, markdown setext `Title\n---`, ATX `# Title`, `...` line, `---` between blank lines, multi-line paragraph, HTML comment preamble (`<!-- TOC -->`), empty README, whitespace-only README. Every row pins the threshold.
**Warning signs:** A fixture-rich real README (badges + TOC + heading) yields a Description that starts with `#` or `[![`.

### Pitfall 6: go.mod line-parser false matches on directive-like lines
**What goes wrong:** A line `gopher tools` or `modulex foo` or `toolchain go1.26.4` misparses as the `go`/`module` directive; a quoted module with an escaped quote (`module "a\"b"`) corrupts the value.
**Why it happens:** Naive `HasPrefix("go ")` checks don't respect token boundaries; naive quote stripping doesn't handle escapes.
**How to avoid:** `strings.Fields` + exact first-token equality (Pattern 2); strip only matched outer quotes (first and last rune), never unescape interior sequences — the go tool does unescape interpreted strings, but such paths are pathological; document the leniency. Add rows: `gopher 1.2`, `modulex y`, `toolchain go1.26.4` (must NOT set Version).
**Warning signs:** A probe of a repo with a `toolchain` line returns Version `go1.26.4` — it must return the `go` directive instead.

### Pitfall 7: Exact-case README candidate list surprises
**What goes wrong:** A folder with only `readme.md` (lowercase) or `Readme.md` gets an empty Description even though a README exists.
**Why it happens:** D-08 locks exact-case matching — deliberately (deterministic, cross-platform).
**How to avoid:** It's the locked contract — but pin it with a test row (lowercase `readme.md` alone → `""`), and document in `readme.go`'s doc comment that matching is exact-case per D-08. Don't "helpfully" case-fold.
**Warning signs:** A contributor "fixes" the case-insensitivity and breaks the D-08 contract.

## Code Examples

Verified patterns from official sources plus live probes:

### go.mod parsing — quoted module + trailing comment (Pattern 2 mechanics)
```go
// Input: module "example.com/foo/bar" // my comment
// Tokenize: ["module", `"example.com/foo/bar"`]  (comment stripped first)
// Name:    example.com/foo/bar                    (quotes stripped, D-01)
// Input: go 1.21rc1
// Version: 1.21rc1                                (raw, D-02; rc forms verified valid)
// Input: toolchain go1.26.4
// Version: UNCHANGED                              (toolchain ≠ go directive)
```

### README extraction — skip-table skeleton
```go
// firstRealParagraph returns the first non-blank text block of content after
// skipping badge lines, TOC link lists, and heading/underline pairs (D-09).
func firstRealParagraph(content []byte) string {
	lines := strings.Split(string(content), "\n")
	var para []string
	var prev string // last non-blank line (candidate heading title)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			prev = ""
		case isUnderline(trimmed): // discard title + underline pair
			prev = ""
		case isBadgeLine(trimmed) || isTOCLine(trimmed) || isHeading(trimmed):
			prev = ""
		case para != nil:
			para = append(para, trimmed)
		case prev != "" && looksLikeTitle(prev): // ATX/setext handled above; plain single line
			prev = trimmed
		default:
			para = []string{trimmed}
		}
		prev = trimmed
	}
	return strings.Join(para, " ")
}
```
*(Shape only — the exact branch structure is executor discretion; the behavior matrix in Pitfall 5 is the contract.)*

### Registry position test
```go
// TestDetectorPositions pins the D-11 order contract: Go/JS/PHP at 0/3/6,
// Python/C#/Rust/Java slots nil until phases 12-13.
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
	assert.Len(t, detectors, 7)
	assert.NotNil(t, detectors[0]) // Go
	assert.Nil(t, detectors[1])    // Python — Phase 12
	assert.Nil(t, detectors[2])    // C#/.NET — Phase 13
	assert.NotNil(t, detectors[3]) // JS/TS
	assert.Nil(t, detectors[4])    // Rust — Phase 12
	assert.Nil(t, detectors[5])    // Java/Kotlin — Phase 13
	assert.NotNil(t, detectors[6]) // PHP
}
```

### DATA-02/DATA-04 chain — Go detector (presence-based, D-disc-1)
```go
func detectGo(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "go.mod")
	if !ok {
		return ProjectData{}, false // D-12: never-fail degrade
	}
	name, version := parseGoMod(content) // Pattern 2
	data := ProjectData{Language: LanguageGo} // LanguageGo = "Go" [VERIFIED: project_probe/project.go:30-31]
	data.Name = name
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02
	}
	data.Version = version // raw go directive or "" (D-02)
	data.Description = readmeDescription(folder) // go.mod has no description (D-03)
	return data, true
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `project_detector` sample (deleted Phase 10): extension-count + error-as-control-flow | Manifest-first cascade with presence/parse match rules per language | v1.7 (this phase) | Folders with go.mod/package.json/composer.json now return real Language/Name/Version/Description; Unknown remains for everything else (DETC-09 unchanged) |
| Empty registry (Phase 10) | 3 of 7 slots filled with nil placeholders | This phase | Mixed-repo precedence live: Go wins over JS/PHP; JS wins over PHP (first-match, DETC-01) |
| README as ignored content | README first-paragraph Description fallback | This phase | DATA-04 chain completes; badge/TOC/heading noise filtered by verified heuristics |
| `json.Unmarshal` on raw file content | `readManifest` BOM strip + `Unmarshal` | This phase | BOM'd manifests (common from Windows editors) now parse — verified required |

**Deprecated/outdated:**
- Hand-written `version` fields in composer.json — Composer explicitly recommends omitting them (Packagist infers from tags) [CITED: getcomposer.org/doc/04-schema.md §version]; empty Version is the ecosystem norm.
- `golang.org/x/mod/modfile` for *this* package's needs — deferred per stdlib-only mandate; revisit only under the v2 framework milestone.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go detector matching on go.mod **presence** (even a go.mod with zero parseable directives) is the intended DATA-02/DETC-02 semantics (D-disc-1) | Discretion Decisions | If the user intended parse-required symmetry with JS (D-04), a folder with a garbage go.mod would need to return Unknown instead of Go — small behavior change, cheap to adjust (one test row + one condition) |
| A2 | `//` can never appear inside a valid module path, so stripping inline comments before tokenizing is safe (Pattern 2) | Pattern 2 | go.dev/ref/mod lexical rules ban empty path elements; a pathological quoted path containing `//` would be truncated — acceptable, documented leniency |
| A3 | The block-form `module ( path )` directive (in the EBNF, never emitted by the go tool) can be left unparsed — Name falls back to folder base | Pattern 2 | Frequency ~0 in the wild; if encountered, Name degrades gracefully rather than erroring |
| A4 | Backtick-quoted module lines can be stripped like double quotes even though the go tool rejects them | Pattern 2 | Harmless leniency; D-01 only requires quoted lines to parse, and double-quoted is the only tool-valid form |
| A5 | The underline char set excluding `.`/`:` with a ≥3 length floor is the right badge-vs-text threshold (D-disc-3) | Pattern 4 | Wrong thresholds produce wrong Descriptions on exotic READMEs — mitigated by the extraction matrix; thresholds are CONTEXT-granted discretion |
| A6 | Skipping ATX headings extends D-09's literal skip list without user approval (D-disc-5) | Pattern 4 / OQ-2 | If the user wanted the literal list only, Description would include `# Title` — flagged as OQ-2 for explicit confirmation; the test matrix pins whichever is chosen |
| A7 | `filepath.Base` on the cleaned folder is the correct DATA-02 folder-base (handles `sub/dir` → `dir`; `"."` → `"."`) | DATA-02 | Standard stdlib behavior; edge rows pin it |
| A8 | The existing Phase 10 tests remain green with real detectors registered (unknown_content rows write plain files, not manifests) | Patterns | Verified by inspection: probe_test.go rows create no go.mod/package.json/composer.json; registry tests replace the slice wholesale |
| A9 | `readManifest`'s BOM strip is the only BOM handling needed for JSON (no UTF-16 manifests required) | Pattern 3 | UTF-16 JSON would fail Unmarshal → detector false → Unknown; acceptable per Phase 10 A3 (documented future extension) |

## Open Questions

1. **Should the WR-01 regular-file gate land in this phase?** *(planner's call — advisory from Phase 10 review)*
   - What we know: `readManifest` opens without a regular-file check; a FIFO named `go.mod`/`package.json`/`README.md` blocks `os.Open` forever — a hang is worse than the never-fail contract's failures, and Phase 11 now calls `readManifest` on every probe (three manifest names + three README candidates). Fix shape verified in 10-REVIEW.md: `f.Stat()` + `info.Mode().IsRegular()` after open, before read.
   - What's unclear: whether to include it (CONTEXT says "advisory; consider WR-01 fix if touching readManifest") or defer to the Phase 14 ROBT-05 audit.
   - Recommendation: **include it** as a small task (one `manifest.go` edit + one FIFO test row gated `runtime.GOOS == "windows"` skip; `syscall.Mkfifo` on darwin/linux). Cost is ~15 minutes; benefit is closing a real DoS on the never-fail contract. If the planner defers, `readme.go`'s candidate loop should at least read via `readManifest` (already the case) and the FIFO risk stays documented.

2. **ATX heading skip — confirm D-disc-5 (extends the letter of D-09).**
   - What we know: D-09's literal list (badges, TOC, rst underline headings, blanks) doesn't name `# Title`; the rst-underline rule covers rst/setext headings; ATX is the markdown-only heading form; without the skip, Description = `# My Project` for most real README.md files.
   - What's unclear: whether "first real paragraph" (SC5's wording) intends prose-only or first-non-skipped-line.
   - Recommendation: adopt D-disc-5 (skip `^#{1,6}\s+` lines); the extraction matrix pins it either way. This is within the CONTEXT-granted discretion ("what counts as text") — no user confirmation strictly required, but the planner should note the choice in the plan.

3. **Registry test file churn** — how much to update `registry_test.go` comments vs. leave alone.
   - What we know: all existing tests pass unchanged; two comments ("pins the Phase-10 default state") go stale.
   - Recommendation: minimal comment refresh + the `TestDetectorPositions` row (Code Examples) — no test rewrites needed.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All code/tests | ✓ | go1.27.0 local (module declares 1.26.4; CI pins via go-version-file) | Toolchain auto-download per go.mod |
| `make` | `make coverage-quick` gate | ✓ | GNU Make 3.81 | — |
| `go-test-coverage` | Coverage gate | ✓ | installed (~/go/bin) | `make check-go-test-coverage` auto-installs |
| `golangci-lint` | Pre-commit / CI lint | ✓ | v2.12.2 | `GOTOOLCHAIN=go1.26.4 golangci-lint run` if the toolchain-compat issue bites (scoped `./project_probe/...` run passes both ways — verified this session) |
| `pre-commit` | Commit hooks | ✓ | 4.6.0 | CI enforces the same checks |
| Coverage gate health | Every commit | ⚠️ **known-red** | `make coverage-quick` fails on `release/update.go` 68.9% vs 70% (pre-existing, deferred) | Assert project_probe rows ≥ thresholds + total ≥75% only; document the known-red (Pitfall 2) |
| 3-OS CI matrix | Cross-platform guarantees | ✓ (GitHub Actions) | — | Local darwin + skip-gated FIFO/permission rows |

**Missing dependencies with no fallback:** none — stdlib-only phase, all tooling present.

## Validation Architecture

> **SKIPPED** — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly; per the research contract this section is omitted. The repo's standard gates (pre-commit `go test ./...`, `make coverage-quick`, `testify` assertions) apply as documented in Project Constraints.

## Security Domain

> Required — `workflow.security_enforcement: true` in .planning/config.json. ASVS level 1; block on HIGH.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no users/sessions) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (OS permissions surface via existing sentinels) |
| V5 Input Validation | yes | All manifest/README content is untrusted input: `readManifest` 1 MB cap [VERIFIED: project_probe/manifest.go:12,29-31]; `json.Unmarshal` is panic-safe on the fixed field set (errors, never panics — verified live); parseGoMod/firstRealParagraph operate on capped bytes; no regexes on unbounded input |
| V6 Cryptography | no | — |

### Known Threat Patterns for {stdlib manifest parsing}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| FIFO/special file at a manifest path hangs the probe (WR-01) | DoS | Recommended: regular-file gate (`f.Stat()` + `Mode().IsRegular()`) in `readManifest` (OQ-1); without it, document the hang risk in `readme.go`/detector docs |
| Pathological manifest content causing parser panics | DoS | `callDetector` per-detector recover → non-match [VERIFIED: project_probe/registry.go:28-37]; `encoding/json` returns errors (verified) |
| Oversized README/manifest memory exhaustion | DoS | `readManifest` LimitReader cap inherited by all six call sites (3 manifests × 3 README candidates) |
| Malformed manifest spoofing a different language (broken package.json + valid composer.json) | Spoofing | Parse-success match rule for JS/PHP (D-04) lets the cascade continue to the valid manifest; Go is presence-based (D-disc-1) — a garbage go.mod claims Go, which is the ecosystem-correct reading |
| Symlink-following / network / build-tool execution | Tampering | ROBT-05 anti-features — none introduced; stdlib-only imports; no `EvalSymlinks`, no `os/exec`, no `net/http` (Phase 10 prohibitions stand) |

## Project Constraints (from AGENTS.md)

Directives the planner must honor (verbatim intent from `./AGENTS.md`):

- **Milestone branches:** every milestone gets its own git branch for the PR flow; work on the milestone branch, then open a PR to `main`. Branch name: `gsd/v1.7-{slug}` (current branch: `gsd/v1.7-project-probe`).
- **Before every commit:** run `make coverage-quick` — enforces thresholds in `.testcoverage-quick.yml` (packages ≥80%, files ≥70%, total ≥75%). **Known-red exception:** the gate currently fails on `release/update.go` (68.9% vs 70%, pre-existing, logged in deferred-items.md) — assert project_probe rows + total instead and document the known-red (Pitfall 2). Do not commit if *project_probe* fails.
- **Before pushing a tag:** regenerate and stage the quality report (`make quality-report`, `git add quality-report.md`).
- **Cross-platform testing (CI runs Linux/macOS/Windows):** tests modifying global state (the detector registry slice) must NOT use `t.Parallel()`; FIFO-based test rows (if OQ-1 lands) must be skip-gated on Windows (`syscall.Mkfifo` is Unix-only); never rely on OS-specific file-mode behaviors in assertions.
- **Code style:** standard Go idioms; focused single-purpose functions; `testify` for assertions; unit + edge-case tests; Go doc comments for all exported symbols; Conventional Commits; minimal external dependencies — prefer stdlib.
- **Spike findings:** consult `Skill("spike-findings-go")` before committing (auto-loaded during implementation; the panic-recovery precedent it documents is the registry dispatch this phase extends).

## Sources

### Primary (HIGH confidence)
- [VERIFIED: go.dev/ref/mod] — go.mod file grammar: line-oriented directives, `//`-only comments, `ModulePath = ident | string`, `GoVersion = string | ident`, toolchain directive form, module-path lexical rules
- [VERIFIED (live probe, go1.27.0)] — `go mod edit -json` on 12 go.mod variants: BOM rejected (`unexpected input character '\ufeff'`); `module "example.com/dq"` accepted, backtick form rejected (`invalid quoted string`); `go 1.21` / `go 1.21rc1` / `go 1.21.4` accepted; `go "1.22.5"` rejected; trailing `//` comments stripped; `/* */` rejected; `toolchain go1.26.4` parsed separately
- [VERIFIED (live probe, go1.27.0)] — `encoding/json` behavior table: BOM/malformed/type-mismatch → error; unknown fields ignored; duplicate keys last-wins; private-no-version → `""`; scoped names preserved; `null` description → `""`
- [VERIFIED: getcomposer.org/doc/04-schema.md] — composer.json name (`vendor/package`, regex), version (usually omitted — inferred from VCS tags), description semantics
- [VERIFIED: docutils.sourceforge.io/docs/ref/rst/restructuredtext.html#sections] — RST section titles: single repeated punctuation char, column 1, ≥ title width; recommended chars `= - \` : . ' " ~ ^ _ * + #`; paragraph = left-aligned block separated by blank lines; transitions = 4+ repeated chars
- [VERIFIED (in-repo): project_probe/registry.go:6-37] — `detectorFunc` contract, `var detectors = []detectorFunc{}`, `runDetectors`, `callDetector` recover
- [VERIFIED (in-repo): project_probe/manifest.go:12-35] — `maxManifestSize = 1 << 20`, `utf8BOM = []byte{0xEF, 0xBB, 0xBF}`, `readManifest` BOM strip + LimitReader cap
- [VERIFIED (in-repo): project_probe/project.go:3-53] — `ProjectData{Folder, Language, Name, Version, Description}`; constants incl. `LanguageGo Language = "Go"`, `LanguageJavaScript Language = "JavaScript"`, `LanguagePHP Language = "PHP"`, `LanguageUnknown Language = "unknown"`
- [VERIFIED (in-repo): project_probe/registry_test.go:12-85] — no-parallel + save/restore test pattern; `TestProbe_MergeRule` merge-rule coverage
- [VERIFIED (in-repo): .testcoverage-quick.yml, Makefile, deferred-items.md] — coverage thresholds; `make coverage-quick` red on `release/update.go` (pre-existing)
- [VERIFIED (in-repo): .planning/phases/10-*/10-REVIEW.md] — CR-01/WR-01/WR-02 findings; WR-01 regular-file gate fix shape
- [VERIFIED (in-repo): .planning/codebase/CONVENTIONS.md, TESTING.md, STACK.md] — naming, inline fixtures via `t.TempDir()`, stdlib-only, testify, no `testdata/` dirs in repo

### Secondary (MEDIUM confidence)
- [CITED: websearch corroboration] — markdown badge syntax `[![alt](url)](url)` and setext headings (`Title` + `===`/`---` underline) — standard markdown facts corroborated by multiple cheatsheet sources

### Tertiary (LOW confidence)
- [ASSUMED] — heuristic thresholds beyond the verified RST/setext rules (underline min length 3, char-set minus `.`/`:`): calibrated judgment per CONTEXT-granted discretion, pinned by the extraction matrix tests

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib-only mandate; every library behavior verified live or against official docs; zero new dependencies
- Architecture: HIGH — registry extension pattern verified against the Phase 10 contract; parser mechanics verified against the real go tool; heuristic thresholds flagged honestly (MEDIUM sub-part)
- Pitfalls: HIGH — the coverage-gate red state and json BOM behavior reproduced live this session; remaining pitfalls are standard fixture/threshold discipline

**Research date:** 2026-09-29
**Valid until:** 2026-10-29 (go.mod grammar and encoding/json semantics stable; heuristic thresholds are ours to own)
