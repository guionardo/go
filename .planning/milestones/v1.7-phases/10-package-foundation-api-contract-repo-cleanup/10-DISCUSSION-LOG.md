# Phase 10: Package Foundation — API Contract + Repo Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-28
**Phase:** 10-Package Foundation — API Contract + Repo Cleanup
**Areas discussed:** Error contract edge cases, Language type shape, Folder field semantics, Ignore-list composition

---

## Error contract edge cases

| Option | Description | Selected |
|--------|-------------|----------|
| Sentinel errors | ErrFolderNotFound + ErrNotDirectory + ErrFolderRead; errors.Is checkable; matches repo convention | ✓ |
| Plain wrapped errors | Wrap underlying OS error, no exported sentinels | |
| Typed ProbeError struct | Rich typed error with Folder/Op/Err fields — over-engineered for a utility | |

**User's choice:** Sentinel errors (then refined to 3 sentinels + fallback: ErrFolderNotFound, ErrNotDirectory, ErrPermissionDenied, other → plain wrapped)

| Option | Description | Selected |
|--------|-------------|----------|
| 3 sentinels + fallback | ENOENT/ENOTDIR/EACCES mapped; anything else wrapped | ✓ |
| 2 sentinels | not-found + read-failure only | |
| 1 sentinel only | ErrFolderNotFound; all else plain wrapped | |

**User's choice:** 3 sentinels + fallback

| Option | Description | Selected |
|--------|-------------|----------|
| Wrap with folder path | fmt.Errorf %w with failing path; errors.Is still matches | ✓ |
| Bare sentinel only | No path context | |
| Raw OS error | Leak OS-specific error shapes | |

**User's choice:** Wrap with folder path. Also accepted: manifest-read failures inside detectors degrade to Unknown (never-fail best-effort).
**Notes:** Requirements already locked missing/unreadable → error; empty/unrecognized → LanguageUnknown nil error — not re-asked.

---

## Language type shape

| Option | Description | Selected |
|--------|-------------|----------|
| Typed constants | type Language string + exported constants; zero-cost, switchable | ✓ |
| Plain string field | No constants; magic strings | |
| Typed + Stringer/JSON | fmt.Stringer + marshal — YAGNI | |

**User's choice:** Typed constants

| Option | Description | Selected |
|--------|-------------|----------|
| PascalCase IDs + display values | LanguageGo = "Go", LanguageCSharp = "C#/.NET" | ✓ |
| Lowercase values | "go", "csharp" — machine-friendly, loses display form | |
| Verbatim long identifiers | LanguageCSharpDotNET — clunky | |

**User's choice:** PascalCase IDs + display values

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit "unknown" value | LanguageUnknown = "unknown" — reads well in logs/JSON | ✓ |
| Zero-value "" | Natural default but ambiguous in JSON | |

**User's choice:** Explicit "unknown" value
**Notes:** Extensible for v2 REFN-01 (typescript value).

---

## Folder field semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Clean, keep relative | filepath.Clean(as-given); deterministic tests; Windows-safe | ✓ |
| Always absolute | filepath.Abs — machine-dependent results | |
| Raw as-given | Unusable comparisons | |

**User's choice:** Clean, keep relative

| Option | Description | Selected |
|--------|-------------|----------|
| Clean once, reuse | One canonical path for stat/read/errors/Folder | ✓ |
| Clean only for Folder | Mismatched error text vs data | |

**User's choice:** Clean once, reuse

---

## Ignore-list composition

| Option | Description | Selected |
|--------|-------------|----------|
| 5 core + common extras | node_modules, vendor, .git, dist, .idea + .venv, venv, __pycache__, target, build, bin, obj, .cache | ✓ |
| Exact 5 only | Minimal; noise-only folders report Unknown less precisely | |
| All dot-dirs | Over-ignores legitimate dot-dir projects | |

**User's choice:** 5 core + common extras

| Option | Description | Selected |
|--------|-------------|----------|
| Exact-case match | Deterministic all platforms; Windows Node_Modules edge documented | ✓ |
| Case-insensitive all platforms | Catches variants, costs ToLower per entry | |
| Windows-only insensitive | Correct per-platform, adds CI matrix burden | |

**User's choice:** Exact-case match

---

## the agent's Discretion

- Registry data structure (slice of detector funcs vs. registry type) — `(ProjectData, bool)` contract locked by requirements
- readManifest internals (size cap mechanism, BOM strip scope)
- File layout within `project_probe/` — repo conventions apply (CONVENTIONS.md)
- Ignore-list data structure — map vs slice

## Deferred Ideas

None — discussion stayed within phase scope. v2 backlog (framework detection, TOML dep, REFN-01 typescript) already tracked in REQUIREMENTS.md.
