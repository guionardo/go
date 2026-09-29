# Phase 10: Package Foundation — API Contract + Repo Cleanup - Context

**Gathered:** 2026-09-28
**Status:** Ready for planning

<domain>
## Phase Boundary

The `project_probe` package foundation: public API contract (never-fail `Probe`, `ProjectData` model, `Language` values, error sentinels), the internal ordered detector registry with `(ProjectData, bool)` contract, the ignore list, and the shared `readManifest` helper — plus deleting the broken `project_detector/` sample so `go build ./...` passes across the module.

No language detectors are implemented in this phase (phases 11-13). Detector ORDER is locked now via the registry contract; detector bodies arrive later.

</domain>

<decisions>
## Implementation Decisions

### Error Contract
- **D-01:** Three exported sentinel errors: `ErrFolderNotFound` (missing folder), `ErrNotDirectory` (path is a file), `ErrPermissionDenied` (EACCES). Any other I/O failure returns a plain wrapped error. — **Reversibility:** one-way — exported sentinels are a published contract; removing them later breaks consumer `errors.Is` checks.
- **D-02:** Probe errors wrap the failing folder path: `fmt.Errorf("probe %s: %w", folder, err)` — callers get context AND `errors.Is` still matches. Matches repo convention (`config`, `fraction` sentinels).
- **D-03:** Manifest-read failures inside detectors (e.g., unreadable go.mod in a readable folder) degrade to Unknown / nil error — never surface as Probe errors. The never-fail contract reserves errors for hard folder-level I/O failures only. — **Reversibility:** costly — changing this later means detectors need an error return channel, touching the `(ProjectData, bool)` registry contract.
- **D-04:** Error mapping: `os.Stat`/`os.ReadDir` ENOENT → `ErrFolderNotFound`; ENOTDIR → `ErrNotDirectory`; EACCES **or EPERM** → `ErrPermissionDenied`; everything else → wrapped raw error. — *(Amended 2026-09-28, plan-revision pass: EPERM added to the permission arm per RESEARCH.md assumption A2 and the researched Pattern 3 code example — some Unix filesystems report EPERM for permission failures, and EPERM is never produced by Stat/ReadDir on Windows. The behavioral change is narrow: EPERM errors now `errors.Is`-match `ErrPermissionDenied` instead of falling through to the raw wrap; D-01's one-way sentinel contract is unchanged. Plan 10-02's mapping arm and EPERM test row implement this amended scope.)*

### Language Type
- **D-05:** `type Language string` with exported constants: `LanguageGo`, `LanguagePython`, `LanguageJavaScript`, `LanguageCSharp`, `LanguageRust`, `LanguageJava`, `LanguagePHP`, `LanguageUnknown`. — **Reversibility:** one-way — exported type + constants are a published contract for switch statements and JSON output.
- **D-06:** Constant string values are human-readable display values matching success criteria: `"Go"`, `"Python"`, `"JavaScript"`, `"C#/.NET"`, `"Rust"`, `"Java"`, `"PHP"`. Identifiers are PascalCase for Go code; values are what users see/log.
- **D-07:** `LanguageUnknown = "unknown"` — explicit sentinel value, NOT the zero-value `""`. Detectors always set Language explicitly; `"unknown"` reads better in logs/JSON than `""`. Extensible for v2 `typescript` value (REFN-01).

### Folder Field Semantics
- **D-08:** `ProjectData.Folder` = `filepath.Clean(as-given)` — relative stays relative, never absolutized. Deterministic for temp-dir tests; Windows paths survive. — **Reversibility:** costly — absolutizing later changes returned values for every relative probe.
- **D-09:** Clean once at entry; the cleaned path is reused for `os.Stat`/`os.ReadDir`, error wrapping, and the Folder field — one canonical path, errors and data always agree.

### Ignore List (ROBT-01)
- **D-10:** Default ignore list (unexported): the 5 criteria-named dirs `node_modules`, `vendor`, `.git`, `dist`, `.idea` PLUS common ecosystem noise `.venv`, `venv`, `__pycache__`, `target`, `build`, `bin`, `obj`, `.cache`.
- **D-11:** Exact-case name matching on all platforms — no case-insensitivity branching. Windows-created `Node_Modules` edge documented as acceptable.
- **D-12:** Root-scoped only (consistent with DETC-01); ignore list only affects the "folder contains nothing but ignored dirs" case → `LanguageUnknown`, nil error.

### Registry & Shared Helpers (FND-03, DETC-01, ROBT-02) — agent discretion
- Ordered private detector registry with `func(folder string) (ProjectData, bool)` contract — first match wins. Detector order from research: Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP (empty registry in Phase 10; entries added phases 11-13).
- Shared `readManifest` — 1 MB size cap, BOM strip, no panics on pathological input (precedent: singleflight `PanicError` recovery pattern from v1.6 spikes).

### the agent's Discretion
- Registry data structure (slice of funcs vs. small registry type), readManifest internals, file layout within `project_probe/` (snake_case, doc.go), ignore-list data structure (map vs slice) — follow repo conventions (CONVENTIONS.md).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone requirements & roadmap
- `.planning/REQUIREMENTS.md` — v1.7 requirements; Phase 10 covers FND-01, FND-02, FND-03, DETC-01, DETC-09, DATA-01, ROBT-01, ROBT-02, ROBT-04
- `.planning/ROADMAP.md` §Phase 10 — goal, success criteria (5), depends-on, requirements mapping
- `.planning/STATE.md` — accumulated research decisions (never-fail contract, manifest-first cascade, detector order)

### Spike findings
- `.opencode/skills/spike-findings-go/SKILL.md` — repo-wide validated patterns; panic-recovery precedent relevant to readManifest

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md` — naming, error handling (sentinel errors, %w wrapping), doc.go requirements, declaration order (decorder), coverage gates
- `.planning/codebase/STRUCTURE.md` — package layout rules (snake_case dirs, package-name concatenation like `projectprobe`), where new packages go, README index
- `.planning/codebase/STACK.md` — Go 1.26.4, stdlib-only, testify, toolchain constraints

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `path_tools/` — `DirExists`, `FileExists` helpers usable inside detectors; `root_folder.go` shows go.mod-upward search precedent
- `project_detector/` (deprecated) — DELETED in this phase (FND-01); its detector files are a reference for what NOT to do (missing gs-dev dep, broken build)

### Established Patterns
- Sentinel errors + `%w` wrapping (all packages) — D-01/D-02 follow this
- Panic-recovery wrapper for external-input parsing (v1.6 singleflight `PanicError`) — readManifest must never panic
- `doc.go` + godoclint-enforced package docs — FND-02 requires the never-fail contract documented
- `example_test.go` per package, testify assertions, 95% coverage gate (`make coverage-quick`)

### Integration Points
- New top-level `project_probe/` directory at module root (`github.com/guionardo/go/project_probe`)
- README.md package index table row added at Phase 14 polish (per roadmap), not this phase
- CI runs `go build ./...` — FND-01 (project_detector deletion) keeps it green

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the discussion — decisions above are the contract. Open to standard approaches for registry internals.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (Framework detection, TOML dependency, monorepo detection already tracked in v2 requirements/Out of Scope.)

</deferred>

---

*Phase: 10-Package Foundation — API Contract + Repo Cleanup*
*Context gathered: 2026-09-28*