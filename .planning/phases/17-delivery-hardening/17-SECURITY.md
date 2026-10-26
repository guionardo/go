---
phase: "17"
slug: "delivery-hardening"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-26"
verified: "2026-10-26"
---

# Phase 17 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| CI workflow → runner shells | The three OS runners execute the same `run:` blocks through different default shells (bash vs pwsh); exit-status propagation differs and can silently mask a failure | Command text, exit status |
| job env → step env | A step-level `env:` silently overrides the job-wide build property; the race step is the one intentional crossing | Build properties (`CGO_ENABLED`) |
| workflow → third-party Actions | `actions/checkout@v4`, `actions/setup-go@v5` and `vladopajic/go-test-coverage@v2` are supply-chain inputs; this phase adds none | Action code, runner credentials |
| build logs → public CI output | Everything `run:` prints lands in a public log | Build diagnostics |
| benchmark → filesystem | `sqlite/File` and `BenchmarkPrimitives/sqlite/File` create real databases; a fixed or user-cache path would write outside the workspace | Local SQLite files |
| example → filesystem | The durability example creates a real SQLite database on disk during `go test` | Local SQLite files |
| package docs → consumers | `doc.go` and README describe the durability and batch contract; wrong prose becomes a false guarantee | Documentation claims |
| coverage configs → gate outcome | Any override added to a config file silently weakens the repo-wide threshold gate | Threshold settings |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-17-01 | Repudiation / Tampering | `Build (CGO-free)` step (false assurance that a CGO-free build was proven) | high | mitigate | `shell: bash` + `set -euo pipefail`, so any non-zero exit among the step's commands fails it; `cgo=$(go env CGO_ENABLED)` is captured and compared to `0` with an explicit failure, then `go build`/`go vet`/`go list -m` run | closed |
| T-17-02 | Denial of service | `SQLite race detector` step under job-level `CGO_ENABLED=0` | medium | mitigate | Step-level `env: CGO_ENABLED: "1"` on the race step only, with its `if: runner.os != 'Windows'` gate intact; negative local probe (`CGO_ENABLED=0 GOOS=linux go test -race -c`) prints `go: -race requires cgo`, proving the override is necessary; `CGO_ENABLED=1 -race` is clean | closed |
| T-17-03 | Tampering | New or unpinned third-party Action | high | mitigate | No `uses:` reference added by this phase; the `test` job keeps `actions/checkout@v4` and `actions/setup-go@v5` with `permissions: contents: read` / `pull-requests: read` | closed |
| T-17-04 | Information disclosure | Build step's log output | low | mitigate | The step prints only the cgo flag value and two module versions; no environment dump, no token reference, `secrets.*` never named | closed |
| T-17-05 | Tampering (local) | file-mode benchmark paths in `cache/bench_test.go` | medium | mitigate | Every path is `filepath.Join(b.TempDir(), "<fixed>.db")` (`batch.db`, `primitives.db`); no absolute path, no `sqlite.WithName`, no user cache directory | closed |
| T-17-06 | Denial of service (CI flake) | SQLite handle / sidecar leak on the windows leg | medium | mitigate | `b.Cleanup(func(){ _ = c.Close() })` is registered after `b.TempDir()` in all four sqlite arms, so LIFO ordering closes each cache before its temp dir is removed | closed |
| T-17-07 | Denial of service | unbounded database growth during a timed loop | low | mitigate | `const primitiveKeyRing = 1024` bounds the primitive key space; `batchSizes = []int{1, 10, 100, 1000}` unchanged; `make benchmark-quick` (~20 s) is the validation target | closed |
| T-17-08 | Repudiation (doc drift implying protection) | `cache/sqlite/doc.go` batch paragraph, README clause | medium | mitigate | New prose describes behaviour only — no confidentiality, encryption, or retention guarantee; the existing "Cache contents are unencrypted at rest" sentence is preserved verbatim; claims are gated against `batch.go` behaviour | closed |
| T-17-09 | Tampering (local data) | `ExampleNew_withPath` filesystem use | medium | mitigate | `os.MkdirTemp("", "sqlite-cache-example")` + `defer os.RemoveAll(dir)`; path built with `filepath.Join(dir, "cache.db")`; no `sqlite.WithName`, no fixed path | closed |
| T-17-10 | Repudiation | `17-COVERAGE-AUDIT.md` numbers | medium | mitigate | Every figure quoted from a `make coverage-quick` run performed in the same task (exit 0, verbatim tail included); five-class rows cite `file.go:LINE` locations that resolve to real test functions (verifier re-checked all 33) | closed |
| T-17-SC | Tampering | npm/pip/cargo installs (reserved) | high | mitigate | Reserved row, never exercised: this phase adds no package install, no module, no action; `go.mod` is untouched | closed (reserved, unused) |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**Note on ASVS level.** `security_asvs_level: 1`, `security_block_on: high`. The register was authored at plan time in all three PLAN.md files, so L1 grep-depth verification is the required and sufficient depth (no auditor spawn). No auth, session, access-control, input-validation or cryptography surface is touched: this phase changes CI configuration, test-only benchmark code, and documentation.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-17-01 | — (pre-existing, not a T-17 threat) | The separate `coverage` CI job still pins `actions/checkout@v3` / `actions/setup-go@v3` and holds `permissions: contents: write` (badge publishing). Observed during this phase and recorded as out-of-scope follow-up `IN-04` in `17-REVIEW.md`; changing it would alter the repo-wide gate outside Phase 17's boundary. | phase orchestrator, user-visible via 17-REVIEW-DISPOSITION.md | 2026-10-26 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-26 | 10 (+1 reserved) | 10 | 0 | gsd-secure-phase (L1 short-circuit, no auditor spawn) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-26
