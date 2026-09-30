# Phase 14: Semantics, Hardening, and Release Polish - Context

**Gathered:** 2026-09-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Final v1.7 phase: version semantics resolved and documented across all 7 detectors (DATA-03), package-wide anti-feature audit (ROBT-05), fuzz targets seeded with real manifests (never panic), review dispositions folded in, and release polish — README package index row, doc.go contract finalized, `make coverage-quick` passing, milestone complete.

Depends on Phase 13 (all 7 detectors live).

</domain>

<decisions>
## Implementation Decisions

### Version Semantics (DATA-03)
- **D-01:** Package-wide audit confirms Version is always the raw manifest string across all 7 detectors: go.mod `go` directive (toolchain floor, documented), Maven `<parent><version>` inheritance, Cargo `version.workspace` → empty, pyproject `dynamic` → empty. Never normalized, never fabricated.
- **D-02:** doc.go gains a final Version-semantics contract section documenting these rules (the single place downstream consumers read them).

### Anti-Feature Audit (ROBT-05)
- **D-03:** Systematic grep-verified audit across all 7 detectors + reader helpers: no os/exec (build-tool execution), no net/http (network), no EvalSymlinks/WalkDir (symlink following), no version normalization in any code path. Results documented in the verification report.

### Fuzz Targets (SC3)
- **D-04:** 3 Fuzz functions — FuzzJSONManifest (package.json/composer.json), FuzzTOMLManifest (pyproject/Cargo), FuzzXMLManifest (.csproj/pom.xml) — seeded with real manifest files as corpus, running under normal `go test` (fuzz functions execute seed corpus in test mode). Malformed inputs never panic, never crash the probe.
- **D-05:** Fuzz bodies exercise the detectors' parse paths (readManifest → decode → chains), not just the raw readers.

### Review Dispositions Fold-In
- **D-06:** Fix the open advisory findings that affect correctness: readme.go plain-badge form `![alt](url)` (11 WR-01), multi-line HTML comment preambles (11 WR-02), XML element text trimming (13 WR-01 — strings.TrimSpace per field before chain resolution).
- **D-07:** Fix test hygiene: TestRunDetectors_EmptyRegistry injects an explicit empty slice (12/13 WR-02), stale comments (IN-01), `.csproj` exact-name guard (13 IN-03).
- **D-08:** Deferred dispositions recorded explicitly with rationale: Phase 10 CR-01 (Windows errno mapping — requires Windows CI evidence, tracked), panic logging (10 WR-02) — noted for future milestone, not silently dropped.

### Coverage Gate (SC4)
- **D-09:** `make coverage-quick` must pass — add missing `release/update.go` tests (the pre-existing 68.9% < 70% file-threshold known-red is the ONLY repo-wide gate failure; SC4 explicitly requires the gate to pass; project_probe rows already pass). Small test addition in the release package closes it.

### Release Polish
- **D-10:** README.md gains the project_probe package index row (structure precedent: table of packages); doc.go never-fail contract finalized; go vet + golangci-lint clean (GOTOOLCHAIN=go1.26.4 workaround); milestone v1.7 complete.

### the agent's Discretion
- Fuzz corpus layout (testdata/fuzz dirs), exact test additions for release/update.go, README row wording. Follow repo conventions (CONVENTIONS.md) and prior-phase patterns.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone requirements & roadmap
- `.planning/REQUIREMENTS.md` — Phase 14 covers DATA-03, ROBT-05; FRAM-01/02, REFN-01..04 remain v2 backlog
- `.planning/ROADMAP.md` §Phase 14 — goal, 4 success criteria, depends-on Phase 13, is_last_phase true
- `.planning/STATE.md` — blockers (open review dispositions, verification debt), research decisions
- `.planning/PROJECT.md` — Key Decisions table, milestone state

### Prior phase contracts (all 7 detectors)
- `.planning/phases/10-package-foundation-api-contract-repo-cleanup/10-CONTEXT.md` + 10-REVIEW.md (CR-01 Windows errno, WR-02 panic logging — deferred dispositions)
- `.planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-CONTEXT.md` + 11-REVIEW.md (WR-01 plain badge, WR-02 HTML comments — fold-in fixes)
- `.planning/phases/12-toml-subset-python-rust-detectors/12-CONTEXT.md` + 12-REVIEW.md (CR-01 quote-aware skip states — FIXED via gap plan; WR-01 registry test — fold-in)
- `.planning/phases/13-xml-detectors-csharp-net-java-kotlin/13-CONTEXT.md` + 13-REVIEW.md (WR-01 XML trim, WR-02 registry test, IN-01..03 — fold-in)
- Each phase's `*-REVIEW-DISPOSITION.md` — the ledger of open findings

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/STACK.md`
- `README.md` — package index table (the row to extend)
- `.testcoverage-quick.yml` — the gate thresholds (file:70, pkg:80, total:75)
- `release/update.go` — the known-red file needing test additions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `project_probe/` — all 7 detectors, readManifest (1MB cap, BOM strip, FIFO gate), readmeDescription, readTOMLSection (quote-aware), registry 7-slot literal — the hardened package being polished
- `release/update.go` — the file needing coverage tests (68.9% < 70% — lines 60-63, 65-68, 70-73, 75-80, 91-92, 102-103, 112-113, 117-118, 130-131, 136-137, 141-144 uncovered)
- `README.md` — package index table with rows per package (precedent for the project_probe row)

### Established Patterns
- Never-fail contract, presence-match, TDD adaptation (RED verified-but-uncommitted), fuzz-friendly pure parsers
- Coverage gate: make coverage-quick (release/update.go is the ONLY remaining red)
- GOTOOLCHAIN=go1.26.4 workaround for golangci-lint

### Integration Points
- README.md package index table — add project_probe row
- doc.go — final Version-semantics + never-fail contract
- release/update.go tests — close the coverage gate
- All 7 detectors — fuzz targets wrap their parse paths

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the discussion — decisions above are the contract.

</specifics>

<deferred>
## Deferred Ideas

- Framework detection (FRAM-01), full TOML dep (FRAM-02), typescript value (REFN-01), kotlin value (REFN-02), lockfile confirmation (REFN-03), multi-csproj rule (REFN-04) — v2 backlog
- Phase 10 CR-01 (Windows errno verification) + WR-02 (panic logging) — deferred with rationale (D-08), tracked

</deferred>

---

*Phase: 14-Semantics, Hardening, and Release Polish*
*Context gathered: 2026-09-29*