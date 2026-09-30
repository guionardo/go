package projectprobe

import (
	"path/filepath"
)

// detectPython reports a Python project when folder contains a pyproject.toml
// (D-09: presence-based match — malformed TOML still matches, fields degrade
// to empty). Metadata comes from [project] (PEP 621: name/version/description)
// with [tool.poetry] as the legacy whole-section fallback when [project]
// yields an empty map (D-04/D-disc-3 — no per-field mixing across sections).
// Content flows exclusively through readManifest (1 MB cap + BOM strip +
// WR-01 FIFO gate); any read failure degrades to a non-match, never an error
// (D-09).
func detectPython(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "pyproject.toml")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	fields := readTOMLSection(content, "project")
	if len(fields) == 0 {
		fields = readTOMLSection(content, "tool.poetry") // D-04 legacy fallback
	}
	data := ProjectData{Language: LanguagePython}
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = fields["version"] // raw string or "" (DATA-03 — dynamic/unsupported → "", never fabricated)
	data.Description = fields["description"]
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 chain
	}
	return data, true
}