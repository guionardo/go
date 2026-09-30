# Phase 12: TOML Subset + Python/Rust Detectors - Context

**Gathered:** 2026-09-29
**Status:** Ready for planning

<domain>
## Phase Boundary

The unexported section-aware TOML-subset reader (ROBT-03, ~80 lines) plus the Python detector (pyproject.toml PEP 621 `[project]` + legacy `[tool.poetry]`) and Rust detector (Cargo.toml `[package]`) wired into the existing 7-slot registry at their research-locked positions (Python index 1, Rust index 4).

Depends on Phase 11 (registry literal, readManifest, readJSONManifest, readmeDescription, never-fail semantics) — all locked there.

</domain>

<decisions>
## Implementation Decisions

### TOML-Subset Reader (ROBT-03)
- **D-01:** Unexported section-aware reader (~80 lines), stdlib-only. Supported grammar: section headers `[name]`, bare or `"quoted"` keys, `=` separator, `"`/`'` quoted scalar values, `#` comments, blank lines.
- **D-02:** **Strict degrade-to-empty** on legal-but-unsupported TOML: multiline strings, dotted keys, inline tables, arrays, unquoted values → empty fields for that read, never a panic, never partial data. (Phase 12 blocker: TOML strict-degrade needs fixture-driven validation.)
- **D-03:** Reader API shape: `readTOMLSection(content []byte, section string) map[string]string` (or equivalent unexported helper) — consumes `readManifest` output; section missing → empty map (not error).

### Python Detector (DETC-05)
- **D-04:** `detectPython(folder) (ProjectData, bool)` — pyproject.toml via readManifest; primary fields from `[project]` (PEP 621: name/version/description); legacy fallback to `[tool.poetry]` name/version/description when `[project]` absent.
- **D-05:** Description → README fallback via `readmeDescription` (DATA-04); name → folder-base fallback (DATA-02) when both sections lack a name.

### Rust Detector (DETC-06)
- **D-06:** `detectRust(folder) (ProjectData, bool)` — Cargo.toml via readManifest; `[package]` name/version/description.
- **D-07:** `version.workspace = true` → empty Version — never fabricated, never resolved from a workspace root (Phase 12 SC3). Other dynamic/unsupported version forms degrade to empty.

### Registry Wiring
- **D-08:** `detectPython` at registry index 1, `detectRust` at index 4 — completes 5 of 7 live slots (C#/.NET index 2, Java/Kotlin index 5 remain for Phase 13). Order preserved: Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP.
- **D-09:** Both detectors follow the never-fail contract: `readManifest` failure → `(ProjectData{}, false)`; malformed TOML → strict empty → detector still matches on manifest presence (like detectGo D-disc-1) with empty/fallback fields.

### the agent's Discretion
- Exact reader implementation (scanner vs line-based), fixture layout for TOML edge cases, test row organization. Follow repo conventions (CONVENTIONS.md) and the Phase 10/11 package patterns (PATTERNS.md).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone requirements & roadmap
- `.planning/REQUIREMENTS.md` — Phase 12 covers DETC-05, DETC-06, ROBT-03; FRAM-02 (full TOML dep) deferred to v2
- `.planning/ROADMAP.md` §Phase 12 — goal, 4 success criteria, depends-on Phase 11
- `.planning/STATE.md` — blockers (TOML strict-degrade fixture validation), research decisions

### Prior phase contracts (the detectors build on)
- `.planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-CONTEXT.md` — D-01..D-12: detector patterns, registry literal, chains
- `.planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-RESEARCH.md` — detector implementation research
- `.planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-PATTERNS.md` — analog patterns (detect_go.go, json.go, readme.go, registry.go)
- `.planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-VERIFICATION.md` — verified Phase 11 state
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-CONTEXT.md` — registry contract, readManifest, never-fail (D-01..D-12)

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/STACK.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `project_probe/registry.go` — 7-slot literal `{detectGo, nil, nil, detectJS, nil, nil, detectPHP}`; fill indices 1 and 4
- `project_probe/manifest.go` — readManifest (1MB cap, BOM strip, WR-01 regular-file gate)
- `project_probe/readme.go` — readmeDescription fallback
- `project_probe/detect_go.go` — detector pattern reference (presence-match, folder-base fallback)
- `project_probe/detect_javascript.go`, `detect_php.go` — JSON/never-fail patterns

### Established Patterns
- Never-fail: detectors return `(ProjectData, bool)`, manifest presence = match, content failure = empty fields
- TDD adaptation: RED verified-but-uncommitted, RED evidence in feat commit body (Phase 10 precedent)
- `_javascript` filename over `_js` (legacy GOARCH constraint — Phase 11 lesson)

### Integration Points
- Registry indices 1 and 4 in the existing 7-slot literal
- `readTOMLSection` consumed only by detectPython/detectRust (unexported, like readManifest)
- doc.go detector list refresh (names all live detectors after this phase)

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the discussion — decisions above are the contract. Open to standard approaches for reader internals.

</specifics>

<deferred>
## Deferred Ideas

- Full TOML dependency (`pelletier/go-toml/v2`) — v2 backlog (FRAM-02), stdlib-only constraint holds for v1.7
- Workspace-root version resolution for Cargo — not in scope; `version.workspace` degrades to empty

</deferred>

---

*Phase: 12-TOML Subset + Python/Rust Detectors*
*Context gathered: 2026-09-29*
