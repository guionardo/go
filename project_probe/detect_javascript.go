package projectprobe

import "path/filepath"

// detectJS reports a JavaScript project when folder contains a package.json
// that decodes as JSON (DETC-03). The match rule is parse success (D-04): a
// malformed manifest yields a non-match so the cascade continues.
//
// The Language is ALWAYS LanguageJavaScript for a package.json — even a
// TS-shaped manifest — the distinct typescript value is v2 backlog (D-05,
// REFN-01). Name falls back to the folder base when the manifest has none
// (DATA-02). Version is the raw manifest value; "" when absent — a
// private:true manifest without a version is a zero-value behavior pinned by
// a test row, with no dedicated decode field (D-06). Description falls back
// to the README first real paragraph when the manifest has none (DATA-04).
// All content flows through readJSONManifest → readManifest (1 MB cap + BOM
// strip, ROBT-02); any read or decode failure degrades to a non-match, never
// an error (D-12).
func detectJS(folder string) (ProjectData, bool) {
	var m struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if !readJSONManifest(folder, "package.json", &m) {
		return ProjectData{}, false // D-04: malformed → cascade continues
	}

	data := ProjectData{Language: LanguageJavaScript} // D-05: always JavaScript
	data.Name = m.Name
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = m.Version // raw; "" when absent (D-06 — zero value, no code branch)
	data.Description = m.Description
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 chain
	}
	return data, true
}