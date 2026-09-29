---
phase: 14-semantics-hardening-release-polish
verified: 2026-09-29T12:04:44Z
status: passed
score: 15/15 must-haves verified
covered_files:
  - .planning/phases/14-semantics-hardening-release-polish/14-01-PLAN.md
  - .planning/phases/14-semantics-hardening-release-polish/14-01-SUMMARY.md
  - .planning/phases/14-semantics-hardening-release-polish/14-02-PLAN.md
  - .planning/phases/14-semantics-hardening-release-polish/14-02-SUMMARY.md
  - .planning/phases/14-semantics-hardening-release-polish/14-03-PLAN.md
  - .planning/phases/14-semantics-hardening-release-polish/14-03-SUMMARY.md
  - .planning/phases/14-semantics-hardening-release-polish/14-04-PLAN.md
  - .planning/phases/14-semantics-hardening-release-polish/14-04-SUMMARY.md
  - .planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md
  - .planning/phases/14-semantics-hardening-release-polish/14-REVIEW.md
  - .planning/phases/14-semantics-hardening-release-polish/14-REVIEW-DISPOSITION.md
  - project_probe/fuzz_test.go
  - project_probe/readme.go
  - project_probe/readme_test.go
  - project_probe/detect_csharp.go
  - project_probe/detect_csharp_test.go
  - project_probe/detect_java_kotlin.go
  - project_probe/detect_java_kotlin_test.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - release/update_test.go
  - README.md
  - project_probe/testdata/fuzz/FuzzJSONManifest/bom-prefixed.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/composer.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/duplicate-keys.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/malformed-truncated.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/package.json
  - project_probe/testdata/fuzz/FuzzJSONManifest/version-null.json
  - project_probe/testdata/fuzz/FuzzTOMLManifest/cargo-real.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/cargo-workspace.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-dynamic.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-poetry.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/pyproject-real.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/toml-6quote.toml
  - project_probe/testdata/fuzz/FuzzTOMLManifest/toml-bracket-string.toml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-oldstyle.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-sdk.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/csproj-whitespace-only.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/pom-child.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/pom-pretty.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/xml-truncated.xml
  - project_probe/testdata/fuzz/FuzzXMLManifest/xml-whitespace-only.xml
covered_digest: "v2:sha256:325f7f1fffcd79d65b8df413662441a5db57c3328cf9a56108188dc8a6c1b8e0"
behavior_unverified: 0
overrides_applied: 2
overrides:
  - must_have: "Anti-feature audit recorded in 14-AUDIT.md (D-03/ROBT-05): the five grep pattern families across project_probe/ all return zero matches"
    reason: "Family-5a's single match is badge-line stripping — isBadgeLine's strings.ReplaceAll(line, \"!\", \"\") at readme.go:157 operates on README content lines (badge detection, 11-WR-01 fix 7f16193), never on Version values. Same category as the already-recorded decode-hygiene nuance (TrimSpace). The audit was refreshed (commit 7b52713) to disclose the exception; the verifier's re-run matches exactly that one disclosed line and nothing else."
    accepted_by: "user (Guionardo Furlan)"
    accepted_at: "2026-09-29T12:00:00Z"
  - must_have: "No version normalization anywhere in project_probe — no string-replace/prefix/suffix stripping on versions (14-04 prohibition gate)"
    reason: "Prohibition substantively honored: the only strings.Replace* call in the package operates on README badge lines in isBadgeLine, and all 7 Version assignments are raw/verbatim/'' (TrimSpace decode hygiene only, documented as audit nuance (b)). The prohibition's verification grep now matches only the disclosed badge-stripping line, recorded as a documented exception in 14-AUDIT.md (preferred closure path)."
    accepted_by: "user (Guionardo Furlan)"
    accepted_at: "2026-09-29T12:00:00Z"
re_verification:
  previous_status: gaps_found
  previous_score: 14/15
  gaps_closed:
    - "Anti-feature audit recorded in 14-AUDIT.md: family-5a row refreshed (commit 7b52713) to disclose the isBadgeLine strings.ReplaceAll badge-stripping exception at readme.go:157; verifier re-run of the verbatim grep matches exactly that one disclosed line and nothing else; families 1-4, 5b, 6 print zero matches"
    - "No version normalization prohibition gate: exception documented in the audit (preferred closure path); substance re-verified — isBadgeLine called only at readme.go:89 on README lines, all 7 Version assignments raw/verbatim/''"
  gaps_remaining: []
  regressions: []
---

# Phase 14: Semantics, Hardening, and Release Polish — Verification Report

**Phase Goal:** Version semantics are resolved across all 7 detectors; the package is hardened (fuzz, fixtures), documented, and meets coverage thresholds.
**Verified:** 2026-09-29T12:04:44Z
**Status:** passed
**Re-verification:** Yes — gap closure (family-5a audit evidence) via user-accepted override + audit refresh

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Version always raw manifest string across all 7 detectors (go.mod toolchain floor, Maven parent inheritance, Cargo version.workspace/pyproject dynamic → empty; never normalized, never fabricated) | ✓ VERIFIED | Code-read of all 7 Version assignments (detect_go.go:26/63 raw directive; detect_python.go:29; detect_rust.go:29 workspace → ""; detect_javascript.go:34; detect_php.go:33 verbatim; detect_csharp.go:110-113 Version→VersionPrefix, never 1.0.0; detect_java_kotlin.go:108-111 parent inheritance, never Super POM 4.0.0). Behavioral probe (go run): go.mod → "1.22.5" (toolchain line ignored), pyproject dynamic → "", Cargo version.workspace → "", padded `<Version> 1.2.3 </Version>` → "1.2.3" (trimmed, decode hygiene). doc.go contract (lines 55-76) matches. Re-verified this round: Version assignments unchanged (git log: no code commits to project_probe/ since previous verification). |
| 2 | SC2 — Package-wide audit confirms anti-features absent: no build-tool execution, no network, no symlink following, no version normalization in any code path | ✓ VERIFIED | Re-ran all 7 audit greps on the current tree: families 1-4, 5b, 6 → zero matches (exit 1). Family 5a → exactly one match (readme.go:157 `strings.ReplaceAll(line, "!", "")`) — badge-line stripping in isBadgeLine; Version values never flow through it (single call site readme.go:89 receives README content lines). No version normalization exists in any code path. |
| 3 | SC3 — Fuzz targets seeded with real manifests run under normal `go test`; malformed JSON/XML/TOML never panic, never crash the probe | ✓ VERIFIED | Regression re-run: `go test ./project_probe/...` → 284 passed (fuzz seed mode + fold-in rows). No code changes since prior full verification (34 seed cases + 284 suite). |
| 4 | SC4 — Package ships complete: make coverage-quick passes, doc.go contract finalized, README package index row added, go vet and lint clean | ✓ VERIFIED | Prior verification: `make coverage-quick` → PASS (file 70% ✓, pkg 80% ✓, total 80.7% ✓), go vet/build/GOOS=windows vet clean, doc.go §Version semantics present, README row :32 + section :312, no debt markers. No code changed since (git log empty for project_probe/ + release/ after 2026-09-29T12:30:00Z). |
| 5 | 14-01 — FuzzJSONManifest/FuzzTOMLManifest/FuzzXMLManifest exist with 3-5 f.Add seeds, bodies exercise full parse paths (D-05), results ignored | ✓ VERIFIED | fuzz_test.go: 3 targets, 11 seeds, D-05 paths, 0 assertions/testify/t.Parallel (prior verification; file unmodified since). |
| 6 | 14-01 — 20-file corpus under testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/, `go test fuzz v1`-encoded, real-world + pinned malformed shapes, not fixture byte-copies | ✓ VERIFIED | All 20 files verified header + single `[]byte` line, real + malformed shapes (prior verification; corpus unmodified since). |
| 7 | 14-02 — readme.go plain-badge (11 WR-01) + multi-line HTML comment (11 WR-02) fixes with matrix rows | ✓ VERIFIED | readme.go:157 two-clause form with `![`-guard; inComment state machine + `-->` opener guard; 6 matrix rows; tests pass (file unmodified since prior verification — 353 tests green this round). |
| 8 | 14-02 — XML element text trimmed at decode time in both XML detectors (13 WR-01), exact-name guard (13 IN-03), IN-01 decode comments | ✓ VERIFIED | detect_csharp.go: 5× TrimSpace + exact-name guard + IN-01 wording; detect_java_kotlin.go: 5× TrimSpace + IN-01 wording (prior verification; files unmodified since). |
| 9 | 14-02 — TestRunDetectors_EmptyRegistry injects explicit empty slice, probes non-colliding folder | ✓ VERIFIED | registry_test.go:23/26 explicit empty slice + non-colliding probe (prior verification; file unmodified since). |
| 10 | 14-03 — release/update.go file coverage ≥70%; make coverage-quick passes; error paths return errors, never panics; update.go unmodified | ✓ VERIFIED | 69 release tests pass this round (`go test ./release/...` included in 353); update.go byte-identical (last modified Phase 4); error-path rows verified previously. |
| 11 | 14-04 — doc.go Version-semantics contract section (D-02): six rules + placeholders raw + decode-hygiene note | ✓ VERIFIED | doc.go:55-76 contract present; go build + vet clean (prior verification; file unmodified since). |
| 12 | 14-04 — README package index row + per-package section (D-10) | ✓ VERIFIED | README.md:32 row + :312 section (prior verification; file unmodified since). |
| 13 | 14-04 — Anti-feature audit recorded in 14-AUDIT.md: the five grep pattern families across project_probe/ all return zero matches | ✓ VERIFIED (PASSED — override) | Audit refreshed (commit 7b52713): family-5a row now discloses the single exception — `strings.ReplaceAll(line, "!", "")` badge stripping at readme.go:157 (11-WR-01 fix) — with line-23 prose explaining it is NOT version normalization. Verifier re-ran the verbatim family-5a grep: **exactly one match, the disclosed line, nothing else**. Families 1-4, 5b, 6 re-ran: zero matches each. Audit claim now reproducible on the verifier's tree. Deviation from the literal "all zero matches" wording accepted by user (override). |
| 14 | 14-04 — No version normalization anywhere in project_probe — no string-replace/prefix/suffix stripping on versions (prohibition gate) | ✓ VERIFIED (PASSED — override) | Prohibition substantively honored: isBadgeLine (sole strings.Replace* user) is called only at readme.go:89 on README content lines; all 7 Version assignments raw/verbatim/''. The verification grep's single match is the disclosed badge-stripping exception, documented in 14-AUDIT.md (preferred closure path from prior gap). User-accepted override records the proxy/statement mismatch. |
| 15 | 14-04 — Deferred dispositions ledger (D-08): CR-01 + panic logging with rationale, residual INFO findings accepted not-in-scope | ✓ VERIFIED | 14-AUDIT.md §2 ledger intact (prior verification; audit refresh touched only §1 family-5a evidence). |

**Score:** 15/15 truths verified (13 VERIFIED + 2 PASSED via user-accepted override)

### Deferred Items

None — no gaps remain; nothing deferred to later phases (Phase 14 is the final v1.7 phase).

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ----------- | ------ | ------- |
| `project_probe/fuzz_test.go` | 3 fuzz targets, D-05 paths, no assertions | ✓ VERIFIED | 11 seeds; seed-mode green (284 suite) |
| `project_probe/testdata/fuzz/` (20 files) | go test fuzz v1 encoded corpus | ✓ VERIFIED | header + 1 value line each |
| `project_probe/readme.go` | badge + comment fixes | ✓ VERIFIED | ReplaceAll form at :157; inComment state machine |
| `project_probe/detect_csharp.go` | trim ×5, exact-name guard, IN-01 | ✓ VERIFIED | lines 86-98, 28, 74 |
| `project_probe/detect_java_kotlin.go` | trim ×5, IN-01 | ✓ VERIFIED | lines 101-112, 95 |
| `project_probe/registry_test.go` | empty-slice injection | ✓ VERIFIED | line 23 |
| `release/update_test.go` | 8 new rows, WR-02/IN-02 fixes | ✓ VERIFIED | 491 lines; 69 release tests pass |
| `project_probe/doc.go` | Version-semantics contract | ✓ VERIFIED | lines 55-76 |
| `README.md` | row + section | ✓ VERIFIED | :32, :312 |
| `.planning/.../14-AUDIT.md` | audit evidence, ledger, gates | ✓ VERIFIED | family-5a exception disclosed (commit 7b52713); reproducible on verifier's tree |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Fuzz body | detector parse path | t.TempDir → os.WriteFile → detectX → readManifest → decode → chains | WIRED | exercised in seed mode; no-panic invariant holds |
| Corpus files | internal/fuzz seed load | testdata/fuzz/<Target>/ → go test | WIRED | 20 files execute as named seed cases |
| readme.go isBadgeLine | firstRealParagraph | `![`-guard + two-clause rule | WIRED | rows pin thresholds; sole call site :89 on README lines |
| XML detectors | chains | TrimSpace per field before chain resolution | WIRED | whitespace-only no longer blocks fallbacks |
| CheckForUpdate | module derivation | getCurrentModule → url.Parse → words[1]/words[2] | WIRED | handler asserts /repos/guionardo/go/releases/latest |
| doc.go contract | detector implementations | Version-semantics claims | WIRED | matches all 7 assignments (code-read) |
| README row | section | #package-project_probe anchor | WIRED | row :32 → section :312 |
| 14-AUDIT.md family-5a row | actual grep output | verbatim command re-run | WIRED | matches exactly the one disclosed line (readme.go:157); families 1-4, 5b, 6 zero |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detect_go.go Version | goVersion | parseGoMod → go.mod `go` directive | Yes — behavioral probe returned "1.22.5" (toolchain line excluded) | ✓ FLOWING |
| detect_python.go Version | fields["version"] | readTOMLSection → [project]/[tool.poetry] | Yes — dynamic → "" (probe) | ✓ FLOWING |
| detect_rust.go Version | fields["version"] | readTOMLSection → [package] | Yes — version.workspace → "" (probe) | ✓ FLOWING |
| detect_csharp.go Version | version/versionPrefix | xml.Unmarshal → TrimSpace → chain | Yes — padded 1.2.3 → "1.2.3" (probe) | ✓ FLOWING |
| detect_java_kotlin.go Version | pom.Version/Parent.Version | xml.Unmarshal → TrimSpace → chain | Yes — parent inheritance tested | ✓ FLOWING |
| readme.go Description | firstRealParagraph | readManifest → README candidates | Yes — badge/comment noise excluded; badge-stripping operates on README lines only, never Versions | ✓ FLOWING |
| release CheckForUpdate | rel | githubAPIBase → mock server JSON | Yes — 8 rows cover derivation + error paths | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Regression: project_probe + release suites (no code changed since prior verification) | `go test ./project_probe/... ./release/...` | 353 passed (284 + 69) | ✓ PASS |
| Anti-feature family 5a (re-run, verbatim) | `grep -rnE 'ParseVersion\|semver\|version\.Compare\|strings\.(Replace\|TrimPrefix\|TrimSuffix)' project_probe/` | exactly 1 match — readme.go:157, the disclosed line | ✓ PASS |
| Anti-feature families 1-4, 5b, 6 (re-run, verbatim) | 6 greps | zero matches each (exit 1) | ✓ PASS |
| Audit refresh committed | `git show 7b52713 --stat` | docs(14): refresh audit family-5a evidence; 14-AUDIT.md +2 -2 only | ✓ PASS |
| Code unchanged since prior verification | `git log --since 2026-09-29T12:30:00Z -- project_probe/ release/` | empty | ✓ PASS |

Prior full behavioral evidence stands (no code drift): 34 fuzz seed cases, 46 fold-in rows, DATA-03 end-to-end probe (go: "1.22.5"; dynamic: ""; workspace: ""; padded: "1.2.3"), coverage-quick PASS, vet/build/Windows-vet clean, `go test ./release/... -race` clean, update.go byte-identical.

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| Anti-feature families 1-4, 5b, 6 | verbatim greps from 14-AUDIT.md | zero matches each (re-run this round) | PASS |
| Anti-feature family 5a | verbatim grep from 14-AUDIT.md | 1 match — readme.go:157, exactly the disclosed exception | PASS (audit record now accurate) |
| Fuzz flag prohibition | `grep -rnE 'fuzz=' .github/ Makefile` | nothing | PASS |
| Dependency stability | `git diff HEAD -- go.mod go.sum` | 0 lines | PASS |
| update.go untouched | `git log -- release/update.go` | last change Phase 4 | PASS |
| Corpus format | `head -1` + `grep -c '^\[\]byte'` per file | header + 1 value line each | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DATA-03 | 14-02, 14-04 | Version: raw manifest string, empty when absent/dynamic, never fabricated | ✓ SATISFIED | All 7 assignments raw/"" (code-read + behavioral probe); doc.go contract; REQUIREMENTS.md maps DATA-03 → Phase 14 Complete |
| ROBT-05 | 14-01, 14-02, 14-03, 14-04 | No build-tool execution, no network, no symlink following, no version normalization (anti-features enforced) | ✓ SATISFIED | 6/7 grep families zero matches (re-run); family-5a single match is disclosed badge stripping (readme.go:157) — not version normalization, exception recorded in refreshed audit; fuzz no-panic; error-not-panic rows; REQUIREMENTS.md maps ROBT-05 → Phase 14 Complete |

No orphaned requirements: only DATA-03 and ROBT-05 map to Phase 14; both claimed by plans and satisfied.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `project_probe/readme.go` | 157 | `strings.ReplaceAll(line, "!", "")` trips the family-5a proxy grep | ℹ️ Info | Proxy/statement mismatch only — no version normalization in any code path; exception disclosed in 14-AUDIT.md and accepted via user override |
| `.planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md` | 19-20 | Cosmetic typos in the refreshed rows ("land ed", "adjacen t" — word splits) | ℹ️ Info | No impact on evidence accuracy; row 19's disclosure and line-23 prose are unambiguous |
| `.planning/phases/14-semantics-hardening-release-polish/14-AUDIT.md` | 23 | Prose references "the 14-REVIEW-DISPOSITION.md ledger ... document this as an accepted, evidence-refreshed override" — the ledger itself records WR-01 as "fixed" without explicit override wording | ℹ️ Info | The exception is fully disclosed in the audit's own row 5a + line 23; this refreshed VERIFICATION.md now carries the override record, making the reference accurate |

No TBD/FIXME/XXX markers, no stubs, no hollow props, no console-only implementations in phase-14 touched files.

### Overrides Applied (user-accepted)

| Must-Have | Reason | Accepted By | Accepted At |
| --------- | ------ | ----------- | ----------- |
| "Anti-feature audit ... five grep pattern families ... all return zero matches" | Family-5a's single match is badge-line stripping (isBadgeLine strings.ReplaceAll on README lines at readme.go:157, 11-WR-01 fix 7f16193) — never on Version values; audit refreshed (7b52713) to disclose it; verifier re-run matches exactly that one line | user (Guionardo Furlan) | 2026-09-29 |
| "No version normalization anywhere in project_probe" | Prohibition substantively honored — sole strings.Replace* call operates on README badge lines; all 7 Version assignments raw/verbatim/''; exception documented in audit (preferred closure path) | user (Guionardo Furlan) | 2026-09-29 |

### Human Verification Required

None. Every behavior-dependent truth is exercised by a passing test or a direct behavioral run (no code changed since the prior full verification). The family-5a evidence question was resolved by explicit user acceptance of the override ("Accept override, refresh audit") — a human decision, recorded in the overrides frontmatter.

### Gaps Summary

**0 gaps.** Both prior gaps closed:

1. **Audit evidence staleness (family 5a)** — CLOSED. 14-AUDIT.md refreshed in commit `7b52713` ("docs(14): refresh audit family-5a evidence (badge-stripping override)"): the family-5a row now reads "zero matches — except `strings.ReplaceAll` badge stripping in `isBadgeLine` (readme.go:157, the 11-WR-01 fix...)", and line 23 explains the exception is NOT version normalization. Verifier re-ran the verbatim grep: **exactly one match — the disclosed line — nothing else**; families 1-4, 5b, 6 print zero. The audit's reproducibility claim now holds on the verifier's tree.
2. **Prohibition gate proxy mismatch** — CLOSED. The prohibition is substantively honored (isBadgeLine's only call site is readme.go:89 on README content lines; all 7 Version assignments raw/verbatim/''); the verification grep's single match is the documented badge-stripping exception, recorded in the audit per the preferred closure path. User accepted the override.

Substance unchanged and re-confirmed: no version normalization exists in any code path; 353 tests pass (284 project_probe + 69 release); no code commits to project_probe/ or release/ since the prior verification.

---

_Verified: 2026-09-29T12:04:44Z_
_Verifier: the agent (gsd-verifier)_