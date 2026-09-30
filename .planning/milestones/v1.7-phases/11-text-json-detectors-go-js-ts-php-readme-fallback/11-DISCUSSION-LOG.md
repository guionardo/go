# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-29
**Phase:** 11-Text/JSON Detectors — Go, JS/TS, PHP + README Fallback
**Areas discussed:** Go detector parsing, JS/TS detector details, PHP composer name handling, README fallback selection, README first-paragraph extraction

---

## Discussion Mode

Auto mode (transition auto-advance from Phase 10, yolo). All gray areas auto-selected and resolved with recommended defaults in a single pass. No interactive questions.

## Go Detector (DETC-02)

[auto] Q: "What's the go.mod name semantics?" → **Selected:** module path verbatim (raw string, not the last path segment) — matches go.mod module convention and DETC-02 "module → name"
- Version = `go` directive raw string, documented as toolchain floor (carried RESEARCH decision from STATE.md)
- Quoted module lines parse correctly; BOM handled by readManifest

## JS/TS Detector (DETC-03)

[auto] Q: "Which Language value for package.json?" → **Selected:** always `LanguageJavaScript`
- Distinct `typescript` value deferred to v2 (REFN-01)
- Malformed JSON → detector returns false (cascade continues, never-fail)
- `private: true` without version → empty Version

## PHP Detector (DETC-04)

[auto] Q: "Composer name format?" → **Selected:** full `vendor/package` string (ecosystem identifier, consistent with Go module-path reporting)

## README Fallback (DATA-04)

[auto] Q: "Which README files count?" → **Selected:** ordered candidate list `README.md` → `README.rst` → `README` (deterministic, exact-case)
[auto] Q: "What does 'first real paragraph' skip?" → **Selected:** image-only badge lines, TOC-style link lists, rst underline headings (===/---), blank lines — then first consecutive non-blank text block

## Deferred Ideas

- Distinct `typescript` Language value — v2 backlog (REFN-01)
