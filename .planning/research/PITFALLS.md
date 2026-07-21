# Domain Pitfalls: Go Utility Library Authors

**Domain:** Go utility packages / developer tools
**Researched:** 2026-07-21
**Sources:** go.dev/blog, dave.cheney.net, Go skills (code-style, design-patterns, safety, security), Go team talks (Jonathan Amsterdam, Russ Cox)
**Overall Confidence:** HIGH — authoritative sources (Go team blog, Dave Cheney's established design advice, plus project-specific concerns already validated in CONCERNS.md)

---

## Critical Pitfalls

Mistakes that cause rewrites, major version bumps, or break downstream consumers.

### Pitfall 1: Changing Function Signatures Instead of Adding

**What goes wrong:** Adding a required parameter, converting a parameter from positional to variadic, or changing parameter types on an exported function. This breaks all callers at compile time.

**Why it happens:** The author needs to pass additional information to a function. It seems "small" so the author changes the existing function rather than adding a new one.

**Consequences:** Every downstream caller stops compiling. In a utility library consumed by many projects, this forces mass updates or pins users to old versions.

**Go-specific nuance:** Even adding variadic parameters breaks function type compatibility. `func Run(name string)` has type `func(string)`, but `func Run(name string, opts ...Option)` has type `func(string, ...Option)`. Assignments like `var fn func(string) = Run` break.

**Prevention:**
- Never change an exported function's signature — **add, don't change or remove** (Go team's first rule of compatibility)
- Use the *add new function* pattern: `Query()` → `QueryContext()` (stdlib pattern)
- Plan ahead with option structs (`Config` struct with nil-acceptance), functional options (`type Option func(*T)`), or variadic option parameters
- When adding a feature that needs new params, add a new function with a descriptive name rather than tacking on more arguments

**Detection:**
- `go vet` catches some type incompatibilities
- `gorelease` (golang.org/x/exp/cmd/gorelease) detects API changes
- `go build ./...` on downstream test projects
- Adding `//go:build api-compat` tests that compile the old API surface

**Phase mapping:** Every phase that adds or modifies exported functions. Enforce during code review — flag ANY change to exported function signatures.

---

### Pitfall 2: Forcing Allocation on Callers

**What goes wrong:** An API allocates memory internally and returns it, preventing callers from reusing buffers. Over time this creates GC pressure that can't be eliminated without breaking the API.

**Why it happens:** The author optimizes for convenience (returning `[]byte` or `string` from internal allocation) rather than allocation-control. Dave Cheney's example: `func Read() ([]byte, error)` vs `func Read(buf []byte) (int, error)`.

**Consequences:** Once the API is committed (v1+), the allocation pattern can't be changed without a breaking change. Callers who need performance can't pool or reuse allocations.

**Prevention:**
- Accept buffers from callers when the function reads or produces byte data
- Use `io.Reader`/`io.Writer` patterns instead of returning allocated slices
- If returning allocated values is necessary, document the allocation behavior
- For utility functions that transform data, accept the destination as a parameter

**Detection:**
- Benchmark tests with `-benchmem` show allocation counts
- Review: functions that return `[]byte` or `string` that they created internally
- Review: methods that allocate without accepting a caller-provided buffer

**Phase mapping:** New package creation phase. Retrofit requires breaking change (v2). Design for allocation control from day one.

---

### Pitfall 3: Exporting Interfaces Users Must Implement

**What goes wrong:** An exported interface with public methods that users outside the package implement. Later, adding a method to that interface breaks all external implementations.

**Why it happens:** The author uses interfaces for testability or abstraction, but doesn't anticipate that downstream users will implement them.

**Consequences:** Adding a method to the interface (even a useful one) becomes a breaking change. The author is locked into the interface shape forever.

**Prevention:**
- **Accept interfaces, return structs** (Jack Lindamood / Dave Cheney rule)
- If you must export an interface, add an **unexported method** to prevent external implementation (stdlib pattern: `testing.TB` has a `private()` method)
- Consider whether callers actually need to implement the interface vs. just consume it
- Prefer returning concrete types from constructors so you can add methods later

**Applied to this project (go):** The `config.Provider` type returns a concrete struct, not an interface — good. The `set.Set[T]` uses concrete methods — good. Avoid creating interfaces consumers would implement.

**Detection:**
- Dynamic: `gorelease` detects new methods added to exported interfaces
- Static: grep for `type.*interface` in exported packages — review each for the "can external implement this?" question

**Phase mapping:** API design / new package phase. Retroactively adding a private method to an interface breaks no one, but requires a major version if the interface is already published.

---

### Pitfall 4: Package-Level Global State (The Logger Anti-Pattern)

**What goes wrong:** Declaring a package-level variable (logger, config, DB connection) that creates a compile-time dependency on a specific library.

**Why it happens:** "Every package needs to log" — the author adds `var log = mylogger.GetLogger(...)` which couples every importer to `mylogger`.

**Consequences:** All downstream consumers inherit the transitive dependency. Projects composed of multiple utility packages end up coupled to multiple logging/monitoring frameworks. Testing becomes harder because global state is hard to replace.

**Prevention (from Dave Cheney's advice):**
- Inject dependencies via struct fields, not package variables
- Define narrow interfaces in the consuming package (e.g., `type logger interface { Printf(string, ...interface{}) }`)
- Defer binding to runtime via constructor parameters
- For configuration, pass it explicitly rather than relying on a global singleton

**Applied to this project (go):** The `config` package already uses a `Provider` struct with injection — good. But `config/environment` uses recursive panic-recovery — see CONCERNS.md. Avoid adding more global singletons.

**Detection:**
- Search for `var (` at package level with dependencies on external packages
- Search for `init()` functions — almost always wrong in library code
- `go mod graph` reveals unexpected transitive dependencies

**Phase mapping:** Any phase adding cross-cutting concerns (logging, metrics, tracing). Address early — retrofitting dependency injection is expensive.

---

### Pitfall 5: Semantic Versioning Violations (Breaking Changes Under Same Module Path)

**What goes wrong:** Publishing a breaking change (removing an exported function, changing a type, modifying a signature) under the same module path without bumping the major version.

**Why it happens:** The author doesn't realize Go's import compatibility rule applies, or thinks "it's a small change."

**Go's rule (from research.swtch.com/vgo-import):** If an old package and a new package have the same import path, the new package must be backwards compatible with the old package.

**Consequences:** When downstream users run `go get -u`, their code breaks silently at compile time. Some may pin to old versions, creating a fractured ecosystem. For pre-v1 (v0.x.x), breaking changes are expected — but for v1+, this breaks Go's compatibility promise.

**Prevention:**
- Follow semver strictly: breaking change = new major version = new module path with `/v2` suffix
- Use `gorelease` to detect compatibility before tagging
- Keep pre-v1 modules in v0 for as long as the API is experimental
- For breaking internal changes: use `internal/` packages (see Pitfall 6)
- Document your compatibility promises in your module's README and go.mod

**What counts as breaking in Go:**
- Removing or renaming an exported function, type, or constant
- Changing a function's signature (add/remove/change params)
- Adding a method to an exported interface (breaks implementations)
- Changing a method's receiver from `T` to `*T` (or vice versa)
- Changing a type from struct to interface (or vice versa)
- Removing a field from an exported struct
- Adding a non-comparable field to a previously comparable struct

**Detection:**
- Run `gorelease` before tagging — it compares against the last tag
- `go build ./...` on a known downstream project
- `go vet ./...` — catches some interface violations

**Phase mapping:** Every release phase. Automate with CI (run `gorelease` as part of pre-tag checks).

---

### Pitfall 6: Failing to Use `internal/` Package Boundaries

**What goes wrong:** Exporting symbols that are only meant for intra-module use, making them part of the public API commitment.

**Why it happens:** The author doesn't know about `internal/` packages, or organizes code into many small packages without visibility boundaries.

**Consequences:** Every exported name becomes a backward compatibility commitment. The author can't refactor internal helpers without potentially breaking external consumers.

**Prevention (from Dave Cheney, Go team):**
- Use `internal/` directories to hide implementation details from external consumers
- Packages under `internal/` can only be imported by code sharing a common ancestor
- Start with more things internal — you can always promote to public later
- A `pkg/` directory at the project root is often a smell—it's usually an `internal/` opportunity

**Applied to this project (go):** The project's packages are all at top level (config, set, fraction, etc.). If cross-package helper functions exist, they should be in `internal/` not exported.

**Detection:**
- Search for exported functions/types that are only used within the module
- `gorelease` flags public API that changed — internal packages won't appear

**Phase mapping:** Project layout / initial structure phase. Adding `internal/` boundaries later requires moving code (non-breaking if you keep shims, but messy).

---

### Pitfall 7: Value Receiver on Structs with Mutex or Slice Fields

**What goes wrong:** Declaring methods with value receivers on structs that contain `sync.Mutex`, slices, maps, or other reference types.

**Why it happens:** The method doesn't mutate the struct, so the author uses value receivers for "immutability."

**Consequences (from Dave Cheney's T vs *T analysis):**
- Copying a struct with `sync.Mutex` breaks the mutex (it copies lock semantics)
- Copying a struct with a slice field shares the backing array — mutations by one copy affect others
- Copying a struct with a map shares the underlying map reference
- Embedding a value-receiver type in another struct copies the mutex silently

**Prevention:**
- **Prefer `*T` receivers for all methods unless you have a strong reason** (Dave Cheney's rule)
- Only use value receivers for small, immutable types (like `time.Time`, small numerical types)
- For types with any reference field (slice, map, channel, mutex, pointer), use `*T`
- For types that embed others with mutex fields, also use `*T`

**Detection:**
- `go vet` catches `Assignment: copy lock value to ...` for mutex fields
- Manual review: check receiver type on methods of structs with reference fields

**Phase mapping:** New type creation phase. Fixing later requires changing all callers from T to *T — a breaking change.

---

## Moderate Pitfalls

### Pitfall 8: Package Naming and Organization (base, util, common)

**What goes wrong:** Creating catch-all packages named `utils`, `helpers`, `common`, `base`, or `misc`.

**Why it happens:** Import loops force extracting unrelated functions into a shared package. The package name reflects what it *contains* rather than what it *provides*.

**Consequences:** These packages accumulate unrelated functions, have no cohesive purpose, change frequently and for many reasons, and tell consumers nothing about what they do.

**Prevention (from Dave Cheney, Go team):**
- Name packages after what they *provide*, not what they *contain*
- A package's name should be a description of its purpose and a namespace prefix
- Good examples: `net/http`, `encoding/json`, `os/exec`
- Instead of `utils`, split into multiple packages with descriptive names (e.g., `strutil`, `fileutil` only if they have a focused purpose)
- To break import loops, prefer duplicating a small amount of code over creating a `common` package

**Applied to this project (go):** The project has `path_tools`, `shell_tools`, `time_tools`, `reflect_tools` — acceptable because each is focused on a specific domain. If any were named just `tools` or `utils`, that would be a problem.

**Detection:**
- Search for directory/package names: `util`, `utils`, `helper`, `helpers`, `common`, `base`, `misc`

**Phase mapping:** Project initialization / new package phase. Renaming after publication breaks import paths.

---

### Pitfall 9: Orphaned Exported Symbols (Over-Exporting)

**What goes wrong:** Exporting types, functions, and constants that are no longer needed or should be private.

**Why it happens:** The author exports everything "just in case" or doesn't clean up when refactoring. Also, v0 APIs that are expanded prematurely.

**Consequences:** Every exported symbol is a permanent commitment. Over-exporting bloats the API surface, increases documentation burden, and makes future breaking changes more painful.

**Prevention:**
- **Unexport aggressively** — you can always export later; unexporting is a breaking change (Go code style skill)
- After refactoring, review what's actually used outside the package
- Use `internal/` to prevent external access to intra-module symbols
- Mark unstable APIs with documentation comments (`// Deprecated:` or experimental package doc)
- Use the `// Deprecated:` convention when you want to signal removal intent

**Detection:**
- Tools like `staticcheck` detect unused exported symbols
- `gorelease` shows all exported symbols and flags removals
- `go list -u -m` can help identify what symbols are imported downstream

**Phase mapping:** Every phase. Enforce in code review: "Is this export necessary?"

---

### Pitfall 10: Error Handling — Wrapping Implementation Details

**What goes wrong:** Wrapping errors from dependencies (especially database, network, or third-party libraries) with `%w`, making those errors part of the library's API contract.

**Why it happens:** The author uses `fmt.Errorf("context: %w", err)` without considering that `err` is from an underlying dependency.

**Consequences (from Go 1.13 errors post, Damien Neil / Jonathan Amsterdam):**
- Downstream callers can use `errors.Is(err, sql.ErrNoRows)` on your error — if you switch databases or the dependency's error changes, your callers break
- The wrapped error becomes part of your API commitment
- Violates abstraction — callers shouldn't need to know about your dependencies' errors

**Prevention:**
- Use `%v` (not `%w`) when the error is from an implementation detail
- Only use `%w` for errors that are part of your documented contract
- Define your own sentinel errors or error types instead of exposing dependency errors
- For utility libraries wrapping external APIs (like `release/release.go`), return your own error types

**Applied to this project (go):** The `config` package wraps `yaml.Unmarshal` errors — currently using `fmt.Errorf` which is good. The `release` package calls GitHub API — should use its own error types, not propagate HTTP errors.

**Detection:**
- Search for `%w` in error formatting — review each one: "Is this error part of my API?"
- Search for `errors.Is` or `errors.As` in tests — these lock in specific error values

**Phase mapping:** Error handling pattern phase. Can be retrofitted if errors were never exposed (wrapping with `%v` instead of `%w`).

---

### Pitfall 11: Ignoring Zero-Value Design

**What goes wrong:** Exporting types whose zero value is not useful (nil maps that panic on write, uninitialized fields that should have defaults).

**Why it happens:** The author assumes every consumer will use the constructor function, forgetting that struct literals bypass it.

**Consequences:** Users who declare `var t MyType` get a broken value. Maps panic, channels block, nil pointers crash.

**Prevention:**
- **Design useful zero values** — `var buf bytes.Buffer` is the gold standard
- Use lazy initialization for nil-unsafe fields (check-nil-and-init in methods)
- Use `sync.Once` for lazily-initialized fields
- For types that can't have a useful zero value, make the constructor mandatory and document it
- Add `// zero value is not safe to use` to the type doc if necessary

**Examples of good zero-value design:**
- `sync.Mutex` — unlocked and ready
- `bytes.Buffer` — empty buffer ready for writing
- `net/http.Client` — default timeout and transport

**Detection:**
- Look for exported structs with map, slice, or channel fields that aren't initialized in methods
- Test: `var x MyType; x.DoSomething()` should not panic
- Test with `go test -fuzz`

**Phase mapping:** New type creation phase. Fixing zero-value unsafety after release requires migration.

---

### Pitfall 12: No Example Tests or Documentation Tests

**What goes wrong:** Publishing packages without `Example` tests or runnable documentation that demonstrates API usage.

**Why it happens:** The author considers tests a separate concern from documentation, or finds example tests verbose.

**Consequences:**
- Consumers can't see how the API is intended to be used
- `go doc` output is minimal
- Breaking changes to behavior may go undetected (example tests verify they compile and produce expected output)
- Lower discoverability on pkg.go.dev (example tests are surfaced prominently)

**Prevention:**
- Write Example tests for every exported type and significant function
- Example tests are compiled and run as part of `go test` — they verify API correctness
- They double as documentation — `go doc` and pkg.go.dev display them

**Applied to this project (go):** Check if packages like `set`, `fraction`, `flow` have Example tests. If not, add them.

**Detection:**
- `go doc -all | grep "Example"` or grep for `func Example` in `_test.go` files
- Check pkg.go.dev page for the module

**Phase mapping:** Test/documentation phase for each package. Should be part of the acceptance criteria.

---

### Pitfall 13: Not Handling Context in Blocking Operations

**What goes wrong:** Utility functions that perform I/O, network calls, or blocking operations without accepting a `context.Context`.

**Why it happens:** The function "doesn't need cancellation" in its initial use case.

**Consequences:** Adding context support later requires either adding a new function (e.g., `Do` → `DoContext`) or a breaking change. Callers who need to set deadlines or cancel operations can't use the library.

**Go team guidance (from module-compatibility post):** The stdlib added `QueryContext`, `ReadContext`, etc. because changing the original signature was impossible.

**Prevention:**
- Accept `context.Context` as the first parameter for any function that does I/O, calls external APIs, or could block
- For functions that don't block, no context needed
- Plan for the `Do` / `DoContext` pattern if you must offer a convenience version without context

**Applied to this project (go):** The `release` package's `GetLatestRelease()` and `Asset.Download()` don't accept context — already identified in CONCERNS.md (no timeout). Add `GetLatestReleaseContext(ctx)`.

**Detection:**
- Search for functions that call `http.Get`, `http.Post`, `os.Open`, `exec.Command`, or any blocking operation — check if they accept context
- Linter: `contextcheck` / `noctx`

**Phase mapping:** New function creation phase. Retrofitting requires the `Do + DoContext` pattern.

---

## Minor Pitfalls

### Pitfall 14: Missing Example Tests for API Evolution

**What goes wrong:** Not having a test that explicitly verifies the API hasn't changed incompatibly.

**Why it happens:** The author relies on manual review or "just being careful."

**Consequences:** Incompatible changes slip through code review. By the time they're found, they've been published.

**Prevention:**
- Use `gorelease` in CI to compare API against the last published tag
- Add a test file that explicitly asserts expected API shape (golden file of exported symbols)
- For utility libraries, `gorelease` CI step should block PRs with incompatible changes

**Detection:**
- CI pipeline doesn't have `gorelease` step → add it

**Phase mapping:** CI setup phase. Add before first v1 release.

---

### Pitfall 15: Confusing Receiver Choice Inconsistency

**What goes wrong:** Mixing value and pointer receivers inconsistently within the same type.

**Why it happens:** Some methods don't mutate the receiver (value receiver used), others do (pointer receiver used).

**Consequences:** The type doesn't satisfy interfaces consistently. Callers can't predict whether a method modifies the receiver. A type that has both value and pointer receivers is partially usable via values but not consistently.

**Prevention:**
- Be consistent: all methods on a type should use the same receiver type
- When in doubt, use `*T` for all receivers (Dave Cheney's recommendation)
- Only use value receivers for small (~<=4 fields), immutable types with no reference fields

**Detection:**
- `go vet` — catches some cases
- Manual review: check consistency of `func (t T)` vs `func (t *T)` on the same type

**Phase mapping:** Type creation phase. Fixing is a breaking change.

---

### Pitfall 16: init() Functions in Library Code

**What goes wrong:** Using `init()` functions to set up global state, register drivers, or validate configuration.

**Why it happens:** Convenience — global registration runs automatically when the package is imported.

**Consequences (from Go design patterns skill and Dave Cheney):**
- `init()` cannot return errors — failures must panic or `log.Fatal`
- Multiple `init()` functions run in declaration order across files in filename alphabetical order — fragile
- Runs before `main()` and tests — side effects make tests unpredictable
- Makes testing harder (global state persists across test cases)

**Prevention:**
- Use explicit constructors instead of `init()`
- For registration patterns (e.g., SQL drivers), accept that `init()` is the convention but keep its scope minimal
- Never use `init()` for logic that could fail — there's no way to signal failure to the caller

**Detection:**
- `grep -r 'func init()' .` — every result needs justification

**Phase mapping:** Every phase. Ban `init()` in code review (except for `database/sql` driver registration or similar established patterns).

---

### Pitfall 17: Relying on File System State in Library Code

**What goes wrong:** Utility functions that depend on specific file paths, environment variables, or system state without making them configurable.

**Why it happens:** The function "just needs to read this one file."

**Consequences:** Testing becomes environment-dependent. Other users' systems may have different file layouts. The function is not portable.

**Prevention:**
- Accept `io.Reader` or `io.Writer` instead of file paths when possible (interface segregation)
- Make file paths or environment variables configurable parameters
- Document the assumptions about file system state
- For platform-specific paths (like `mid/machineid_linux.go`), document the lookup strategy and fallbacks

**Applied to this project (go):** The `mid` package reads `/var/lib/dbus/machine-id` and `/etc/machine-id` — already flagged in CONCERNS.md for fragile parsing (trailing whitespace not trimmed). The `config` package reads profile files from configurable paths — good.

**Detection:**
- Search for `/etc/`, `/var/`, `/usr/` or hardcoded absolute paths
- Search for `os.Getenv` — is it configurable?

**Phase mapping:** Platform-specific package creation phase. Document system assumptions alongside the code.

---

## Already Present in CONCERNS.md

These pitfalls are already manifest in the codebase. Each maps to a specific issue documented in `.planning/codebase/CONCERNS.md`:

| CONCERNS.md Issue | Pitfall | Severity | Priority |
|---|---|---|---|
| `release/release.go` — unused HTTP request, lost headers | Pitfall 9 (orphaned exports) + Pitfall 13 (no context) | Critical | Immediate |
| `release/release.go` — response body not closed | Resource leak (general Go safety, covered by golang-safety skill) | High | Immediate |
| `config/provider.go` — silent error swallowing | Pitfall 10 (error handling) variant — errors should be returned | High | Next |
| `release/release.go` — no HTTP timeouts | Pitfall 13 (no context) | Medium | Next |
| Duplicate `GetEnv` functions | Pitfall 8 (poor naming/org) + code duplication | Low | Soon |
| `mid/machineid_linux.go` — whitespace not trimmed | Pitfall 17 (filesystem assumptions) | Medium | Soon |
| `config/environment` — panic-recovery instead of errors | Pitfall 16 (init-like patterns) + unsafe error handling | Medium | Next |
| `time_tools/parser.go` — global lock contention | Pitfall 4 (global state) — performance variant | Low | Later |
| `config/provider.go` — lock double-fetch race | Pitfall 4 (global state) — concurrency variant | High | Next |

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|---|---|---|
| New package creation | Pitfall 5 (wrong major version), Pitfall 8 (bad naming) | Start at v0.x.x, choose a descriptive name, use functional options from day one |
| API extension (new features) | Pitfall 1 (changing signatures), Pitfall 3 (interface pollution) | Add new functions, prefer config structs, keep interfaces minimal |
| Error handling design | Pitfall 10 (wrapping implementation errors) | Use `%v` for dependency errors, define your own sentinels |
| Concurrency support | Pitfall 13 (no context), Pitfall 7 (value receivers on mutex structs) | Accept `context.Context` first param, use `*T` receivers |
| Testing and documentation | Pitfall 12 (no example tests), Pitfall 14 (no API compat tests) | Write Example tests, add `gorelease` to CI |
| Cross-platform support | Pitfall 17 (filesystem assumptions) | Use build tags, accept io.Reader, document platform assumptions |
| v1 stable release | Pitfall 5 (semver violations) | Run `gorelease` before tagging, audit exported surface |
| Dependency management | Pitfall 4 (global state coupling) | Prefer interfaces over concrete logger/metrics types |

---

## Sources

- [Go Blog: Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility) — HIGH confidence, official Go team guidance
- [Go Blog: Go Modules: v2 and Beyond](https://go.dev/blog/v2-go-modules) — HIGH confidence
- [Go Blog: Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors) — HIGH confidence
- [Go Blog: Module Version Numbering](https://go.dev/doc/modules/version-numbers) — HIGH confidence
- [Dave Cheney: Functional Options for Friendly APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) — HIGH confidence, established Go design pattern
- [Dave Cheney: SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design) — HIGH confidence
- [Dave Cheney: Avoid Package Names Like base, util, or common](https://dave.cheney.net/2019/01/08/avoid-package-names-like-base-util-or-common) — HIGH confidence
- [Dave Cheney: Use Internal Packages to Reduce Public API Surface](https://dave.cheney.net/2019/10/06/use-internal-packages-to-reduce-your-public-api-surface) — HIGH confidence
- [Dave Cheney: Don't Force Allocations on the Callers of Your API](https://dave.cheney.net/2019/09/05/dont-force-allocations-on-the-callers-of-your-api) — HIGH confidence
- [Dave Cheney: Should Methods Be Declared on T or *T](https://dave.cheney.net/2016/03/19/should-methods-be-declared-on-t-or-t) — HIGH confidence
- [Dave Cheney: Package Level Logger Anti-Pattern](https://dave.cheney.net/2017/01/23/the-package-level-logger-anti-pattern) — HIGH confidence
- [Go Code Style skill](/.agents/skills/golang-code-style/SKILL.md) — Community best practice
- [Go Design Patterns skill](/.agents/skills/golang-design-patterns/SKILL.md) — Community best practice
- [Go Safety skill](/.agents/skills/golang-safety/SKILL.md) — Community best practice
- [Go Security skill](/.agents/skills/golang-security/SKILL.md) — Community best practice
- [CONCERNS.md](/.planning/codebase/CONCERNS.md) — Project-specific verified issues
