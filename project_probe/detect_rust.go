package projectprobe

import (
	"path/filepath"
)

// detectRust reports a Rust project when folder contains a Cargo.toml
// (D-09: presence-based match — malformed TOML still matches, fields degrade
// to empty). Metadata comes from [package]: name, version, description
// (D-06). Workspace-inherited fields — version.workspace = true,
// description.workspace = true — are dotted keys the subset reader classifies
// as unsupported, so they degrade to "" (D-07: never fabricated, never
// resolved from a workspace root). A virtual workspace manifest ([workspace]
// without [package]) still matches on presence; Name falls back to the folder
// base. Content flows exclusively through readManifest (1 MB cap + BOM strip
// + WR-01 FIFO gate); any read failure degrades to a non-match, never an
// error (D-09).
func detectRust(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "Cargo.toml")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	fields := readTOMLSection(content, "package") // D-06: single section read
	data := ProjectData{Language: LanguageRust}
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = fields["version"] // "" for version.workspace = true (D-07 — never resolved)
	data.Description = fields["description"]
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 — also for description.workspace = true
	}
	return data, true
}
