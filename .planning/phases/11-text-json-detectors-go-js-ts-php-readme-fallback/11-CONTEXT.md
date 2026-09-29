# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback - Context

**Gathered:** 2026-09-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the first three detectors into the Phase 10 `projectprobe` registry: Go (go.mod), JavaScript (package.json), PHP (composer.json) — plus the DATA-02 name chain (manifest → folder base) and DATA-04 description chain (manifest → README first paragraph → empty). README fallback must skip badges, tables of contents, and rst-style underline headings.

Depends on Phase 10 (registry contract, `readManifest`, `Language` constants, never-fail semantics) — all locked there.

</domain>

<decisions>
## Implementation Decisions

### Go Detector (DETC-02)
- **D-01:** Name = go.mod `module` line verbatim (full module path, e.g. `github.com/guionardo/go` — not the last path segment). Quoted module lines parse correctly (strip quotes); BOM already stripped by `readManifest`.
- **D-02:** Version = `go` directive raw string (e.g. `1.26.4`), documented as **toolchain floor, not release version** — matches RESEARCH decision carried from STATE.md; empty when no `go` directive. Never fabricated, never normalized (DATA-03).
- **D-03:** go.mod has no description field → description comes from README fallback (DATA-04) or empty.

### JS/TS Detector (DETC-03)
- **D-04:** package.json name/version/description via `encoding/json`; malformed JSON → detector returns false (next detector in cascade; per D-03 never-fail, degrade-to-Unknown).
- **D-05:** Language is ALWAYS `LanguageJavaScript` for package.json — the distinct `typescript` value is v2 backlog (REFN-01), NOT this phase.
- **D-06:** `private: true` with no version → empty Version, never fabricated.

### PHP Detector (DETC-04)
- **D-07:** composer.json name/version/description; Name = full `vendor/package` string (ecosystem identifier — consistent with Go module-path reporting, not the last segment).

### README Fallback (DATA-04)
- **D-08:** Ordered deterministic candidate list: `README.md` → `README.rst` → `README` (exact-case, first existing wins). Only when the manifest has no description.
- **D-09:** "First real paragraph" extraction skips: image-only badge lines (`[![...](...)`), TOC-style link lists (leading `- [` or numbered link lines at the top), rst underline headings (`===`/`---` underlines), and blank lines — then takes the first consecutive non-blank text block.
- **D-10:** No README → empty description (never an error).

### Registry Integration
- **D-11:** Detectors register in the research-locked order (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) — this phase fills Go, JS/TS, PHP entries at their fixed positions; Python/C#/.NET/Rust/Java slots stay empty until phases 12-13.
- **D-12:** Each detector calls `readManifest(folder, name)` for its manifest; failure degrades to `(ProjectData{}, false)` — never-fail contract (D-03 from Phase 10).

### the agent's Discretion
- Exact go.mod line-parsing mechanics (strings within `module`/`go` lines), README content scoring thresholds (what counts as "badge line" vs text), test fixture layout. Follow repo conventions (CONVENTIONS.md) and the Phase 10 package patterns (PATTERNS.md).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone requirements & roadmap
- `.planning/REQUIREMENTS.md` — Phase 11 covers DETC-02, DETC-03, DETC-04, DATA-02, DATA-04; DATA-03 (version semantics) and ROBT-05 (anti-features) belong to Phase 14
- `.planning/ROADMAP.md` §Phase 11 — goal, 5 success criteria, depends-on Phase 10
- `.planning/STATE.md` — research decisions carried forward (go.mod `go` directive = toolchain floor; detector order)

### Phase 10 foundation (locked contracts the detectors build on)
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-CONTEXT.md` — D-01..D-12: sentinels, Language constants, registry `(ProjectData, bool)` contract, ignore list, clean-once
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-RESEARCH.md` — readManifest design, error mapping, panic-recovery at dispatch
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-PATTERNS.md` — analog patterns and code excerpts (registry_test no t.Parallel, etc.)
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-REVIEW.md` — CR-01 (Windows errno), WR-01 (FIFO gate on readManifest) — advisory; consider WR-01 fix if touching readManifest
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-VERIFICATION.md` — verified Phase 10 state, must-haves

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md` — naming, error handling, doc.go requirements, declaration order
- `.planning/codebase/STRUCTURE.md` — package layout rules
- `.planning/codebase/STACK.md` — Go 1.26.4, stdlib-only, testify

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `project_probe/` — registry.go (`detectorFunc`, `runDetectors`), read_manifest.go (`readManifest(folder, name) ([]byte, bool)`), project.go (Language constants), errors.go (sentinels), probe.go (clean-once, content gate) — the whole Phase 10 package the detectors plug into
- `project_probe/registry_test.go` — save/restore detector slice pattern for tests (no t.Parallel)

### Established Patterns
- Never-fail contract: detectors return `(ProjectData, bool)`, never error (Phase 10 D-03)
- Manifest-first cascade: `readManifest` → parse → match; failure degrades to non-match
- TDD adaptation: RED verified-but-uncommitted (pre-commit go-test hook), RED evidence in feat commit body (Phase 10 precedent, user-approved override)

### Integration Points
- Detectors register into the existing `detectors` slice in `project_probe/registry.go` — Go/JS/TS/PHP entries land at research-locked positions
- README fallback lives in a shared helper (description chain consumed by detectors) — no network, no symlink following (ROBT-05 anti-features)

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the discussion — decisions above are the contract. Open to standard approaches for parsing mechanics.

</specifics>

<deferred>
## Deferred Ideas

- Distinct `typescript` Language value — v2 backlog (REFN-01), NOT this phase

</deferred>

---

*Phase: 11-Text/JSON Detectors — Go, JS/TS, PHP + README Fallback*
*Context gathered: 2026-09-29*