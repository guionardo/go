# Phase 13: XML Detectors — C#/.NET + Java/Kotlin - Context

**Gathered:** 2026-09-29
**Status:** Ready for planning

<domain>
## Phase Boundary

The final two detectors wired into the 7-slot registry: C#/.NET (`.csproj`, XMLName local-name matching, namespace-agnostic) at index 2, and Java/Kotlin (pom.xml with parent version inheritance; settings.gradle `rootProject.name` fallback) at index 5 — completing all 7 live slots.

Depends on Phase 12 (registry literal, readManifest, readmeDescription, never-fail semantics) — all locked there.

</domain>

<decisions>
## Implementation Decisions

### XML Parsing (DETC-07)
- **D-01:** `encoding/xml` with XMLName local-name matching (namespace-agnostic by LOCAL name, not URI) — stdlib-only, no dependencies. Works with or without `xmlns` on the root (SC1).
- **D-02:** BOM handled by readManifest (already stripped) — SC1's "with BOM" requirement is satisfied at the read boundary.

### .csproj Detector (DETC-07)
- **D-03:** `detectCSharp(folder) (ProjectData, bool)` — `.csproj` via readManifest; Name chain: `AssemblyName` → `RootNamespace` → folder base (DATA-02); Version: `Version` → `VersionPrefix` → empty (never fabricated); Description → readmeDescription (DATA-04).
- **D-04:** Root-scoped only: `.csproj` in a subdirectory never triggers the parent folder (SC4).

### Java/Kotlin Detector (DETC-08)
- **D-05:** `detectJavaKotlin(folder) (ProjectData, bool)` — pom.xml primary; Name: `<name>` → `<artifactId>` → folder base; Version: `<version>` → when absent inherit `<parent><version>` from parent POM (SC2, single level — no transitive chain).
- **D-06:** settings.gradle `rootProject.name` fallback ONLY when no pom.xml (SC3); Language = LanguageJava for both pom.xml and settings.gradle (Kotlin-distinct value is v2 REFN-02, NOT this phase).
- **D-07:** Root-scoped only: pom.xml in a subdirectory never triggers the parent folder (SC4).

### Registry Wiring
- **D-08:** `detectCSharp` at index 2, `detectJavaKotlin` at index 5 — completes the 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}`; TestDetectorPositions final state (all 7 non-nil); doc.go names all seven.
- **D-09:** Both detectors follow the never-fail presence-match contract: manifest presence = match, parse failure → empty/fallback fields, `readManifest` failure → `(ProjectData{}, false)`.
- **D-10:** Detector ORDER confirmed (STATE.md research decision lands here): Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP.

### the agent's Discretion
- Exact XML struct shapes for decoding, settings.gradle line parsing, fixture layout, test row organization. Follow repo conventions (CONVENTIONS.md) and prior-phase package patterns (PATTERNS.md).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone requirements & roadmap
- `.planning/REQUIREMENTS.md` — Phase 13 covers DETC-07, DETC-08; REFN-01 (typescript), REFN-02 (kotlin value), REFN-03 (lockfiles), REFN-04 (multi-csproj) are v2 backlog
- `.planning/ROADMAP.md` §Phase 13 — goal, 4 success criteria, depends-on Phase 12
- `.planning/STATE.md` — research decisions (detector order confirmation, .NET marker precedence, root-scoped consensus)

### Prior phase contracts (the detectors build on)
- `.planning/phases/12-toml-subset-python-rust-detectors/12-CONTEXT.md` — D-01..D-09: detector patterns, registry slots, chains
- `.planning/phases/12-toml-subset-python-rust-detectors/12-RESEARCH.md` — detector implementation research
- `.planning/phases/12-toml-subset-python-rust-detectors/12-PATTERNS.md` — analog patterns (detect_python.go, detect_rust.go, toml.go)
- `.planning/phases/12-toml-subset-python-rust-detectors/12-VERIFICATION.md` — verified Phase 12 state
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-CONTEXT.md` — registry contract, readManifest, never-fail (D-01..D-12)

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/STACK.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `project_probe/registry.go` — 7-slot literal `{detectGo, detectPython, nil, detectJS, detectRust, nil, detectPHP}`; fill indices 2 and 5
- `project_probe/manifest.go` — readManifest (1MB cap, BOM strip, FIFO gate)
- `project_probe/readme.go` — readmeDescription fallback
- `project_probe/detect_python.go`/`detect_rust.go` — detector pattern references (presence-match, folder-base, readManifest)
- `project_probe/project.go` — Language constants (LanguageCSharp="C#/.NET", LanguageJava="Java" exist)

### Established Patterns
- Never-fail: detectors return `(ProjectData, bool)`, manifest presence = match, content failure = empty fields
- TDD adaptation: RED verified-but-uncommitted, RED evidence in feat commit body (Phase 10 precedent)
- `_python`/`_rust` filename safety verified (no GOOS/GOARCH collision — Phase 11 `_js` lesson); `_csharp`/`_java`/`_kotlin` should be verified the same way

### Integration Points
- Registry indices 2 and 5 in the existing 7-slot literal — final state
- doc.go detector list refresh (all seven named after this phase)
- TestDetectorPositions final flip (all 7 non-nil)

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the discussion — decisions above are the contract. Open to standard approaches for XML decoding internals.

</specifics>

<deferred>
## Deferred Ideas

- Distinct Kotlin Language value — v2 backlog (REFN-02)
- Lockfile secondary confirmation (package-lock.json, Cargo.lock, go.sum) — v2 (REFN-03)
- Multiple `.csproj`/`.sln` selection rule for .NET monorepos — v2 (REFN-04)

</deferred>

---

*Phase: 13-XML Detectors — C#/.NET + Java/Kotlin*
*Context gathered: 2026-09-29*