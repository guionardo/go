# External Integrations

**Analysis Date:** 2026-07-21

## APIs & External Services

**GitHub API:**
- **Package:** `release` (`release/release.go`)
- **Purpose:** Fetch latest release metadata and download assets from GitHub repositories
- **Endpoint:** `https://api.github.com/repos/{owner}/{repo}/releases/latest`
- **Auth:** None (unauthenticated requests)
- **Headers:** `X-Github-Api-Version: 2026-03-10`, `Accept: application/vnt.github+json`
- **Usage:** `release.GetLatestRelease(owner, repo)` for generic repos; `release.GetThisLatestRelease()` for current module
- **Integrity:** Uses `github.com/opencontainers/go-digest` to verify downloaded asset digests

**GoReportCard:**
- **Package:** CI only (`.github/workflows/go.yml`)
- **Purpose:** Trigger and fetch Go code quality report card
- **Endpoint:** `https://goreportcard.com/checks` and `https://goreportcard.com/report/github.com/guionardo/go`
- **Auth:** None

## Data Storage

**Databases:**
- None — no database connection is established at runtime
- The `set` package (`set/scanner_valuer.go`) implements `database/sql.Scanner` and `database/sql/driver.Valuer` interfaces on `Set[T]`, enabling use as a SQL column type, but no actual database driver is imported

**File Storage:**
- Local filesystem only
- Profile-based YAML configuration reads from local filesystem (`config/profile/profile.go`)
- Mock HTTP test definitions loaded from local JSON/YAML files (`httptest_mock/setup.go`)

**Caching:**
- None — no caching layer implemented

## Authentication & Identity

**Auth Provider:**
- None — no authentication system is integrated
- OS-level machine identification (non-authentication, non-identity) provided by `mid/` package:
  - macOS: `system_profiler SPHardwareDataType` (exec) → model number, serial number, hardware UUID
  - Linux: `hostnamectl status` (exec), `/var/lib/dbus/machine-id`, `/etc/machine-id` (file read)
  - Windows: `reg query HKLM\SOFTWARE\Microsoft\SQMClient` (exec) → MachineId GUID

## Monitoring & Observability

**Error Tracking:**
- None — no external error tracking service (Sentry, Datadog, etc.)

**Logs:**
- `log/slog` (stdlib) — used in `config/` and `httptest_mock/` packages
  - `config/logging.go` — structured logging for configuration events
  - `config/provider_base.go` — slog for config operations
  - `httptest_mock/handler.go` — slog and `testing.T.Logf` for mock server diagnostics
- `slog.DiscardHandler` used in test mock servers by default

## CI/CD & Deployment

**Hosting:**
- Not applicable — this is a Go library, not a deployed application
- Published as a Go module at `github.com/guionardo/go`
- Go module documentation at `https://pkg.go.dev/github.com/guionardo/go`

**CI Pipeline:**
- **Provider:** GitHub Actions (`.github/workflows/go.yml`)
- **Workflows:**
  1. `Go tests and checking` — triggered on push/PR to `main` and `develop`:
     - Multi-OS matrix: ubuntu-latest, macos-latest, windows-latest
     - `actions/checkout@v4` + `actions/setup-go@v5`
     - `go test -v ./...` across all OS targets
  2. Coverage check (ubuntu-latest):
     - `go test -coverprofile=./cover.out -covermode=atomic -coverpkg=./...`
     - `vladopajic/go-test-coverage@v2` — enforces 80% total threshold
     - Badge commit to `badges` branch on main pushes
  3. GoReportCard update (ubuntu-latest):
     - Triggers goreportcard.com check and publishes summary to step summary
- **Code scanning:** CodeQL (badge in README via `github-code-scanning/codeql`)
- **Secrets:** `GITHUB_TOKEN` used for coverage badge updates

## Environment Configuration

**Required env vars:**
- None required at minimum
- Config package env vars (all optional): `SCOPE`, `DEFAULT_SCOPE`, `PROFILES_PATH`, `CONFIGURATION_LOG`
- Struct-level env tags: defined by consumer via `env:"VAR_NAME"` tags on config struct fields
- Application config uses `env` and `default` struct tags for environment variable binding (`config/environment/environment.go`)

**Secrets location:**
- No secrets stored — all integrations are unauthenticated
- `.env` file is gitignored (noted in `.gitignore`), but no env file is present

## Webhooks & Callbacks

**Incoming:**
- None — no webhook endpoints defined

**Outgoing:**
- None — no webhook delivery mechanism implemented

---

*Integration audit: 2026-07-21*
