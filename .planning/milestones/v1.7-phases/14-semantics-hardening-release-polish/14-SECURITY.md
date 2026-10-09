---
phase: "14"
slug: "semantics-hardening-release-polish"
status: draft
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-29"
---

# Phase 14 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Untrusted manifest content | project_probe reads arbitrary folder manifests (JSON/TOML/XML) | project metadata (name/version/description) — derived, not sensitive |
| Untrusted README content | firstRealParagraph parses arbitrary README text | Description string |
| Test-only GitHub API surface | release/update_test.go mocks GitHub Releases via httptest | none (mocked responses, closed-server URLs) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-14-01 | DoS | Fuzz targets / detector decode paths | high | mitigate | No-panic invariant: fuzz bodies call detectors, never assert results; callDetector recover net (registry.go); readManifest 1 MB cap + O_NONBLOCK FIFO gate; seeds run under plain go test in CI (284 tests green) | closed |
| T-14-02 | Tampering | Anti-feature regression in project_probe | high | mitigate | 14-AUDIT.md grep audit: os/exec, net/http, EvalSymlinks, WalkDir — zero matches (badge-stripping ReplaceAll disclosed as the one family-5a exception, operates on README lines never Version values) | closed |
| T-14-03 | DoS | XML entity expansion via crafted manifest | medium | mitigate (already) | encoding/xml strict mode resolves only the five predefined entities; no custom Entity/CharsetReader wiring (grep-verified zero matches) | closed |
| T-14-04 | DoS | Fuzz-corpus poisoning — malformed corpus breaks CI | medium | accept | MalformedCorpusError fails the build loudly (verified); corpus is repo-reviewed fixture content; go test well-formedness gate | closed |
| T-14-05 | Tampering | Fuzz bodies asserting detector bools → false crash reports | medium | mitigate | No require/assert/t.Fatal in f.Fuzz bodies (grep-verified); no-panic is the only invariant | closed |
| T-14-06 | Tampering | firstRealParagraph extraction on adversarial READMEs | medium | mitigate | inComment state machine consumes multi-line comment blocks entirely; "!["-presence guard + two-clause badge rule; matrix rows pin thresholds | closed |
| T-14-07 | Tampering | Whitespace-only XML element text blocking DATA-03 chains | medium | mitigate | strings.TrimSpace per decoded field BEFORE chain resolution (detect_csharp.go:86-95, detect_java_kotlin.go); whitespace-only Version falls through chains; table rows pin | closed |
| T-14-08 | Tampering | File literally named ".csproj" matching suffix filter | low | mitigate | exact-name guard len(e.Name()) <= len(suffix) precedes HasSuffix (detect_csharp.go:28); TestDetectCSharp_ExactNameGuard pins non-match | closed |
| T-14-09 | Spoofing | TestRunDetectors_EmptyRegistry running production registry against fixtures | low | mitigate | explicit []detectorFunc{} injection + non-colliding probe folder | closed |
| T-14-10 | DoS | CheckForUpdate/DownloadUpdate error paths on adversarial inputs | medium | mitigate | Seven error-path rows pin error-not-panic on every failure branch; go test -race in verify | closed |
| T-14-11 | Tampering | Test triggers relying on chmod/permission semantics (Windows CI) | low | mitigate | ENOTDIR triggers only (file-as-parent, slash-in-name) + closed-server URLs; zero Chmod (grep-verified) | closed |
| T-14-12 | Spoofing | Module-path derivation returning wrong owner/repo | medium | mitigate | NoOptionsDerivesOwnerRepo mock handler asserts exact derived path "/repos/guionardo/go/releases/latest"; any derivation change fails visibly | closed |
| T-14-13 | Tampering | doc.go contract drifting from implemented behavior | medium | mitigate | Contract backed by Pattern 7 code-read inventory (all 7 Version assignments verbatim) + Pattern 8 grep evidence in 14-AUDIT.md; decode-hygiene note names TrimSpace explicitly | closed |
| T-14-14 | Tampering | Stale/fabricated audit evidence masking a regression | medium | mitigate | Audit greps re-run fresh and recorded; 14-01/02/03 land before 14-04 (wave 2); verifier re-runs greps from 14-AUDIT.md; badge-stripping exception disclosed (commit 7b52713) | closed |
| T-14-15 | Spoofing | README row pointing to wrong anchor or package | low | mitigate | Row anchor locked to #package-project_probe (grep-verified); section heading matches | closed |
| T-14-SC | Tampering | npm/pip/cargo installs | high | mitigate | Package-legitimacy gate: zero installs this phase; go.mod/go.sum unchanged (git diff empty, verified) | closed |

*Status: open · closed · open — below {block_on} threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-14-04 | Fuzz-corpus poisoning: a malformed `go test fuzz v1` corpus file fails the build loudly (MalformedCorpusError) — cannot silently poison CI; corpus is repo-reviewed fixture content with a go test well-formedness gate | Guionardo Furlan | 2026-09-29 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-29 | 16 | 16 | 0 | gsd-security-auditor (orchestrator L1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed