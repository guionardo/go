package projectprobe

import (
	"path/filepath"
	"strings"
)

// detectGo reports a Go project when folder contains a go.mod (D-disc-1:
// presence-based match — a go.mod with zero parseable directives still
// matches, the tool itself assumes go 1.16 when the directive is absent and
// we report "" instead of fabricating, D-02). Content flows exclusively
// through readManifest (1 MB cap + BOM strip, ROBT-02); any read failure
// degrades to a non-match, never an error (D-12).
func detectGo(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "go.mod")
	if !ok {
		return ProjectData{}, false // D-12: never-fail degrade
	}

	name, version := parseGoMod(content)
	data := ProjectData{Language: LanguageGo}
	data.Name = name
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = version                       // raw go directive or "" (D-02 — toolchain floor, never normalized)
	data.Description = readmeDescription(folder) // go.mod has no description field (D-03)
	return data, true
}

// parseGoMod extracts the module path and go directive from go.mod content
// (Pattern 2, verified grammar). Lines are trimmed and tokenized with
// strings.Fields; a trailing "//" comment is stripped before tokenizing (the
// tool bans "/* */" and "//" cannot occur inside a valid module path, A2).
// Directives match on EXACT first-token equality, so "modulex y", "gopher
// 1.2", and "toolchain go1.26.4" never match (Pitfall 6). The module value
// has only its outer quotes stripped (D-01 — both quote kinds, backtick
// leniency A4; interior escapes are never unescaped, documented leniency);
// the go value is reported verbatim, never normalized (D-02). First
// occurrence wins per directive. The block form "module ( path )" is
// intentionally unparsed (A3 — Name falls back to the folder base). Returns
// ("", "") when neither directive is present.
func parseGoMod(content []byte) (modulePath, goVersion string) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i]) // trailing comment
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "module":
			if modulePath == "" {
				modulePath = strings.Trim(fields[1], `"`+"`")
			}
		case "go":
			if goVersion == "" {
				goVersion = fields[1] // raw string, never normalized (D-02/DATA-03)
			}
		}
	}
	return modulePath, goVersion
}
