package projectprobe

import "path/filepath"

// detectPHP reports a PHP project when folder contains a composer.json that
// decodes as JSON (DETC-04). The match rule is parse success (D-04 mirror): a
// malformed manifest yields a non-match so the cascade continues.
//
// Name is the FULL vendor/package string (D-07) — the ecosystem identifier,
// consistent with Go module-path reporting, never the last segment. Name
// falls back to the folder base when the manifest has none (DATA-02).
// Version is the raw manifest value; "" when absent is the ecosystem norm —
// Packagist infers versions from VCS tags, so an empty Version is correct,
// never fabricated. Description falls back to the README first real
// paragraph when the manifest has none (DATA-04). All content flows through
// readJSONManifest → readManifest (1 MB cap + BOM strip, ROBT-02); any read
// or decode failure degrades to a non-match, never an error (D-12).
func detectPHP(folder string) (ProjectData, bool) {
	var m struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if !readJSONManifest(folder, "composer.json", &m) {
		return ProjectData{}, false // D-04: malformed → cascade continues
	}

	data := ProjectData{Language: LanguagePHP}
	data.Name = m.Name // full vendor/package (D-07)
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = m.Version // raw; "" when absent — the composer norm, never fabricated
	data.Description = m.Description
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 chain
	}
	return data, true
}
