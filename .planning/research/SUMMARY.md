# Research Summary: Go Utility Monorepo Organization

**Domain:** Go utility libraries / developer toolkits  
**Researched:** 2026-07-21  
**Overall confidence:** HIGH

## Executive Summary

Go utility monorepos follow six distinct organizational patterns, each with clear tradeoffs around discoverability, versioning independence, and maintenance overhead. The `github.com/guionardo/go` project uses the **flat package-per-utility** pattern (Pattern 2), which is appropriate for its current scale of ~12 independent packages. This research identifies that the current structure is well-aligned with community norms (matching hashicorp's go-* approach within a single module for operational efficiency) and has clear evolution paths as it grows.

## Key Findings

- **Stack:** Go 1.26 single-module monorepo with 12 flat top-level packages, minimal deps, stdlib-first
- **Architecture:** Flat package-per-utility with sub-package structure only in `config/` — no cross-package coupling except `httptest_mock → flow, reflect_tools`
- **Critical pitfall:** At ~15–20 packages, discoverability degrades without naming conventions or categorization; the `config/` sub-packages being public (not under `internal/`) creates a coupling surface that should be addressed before the API stabilizes

## Implications for Roadmap

Based on research, suggested phase structure:

1. **Sub-package encapsulation** — Move `config/environment`, `config/profile`, `config/merger`, `config/validation` under `config/internal/` to prevent external coupling
   - Avoids: Public sub-package coupling pitfall
   - Complexity: Low (mechanical refactor, no API changes for root config package)

2. **Package naming standardization** — Establish and apply a consistent naming convention for packages (singular vs compounded, prefix conventions for related groups)
   - Avoids: Alphabetical sprawl at root
   - Complexity: Low (rename decisions, doc updates)

3. **Contributor package template** — Create a `_template/` directory with the standard file layout, conventions, and checklist for adding new packages
   - Addresses: Consistency enforcement as collection grows
   - Complexity: Low

4. **Multi-module evaluation** — Only if version coupling becomes a real pain point
   - Avoids: Premature complexity

**Phase ordering rationale:**
- Encapsulation first because it's a correctness/API stability issue
- Naming second because it's cosmetic but impacts every new package
- Template third because it institutionalizes the conventions
- Multi-module last (if ever) because it's the most disruptive change

**Research flags for phases:**
- Phase 1 (internal move): Low risk — mostly mechanical, but need to verify no external consumers rely on the sub-package import paths
- Phase 4 (multi-module): Needs significant additional research on Go workspace (`go.work`) tooling

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Patterns identified | HIGH | Six distinct patterns confirmed across 10+ examined repos |
| Current project fit | HIGH | Pattern 2 matches scale and purpose exactly |
| Evolution recommendations | MEDIUM | Growth trajectory estimates are projections, not measured |
| Multi-module evaluation need | MEDIUM | Only two real-world examples (golang.org/x, tailscale) at this scale |
| Sub-package encapsulation priority | HIGH | Go official guidance explicitly recommends `internal/` for this case |

## Gaps to Address

- No data on actual external importers of `config/environment` etc. — needed before deciding to move to `internal/`
- No performance benchmark comparison of single-module vs multi-module for `go get` speed at scale
- Package naming conventions across the broader Go ecosystem are inconsistent — this is a matter of team preference, not correctness
