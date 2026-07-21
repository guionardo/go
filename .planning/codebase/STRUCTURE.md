# Codebase Structure

**Analysis Date:** 2026-07-21

## Directory Layout

```
go/                                          # Module root: github.com/guionardo/go
├── .github/                                 # GitHub configuration
│   ├── copilot-instructions.md              # AI coding assistant instructions
│   └── workflows/
│       └── go.yml                           # CI pipeline (test + coverage + report card)
├── .planning/                               # GSD planning artifacts
│   └── codebase/                            # Codebase mapping documents
├── br_docs/                                 # Brazilian document validation
│   ├── brdocs.go                            # CPF/CNPJ validation logic
│   └── brdocs_test.go                       # Tests
├── config/                                  # Generic typed configuration provider
│   ├── config_test.go                       # Tests for Provider[T]
│   ├── consts.go                            # Constants (env vars, defaults)
│   ├── logging.go                           # Logging helpers (slog, safe-field redaction)
│   ├── options.go                           # Functional options (WithScope, WithProfilesPath, etc.)
│   ├── provider.go                          # Provider[T] — main public API
│   ├── provider_base.go                     # provider internal struct + validation logic
│   ├── environment/                         # Env var parsing into structs
│   │   ├── environment.go                   # ParseEnvironment, GetEnv, setField
│   │   └── environment_test.go
│   ├── merger/                              # Recursive deep-merge of maps
│   │   ├── maps.go                          # MergeMaps, updateMapValues
│   │   └── maps_test.go
│   ├── profile/                             # YAML profile loading with scope layering
│   │   ├── profile.go                       # GetScopedProfileContent, getProfileFiles
│   │   └── profile_test.go
│   └── validation/                          # Struct validation
│       ├── validator.go                     # Validator interface + validate.Struct wrapper
│       └── validator_test.go
├── flow/                                    # Generic control flow utilities
│   ├── default.go                           # Default[T] — zero-value fallback
│   ├── default_test.go
│   ├── example_test.go
│   ├── if.go                                # If[T] — generic ternary
│   └── if_test.go
├── fraction/                                # Immutable fraction type with arithmetic
│   ├── example_test.go
│   ├── fraction.go                          # Fraction type, New, FromFloat64, Add/Subtract/etc.
│   └── fraction_test.go
├── httptest_mock/                           # HTTP mock server for tests
│   ├── builder.go                           # NewMock + fluent builder (WithQueryParam, etc.)
│   ├── builder_test.go
│   ├── custom_handler_test.go
│   ├── handler.go                           # MockHandler — http.Handler implementation
│   ├── handler_test.go
│   ├── helpers.go                           # GetMockHandlerFromServer, GetMocksFrom
│   ├── helpers_test.go
│   ├── interfaces.go                        # Mocker interface, CustomHandlerFunc type
│   ├── mock.go                              # Mock struct — matching, response, assertions
│   ├── mock_test.go
│   ├── README.md                            # Standalone documentation
│   ├── request.go                           # Request — method, path, query, header, body matching
│   ├── request_test.go
│   ├── response.go                          # Response — status, body, headers, delay
│   ├── response_test.go
│   ├── setup.go                             # SetupServer + option functions
│   ├── setup_test.go
│   ├── string_parts.go                      # StringParts — structured key-value logging
│   ├── string_parts_test.go
│   ├── test_utils.go                        # CreateTestRequest, getBodyReader
│   └── (other _test.go files)
├── mid/                                     # Cross-platform machine identifier
│   ├── machineid_darwin.go                  # macOS: system_profiler
│   ├── machineid_linux.go                   # Linux: hostnamectl / dbus / etc
│   ├── machineid_linux_test.go
│   ├── machineid_test.go
│   ├── machineid_windows.go                 # Windows: SQMClient registry
│   └── (shared test file)
├── path_tools/                              # File/directory path utilities
│   ├── find_file_path.go                    # FindFileInPath — PATH search
│   ├── find_file_path_test.go
│   ├── path_tool.go                         # DirExists, FileExists, CreatePath
│   ├── path_tool_darwin.go                  # macOS createPath
│   ├── path_tool_linux.go                   # Linux createPath
│   ├── path_tool_test.go
│   ├── path_tool_windows.go                 # Windows createPath
│   ├── root_directory.go                    # IsRootDirectory (cross-platform)
│   ├── root_directory_test.go
│   ├── root_folder.go                       # GetRootFolder — find go.mod upward
│   └── root_folder_test.go
├── reflect_tools/                           # Reflection utilities
│   ├── reflect_tools.go                     # IsZeroValue
│   └── reflect_tools_test.go
├── release/                                 # GitHub release fetcher
│   ├── release.go                           # Release/Asset types, GetLatestRelease, Download
│   └── release_test.go
├── set/                                     # Generic Set[T] implementation
│   ├── example_test.go
│   ├── marshal.go                           # JSON marshal/unmarshal
│   ├── marshal_test.go
│   ├── scanner_valuer.go                    # database/sql Scanner + Valuer
│   ├── scanner_valuer_test.go
│   ├── set.go                               # Set[T] — core operations
│   └── set_test.go
├── shell_tools/                             # Shell environment utilities
│   ├── environment.go                       # GetEnv — case-insensitive env var lookup
│   ├── environment_test.go
│   ├── example_test.go
│   ├── shell_args.go                        # QuotedShellArgs — parse and reconstruct
│   └── shell_args_test.go
├── time_tools/                              # Flexible time parsing
│   ├── example_test.go
│   ├── parser.go                            # Parse (auto-prioritizing), SetLayouts
│   └── parser_test.go
├── .commitlint.yaml                         # Commit message conventions
├── .golangci.yml                            # Linter configuration (42 linters + 3 formatters)
├── .gitignore
├── .pre-commit-config.yaml                  # Pre-commit hook configuration
├── .pre-commit-hooks.yaml
├── .testcoverage.yml                        # Coverage thresholds (file:70%, pkg:80%, total:95%)
├── CHANGELOG.md                             # Keep a Changelog
├── CONTRIBUTING.md                          # Contribution guide
├── LICENSE                                  # License file
├── Makefile                                 # Build/test/lint/deps targets
├── README.md                                # Project overview + per-package docs
├── go.mod                                   # Module definition + dependencies
└── go.sum                                   # Dependency checksums
```

## Directory Purposes

**`br_docs/`:**
- Purpose: Brazilian document number validation
- Contains: CPF and CNPJ check-digit algorithms, sanitation helpers
- Key files: `brdocs.go`

**`config/`:**
- Purpose: Generic typed configuration provider — YAML profiles + env vars + validation
- Contains: `Provider[T]` public API, functional options, sub-packages for env/profile/merge/validation
- Key files: `provider.go`, `options.go`, `provider_base.go`, `consts.go`, `logging.go`
- Sub-packages: `environment/`, `profile/`, `merger/`, `validation/`

**`flow/`:**
- Purpose: Small generic control-flow helpers
- Contains: 2 exported functions — `If[T]`, `Default[T]`
- Key files: `if.go`, `default.go`

**`fraction/`:**
- Purpose: Immutable fraction arithmetic type (forked from go-fraction)
- Contains: `Fraction` type with add/subtract/multiply/divide/equal/float64, `FromFloat64`
- Key files: `fraction.go`

**`httptest_mock/`:**
- Purpose: HTTP mock server for integration tests — define mocks in code or JSON/YAML files
- Contains: `Mocker` interface, `Mock` struct, `MockHandler`, `SetupServer`, fluent builder, test utilities
- Key files: `mock.go`, `handler.go`, `setup.go`, `request.go`, `response.go`, `builder.go`, `interfaces.go`

**`mid/`:**
- Purpose: Cross-platform machine identifier
- Contains: OS-specific `MachineID()` implementations via build tags
- Key files: `machineid_darwin.go`, `machineid_linux.go`, `machineid_windows.go`

**`path_tools/`:**
- Purpose: File system path and directory operations
- Contains: Existence checks, path creation, Go root detection, PATH search, root directory detection
- Key files: `path_tool.go`, `root_folder.go`, `root_directory.go`, `find_file_path.go`

**`reflect_tools/`:**
- Purpose: Reflection utilities for common introspection patterns
- Contains: `IsZeroValue` — handles all Go types including time.Time, slices, maps
- Key files: `reflect_tools.go`

**`release/`:**
- Purpose: GitHub release API integration
- Contains: `Release`/`Asset` structs, `GetLatestRelease`, `GetThisLatestRelease`, `Asset.Download` with digest verification
- Key files: `release.go`

**`set/`:**
- Purpose: Generic set data structure
- Contains: `Set[T]` type, Union/Diff/Intersection/Filter, JSON marshal, SQL Scanner/Valuer
- Key files: `set.go`, `marshal.go`, `scanner_valuer.go`

**`shell_tools/`:**
- Purpose: Shell argument and environment utilities
- Contains: `QuotedShellArgs` parser/reconstructor, case-insensitive `GetEnv`
- Key files: `shell_args.go`, `environment.go`

**`time_tools/`:**
- Purpose: Flexible time string parser
- Contains: `Parse` (auto-prioritizing layout matching), `SetLayouts`
- Key files: `parser.go`

## Key File Locations

**Entry Points (Public API surfaces — pure library, no executables):**
- `config/provider.go`: `NewProvider[T]`, `Provider.GetConfiguration`, `Provider.UpdateConfiguration`
- `httptest_mock/mock.go`: `Mock` struct
- `httptest_mock/setup.go`: `SetupServer`, `WithRequests`, `WithRequestsFrom`
- `httptest_mock/builder.go`: `NewMock`, fluent builder chain
- `set/set.go`: `New[T]`, `Set[T]` methods
- `fraction/fraction.go`: `New`, `FromFloat64`, `Fraction` methods

**Configuration:**
- `.golangci.yml`: Linter configuration (42 linters, 3 formatters)
- `.testcoverage.yml`: Coverage thresholds
- `.commitlint.yaml`: Conventional commit rules
- `.pre-commit-config.yaml`: Git hook configuration
- `Makefile`: Build, test, lint, and dependency targets
- `go.mod`: Module definition and dependencies

**Core Logic (most complex packages):**
- `config/`: 6 files + 4 sub-packages — configuration loading pipeline
- `httptest_mock/`: 11 source files — HTTP mock infrastructure
- `set/set.go`: 160 lines — generic set operations
- `fraction/fraction.go`: 231 lines — fraction arithmetic + float conversion

**Testing:**
- Tests co-located with source in every package directory
- `_test.go` per source file
- `example_test.go` in `flow/`, `fraction/`, `set/`, `shell_tools/`, `time_tools/`
- `httptest_mock/test_utils.go`: Shared test helpers

## Naming Conventions

**Files:**
- `snake_case.go` — Go convention for multi-word file names: `path_tool.go`, `find_file_path.go`, `shell_args.go`, `reflect_tools.go`, `scanner_valuer.go`
- `_os.go` suffix for platform-specific: `_darwin.go`, `_linux.go`, `_windows.go`
- `_test.go` suffix for test files
- `_internal_test.go` for internal tests in `config/merger/maps_internal_test.go`
- Singular package directory names: `config/`, `flow/`, `set/`, `mid/` (except `br_docs`, `path_tools`, `shell_tools`, `reflect_tools`, `time_tools` — these need disambiguation)

**Directories:**
- `snake_case` for package directories: `br_docs/`, `path_tools/`, `reflect_tools/`, `shell_tools/`, `time_tools/`, `httptest_mock/`
- Short singular names: `flow/`, `set/`, `mid/`, `release/`
- Sub-packages use single-word names: `environment/`, `profile/`, `merger/`, `validation/`

**Go Packages:**
- Package names are short, lowercase, single-word: `config`, `flow`, `fraction`, `mid`, `set`, `release`
- Multi-word package names use concatenation (no underscore in import path): `pathtools` (go.mod visible as `github.com/guionardo/go/path_tools`), `shelltools`, `reflecttools`, `timetools`
- Exception: `httptestmock` — import path is `github.com/guionardo/go/httptest_mock`, package name `httptestmock`

## Where to Add New Code

**New Utility Package:**
- Implementation: `go/<package-name>/` — create a new top-level directory
- Tests: `go/<package-name>/<name>_test.go` — co-located with source
- Example tests: `go/<package-name>/example_test.go` for documentation
- Entry point for README: Add to the TOC and body of `README.md`

**New Feature in Existing Package:**
- Primary code: Append to relevant existing `.go` file or create a new file for distinct concern
- Tests: Add to existing `_test.go` or create separate test file
- Example: `set/marshal.go` for JSON marshaling concerns separate from `set/set.go`

**New Sub-package Under config/:**
- Implementation: `config/<subpackage>/` — e.g., `config/environment/`
- Tests: Co-located in the sub-package directory
- Integration: Wire into `config/provider.go` or `config/provider_base.go`

**Configuration / Tooling:**
- CI: `.github/workflows/` — add new workflow `.yml` files
- Linting: `.golangci.yml` — add/remove linters in `linters.enable` list
- Hooks: `.pre-commit-config.yaml` — add pre-commit hooks
- Make targets: `Makefile` — add under appropriate section heading

## Special Directories

**`.github/`** — Not generated, Committed
- Purpose: GitHub-specific configuration — CI workflow + Copilot instructions
- Contains: `workflows/go.yml` (CI pipeline), `copilot-instructions.md`

**`.planning/`** — Partially generated, Committed
- Purpose: GSD (Goal-oriented Software Development) planning artifacts
- Contains: Codebase mapping documents, phase plans, state tracking

**`config/environment/`, `config/merger/`, `config/profile/`, `config/validation/`** — Not generated, Committed
- Purpose: Internal support packages for the `config` module
- Note: These are public Go packages (no `internal/` visibility restriction), but designed as implementation details of `config`

---

*Structure analysis: 2026-07-21*
