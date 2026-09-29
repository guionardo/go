// Package projectprobe provides project detection for local folders.
//
// Probe is the single entry point. It inspects a folder and returns
// ProjectData — the folder's canonical path, its language, and any metadata
// the winning detector reports (name, version, description).
//
// # Never-fail contract
//
// Probe never fails on content. Empty folders, folders holding only ignored
// directories, and folders with unrecognized content all return ProjectData
// with LanguageUnknown and a nil error. Errors are reserved for hard
// folder-level I/O failures only — ErrFolderNotFound, ErrNotDirectory, and
// ErrPermissionDenied — and are always wrapped with the failing folder path.
//
// The returned Folder field is filepath.Clean(as-given): relative input
// stays relative and is never absolutized. An empty folder argument returns
// ErrFolderNotFound and never probes the current directory.
//
// Usage:
//
//	import "github.com/guionardo/go/project_probe"
//
//	data, err := projectprobe.Probe("/path/to/project")
//	if err != nil {
//	    // hard folder-level I/O failure (missing, not a directory, permission)
//	}
//	// data.Language is LanguageUnknown when the content is unrecognized
//
// Sentinel errors (wrapped with the failing folder path):
//
//	var ErrFolderNotFound   = errors.New("probe: folder not found")
//	var ErrNotDirectory     = errors.New("probe: not a directory")
//	var ErrPermissionDenied = errors.New("probe: permission denied")
//
// Detector implementations land across phases 11-13: the ordered registry
// (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) currently holds
// live detectors at positions 0 (Go), 1 (Python), 3 (JS/TS), 4 (Rust), and 6
// (PHP); slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13. A folder with
// go.mod reports LanguageGo, with Name from the module line (folder-base
// fallback), Version from the go directive (toolchain floor, never
// normalized), and Description from the README first real paragraph via the
// readmeDescription fallback; a folder with a parseable package.json reports
// LanguageJavaScript with Name/Version/Description from the manifest
// (DETC-03); a folder with a parseable composer.json reports LanguagePHP
// with the full vendor/package name (DETC-04); a folder with a pyproject.toml
// reports LanguagePython with [project] (or legacy [tool.poetry]) fields; a
// folder with a Cargo.toml reports LanguageRust with [package] fields,
// workspace-inherited versions degrading to empty. Content-bearing folders
// without a matching manifest still yield LanguageUnknown.
package projectprobe
