package projectprobe

import "os"

// ignoreDirs lists directory names that never count as project content
// (D-10). Matching is exact-case on all platforms (D-11).
var ignoreDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	".git":         {},
	"dist":         {},
	".idea":        {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
	"target":       {},
	"build":        {},
	"bin":          {},
	"obj":          {},
	".cache":       {},
}

// isIgnoredDir reports whether name is an ignored directory name.
// The match is exact-case; no normalization is applied (D-11).
func isIgnoredDir(name string) bool {
	_, ok := ignoreDirs[name]
	return ok
}

// hasContent reports whether entries contain anything that counts as project
// content: a file, or a directory not on the ignore list (D-12). It is false
// when entries is empty or holds only ignored directories.
func hasContent(entries []os.DirEntry) bool {
	for _, entry := range entries {
		if entry.IsDir() && isIgnoredDir(entry.Name()) {
			continue
		}

		return true
	}

	return false
}
