# Phase 12: TOML Subset + Python/Rust Detectors - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-29
**Phase:** 12-TOML Subset + Python/Rust Detectors
**Areas discussed:** TOML-subset reader scope, Python detector fields, Rust detector fields, registry slot wiring

---

## Discussion Mode

Auto mode (transition auto-advance from Phase 11, yolo). All gray areas auto-selected and resolved with recommended defaults in a single pass. No interactive questions.

## TOML-Subset Reader (ROBT-03)

[auto] Q: "Which TOML features must the ~80-line reader support?" → **Selected:** section headers `[name]`, bare/quoted keys, `=` separator, quoted scalar values, `#` comments, blank lines
- **Strict degrade-to-empty** on legal-but-unsupported TOML (multiline strings, dotted keys, inline tables, arrays) — never panic, never partial data (Phase 12 blocker)
- Unexported `readTOMLSection(content, section) map[string]string` consuming readManifest output

## Python Detector (DETC-05)

[auto] Q: "Which pyproject fields?" → **Selected:** `[project]` PEP 621 name/version/description primary; legacy `[tool.poetry]` fallback; README fallback for description; folder-base name fallback

## Rust Detector (DETC-06)

[auto] Q: "Cargo.toml version semantics?" → **Selected:** `[package]` name/version/description; `version.workspace = true` → empty Version (never fabricated)

## Registry Wiring

[auto] Q: "Slot placement?" → **Selected:** `detectPython` at index 1, `detectRust` at index 4 — completes 5 of 7 live slots (C#/.NET and Java/Kotlin remain for Phase 13)

## Deferred Ideas

- Full TOML dependency — v2 backlog (FRAM-02)
- Cargo workspace-root version resolution — not in scope, degrades to empty