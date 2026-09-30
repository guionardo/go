# Phase 13: XML Detectors — C#/.NET + Java/Kotlin - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-29
**Phase:** 13-XML Detectors — C#/.NET + Java/Kotlin
**Areas discussed:** XML parsing approach, .csproj name/version extraction, pom.xml parent inheritance, settings.gradle fallback, registry wiring

---

## Discussion Mode

Auto mode (transition auto-advance from Phase 12, yolo). All gray areas auto-selected and resolved with recommended defaults in a single pass. No interactive questions.

## XML Parsing (DETC-07)

[auto] Q: "How to parse .csproj/pom.xml namespace-agnostically?" → **Selected:** `encoding/xml` with XMLName local-name matching (namespace-agnostic by local name, not URI) — stdlib-only
- BOM satisfied at readManifest boundary

## .csproj Detector (DETC-07)

[auto] Q: "Name/version from .csproj?" → **Selected:** Name chain AssemblyName → RootNamespace → folder base; Version: Version → VersionPrefix → empty; root-scoped only (SC4)

## Java/Kotlin Detector (DETC-08)

[auto] Q: "Parent version inheritance?" → **Selected:** pom.xml primary — Name `<name>` → `<artifactId>` → folder base; Version `<version>` → `<parent><version>` inheritance (single level); settings.gradle `rootProject.name` fallback only when no pom.xml (SC3)
[auto] Q: "Java vs Kotlin value?" → **Selected:** LanguageJava for both (Kotlin-distinct value deferred to v2 REFN-02)

## Registry Wiring

[auto] Q: "Slot placement?" → **Selected:** detectCSharp at index 2, detectJavaKotlin at index 5 — completes 7-of-7 literal; detector order confirmed (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP)

## Deferred Ideas

- Distinct Kotlin Language value — v2 (REFN-02)
- Lockfile secondary confirmation — v2 (REFN-03)
- Multiple .csproj selection — v2 (REFN-04)