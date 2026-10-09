package projectprobe

import "encoding/json"

// readJSONManifest decodes <folder>/<name> as JSON into v. Missing files,
// oversized files, and malformed or type-mismatched JSON all return false —
// callers degrade to a non-match so the cascade continues (D-04, D-12).
// The BOM strip inside readManifest is REQUIRED: encoding/json rejects a
// leading UTF-8 BOM ("invalid character '\ufeff' looking for beginning of
// value"), so a BOM'd manifest would otherwise never parse (Pitfall 1).
func readJSONManifest(folder, name string, v any) bool {
	content, ok := readManifest(folder, name)
	if !ok {
		return false
	}
	return json.Unmarshal(content, v) == nil
}
