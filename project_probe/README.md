# project_probe

Best-effort detection of what a folder holds: `Probe` reads the canonical
manifest of seven ecosystems (Go, Python, C#/.NET, JavaScript, Rust, Java, PHP)
and reports the project's language, name, version, and description. Standard
library only, no configuration, no build step.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/project_probe)

## Install

```bash
go get github.com/guionardo/go/project_probe
```

## Usage

The package is imported as `project_probe` and used as `projectprobe`:

```go
import "github.com/guionardo/go/project_probe"

data, err := projectprobe.Probe("/path/to/project")
if err != nil {
    // hard folder-level I/O failure (missing, not a directory, permission)
}
fmt.Println(data.Folder, data.Language, data.Name, data.Version)
```

`ProjectData` carries `Folder`, `Language`, `Name`, `Version`, and `Description`.

## The never-fail contract

`Probe` never fails on content — it degrades. Empty folders, folders holding only
ignored directories, unreadable manifests, and simply unrecognized content all
return `ProjectData` with `Language: LanguageUnknown` and a **nil** error.

Errors are reserved for hard folder-level I/O failures. Each of the three
sentinels is wrapped with the failing path:

| Sentinel | When |
|----------|------|
| `ErrFolderNotFound` | The folder does not exist. An empty argument also returns this and never probes the current directory. |
| `ErrNotDirectory` | The path exists but is not a directory. |
| `ErrPermissionDenied` | `EACCES` or `EPERM` on the folder read. |

Any other I/O error is returned wrapped with the path but matches none of the
three sentinels.

`ProjectData.Folder` is `filepath.Clean` of the argument as given: relative input
stays relative and is never absolutized.

## Detection order

Checks are root-scoped and first match wins. The order is fixed:

| # | Manifest in the folder root | Reported `Language` |
|---|-----------------------------|---------------------|
| 1 | `go.mod` | `LanguageGo` (`"Go"`) |
| 2 | `pyproject.toml` | `LanguagePython` (`"Python"`) |
| 3 | `*.csproj` | `LanguageCSharp` (`"C#/.NET"`) |
| 4 | `package.json` | `LanguageJavaScript` (`"JavaScript"`) |
| 5 | `Cargo.toml` | `LanguageRust` (`"Rust"`) |
| 6 | `pom.xml`, else `settings.gradle` | `LanguageJava` (`"Java"`) |
| 7 | `composer.json` | `LanguagePHP` (`"PHP"`) |

There is no distinct TypeScript or Kotlin value: a TypeScript-shaped
`package.json` reports `JavaScript`, and a Kotlin project reports `Java`
(`settings.gradle` is only consulted when `pom.xml` is absent, and only the
Groovy `rootProject.name` form is read).

Match rules differ by detector. `pyproject.toml`, `Cargo.toml`, `.csproj`, and
`pom.xml` match on the presence of a readable manifest, so a malformed one still
reports its language with whatever fields decoded (often empty). `package.json`
and `composer.json` must decode as JSON to match; a malformed one falls through
to the next detector. A `.csproj` folder with several project files uses the
first readable one in filename order.

## Metadata

| Language | Name | Version | Description |
|----------|------|---------|-------------|
| Go | `module` line, else folder base | `go` directive verbatim (toolchain floor; `""` when absent) | README paragraph (`go.mod` has no description) |
| Python | `[project].name`, else `[tool.poetry].name`, else folder base | `[project].version` verbatim; a `dynamic` version degrades to `""` | `description`, else README |
| C#/.NET | `AssemblyName` → `RootNamespace` → folder base | `Version` → `VersionPrefix` → `""` | `Description`, else README |
| JavaScript | `name`, else folder base | `version` verbatim; `""` when absent | `description`, else README |
| Rust | `[package].name`, else folder base | `[package].version`; `version.workspace = true` degrades to `""` | `description`, else README |
| Java | `<name>` → `<artifactId>` → folder base (or `rootProject.name`) | `<version>` → `<parent><version>` (single level, same file) → `""` | `<description>`, else README |
| PHP | full `vendor/package` `name`, else folder base | `version` verbatim; `""` when absent | `description`, else README |

`Version` is always the raw manifest string — never normalized, never resolved,
never fabricated. Ecosystem defaults are deliberately not reported: MSBuild's
implicit `1.0.0` and Maven's Super POM `4.0.0` yield `""`. Placeholders are
reported raw (`$(...)` in MSBuild, `${...}` in Maven). XML element text is
trimmed at decode time; that is decode hygiene, not normalization.

The README fallback takes the first real paragraph of `README.md` → `README.rst`
→ `README` (exact-case, first readable wins), skipping badges, TOC links,
headings, setext underlines, and HTML comments.

## Ignored content

A folder containing only ignored directories counts as empty. The ignore list is
exact-case: `node_modules`, `vendor`, `.git`, `dist`, `.idea`, `.venv`, `venv`,
`__pycache__`, `target`, `build`, `bin`, `obj`, `.cache`. Any file counts as
content.

## Limitations

- Root-scoped only: detectors read the folder's own manifests. There is no
  recursion and no sibling or parent manifest lookup.
- Manifests are capped at 1 MB, read with a UTF-8 BOM strip, and must be regular
  files — a FIFO or device named `go.mod` is rejected instead of blocking the
  probe.
- A detector panic is recovered and treated as a non-match, so it can never turn
  into a `Probe` error.
