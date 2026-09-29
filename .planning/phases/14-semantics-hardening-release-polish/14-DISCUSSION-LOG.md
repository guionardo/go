# Phase 14: Semantics, Hardening, and Release Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-29
**Phase:** 14-Semantics, Hardening, and Release Polish
**Areas discussed:** version-semantics audit, fuzz targets, anti-feature audit, review-disposition fold-in, release polish

---

## Discussion Mode

Auto mode (transition auto-advance from Phase 13, yolo). All gray areas auto-selected and resolved with recommended defaults in a single pass. No interactive questions.

## Version Semantics (DATA-03)

[auto] Q: "How to document/resolve version rules?" → **Selected:** package-wide audit + doc.go contract section — raw strings everywhere, toolchain-floor documented, `version.workspace`/`dynamic` → empty (already structural), never normalized/fabricated

## Fuzz Targets (SC3)

[auto] Q: "Which fuzz targets?" → **Selected:** 3 Fuzz functions (JSON/TOML/XML manifests) seeded with real manifest corpus, running under normal `go test` — malformed inputs never panic

## Anti-Feature Audit (ROBT-05)

[auto] Q: "How to enforce ROBT-05?" → **Selected:** systematic grep-verified audit across all detectors + readers (no os/exec, net/http, EvalSymlinks, WalkDir, version normalization)

## Review Dispositions

[auto] Q: "Which open findings fold in?" → **Selected:** readme.go plain-badge + HTML-comment fixes (11 WR-01/02), XML element trim (13 WR-01), registry test hygiene (WR-02), stale comments; deferred: 10 CR-01 (Windows errno — needs Windows CI evidence), panic logging

## Coverage Gate (SC4)

[auto] Q: "How does make coverage-quick pass?" → **Selected:** add missing release/update.go tests (pre-existing 68.9% known-red is the ONLY repo-wide gate failure; SC4 requires the gate to pass)

## Release Polish

[auto] Q: "What ships?" → **Selected:** README package index row, doc.go contract finalized, go vet + lint clean, milestone complete

## Deferred Ideas

- v2 backlog: framework detection, full TOML dep, typescript/kotlin values, lockfile confirmation, multi-csproj rule
- Phase 10 CR-01 (Windows errno) + WR-02 (panic logging) — deferred with rationale