package projectprobe

import (
	"bytes"
	"encoding/xml"
	"path/filepath"
	"strings"
)

// pomManifest is the decode shape for a pom.xml. The root is the lowercase
// <project> element (XML is case-sensitive — this struct never doubles for
// .csproj, Pitfall 7); the Maven 4.0.0 namespace is tolerated via local-name
// matching (D-01, probe row F). <parent><version> lives in THIS file —
// Maven requires the parent block's version in every child POM; the child's
// own <version> may be omitted and inherits it (single level per D-05 — no
// transitive chain, Pitfall 5).
type pomManifest struct {
	XMLName     xml.Name   `xml:"project"`
	Name        string     `xml:"name"`
	ArtifactID  string     `xml:"artifactId"`
	Version     string     `xml:"version"`
	Description string     `xml:"description"`
	Parent      *parentPOM `xml:"parent"` // allocated only when <parent> present (probe row G)
}

// parentPOM is the <parent> block of a child POM. Only the version is read:
// the parent block — including its version — is REQUIRED in every child POM
// per Maven; what may be omitted is the child's own <version>, which then
// inherits it (single level, D-05 — read from the probed file only).
type parentPOM struct {
	Version string `xml:"version"`
}

// parseRootProjectName extracts the rootProject.name literal from
// settings.gradle content (D-06). The UTF-8 BOM is stripped at entry —
// strings.TrimSpace does NOT strip U+FEFF (unicode.IsSpace(0xFEFF) is
// false), so the parser boundary needs the explicit strip; readManifest
// strips upstream too, this is belt-and-suspenders for pure-parse callers
// (toml.go:27 precedent). The assignment must be an exact left-of-= key
// match ("rootProject.name" or the documented fully-qualified
// "settings.rootProject.name") — a commented line like
// "// rootProject.name = 'x'" fails the exact match and can never fabricate
// a name (Pitfall 6). The value is the first quoted literal (' or " — both
// are legal Groovy; double quotes interpolate, which we never evaluate); a
// non-empty remainder after the closing quote (e.g. 'a' + 'b') is not
// stored — never partial data (D-disc-3, Phase 12 D-disc-6 precedent).
// Lines without '=' (blank lines, unrelated directives, an empty file) are
// skipped BEFORE any slicing — the parser never panics on any input (D-09,
// parseGoMod defensive shape). First match wins (D-disc-4).
func parseRootProjectName(content []byte) string {
	for _, line := range strings.Split(string(bytes.TrimPrefix(content, utf8BOM)), "\n") {
		line = strings.TrimSpace(line)
		i := strings.Index(line, "=")
		if i < 0 {
			continue // no '=' on the line — skip before any slicing (never panics)
		}
		key := strings.TrimSpace(line[:i])
		if key != "rootProject.name" && key != "settings.rootProject.name" {
			continue // exact match only — never HasPrefix (Pitfall 6)
		}
		rest := strings.TrimSpace(line[i+1:])
		if len(rest) < 2 || (rest[0] != '\'' && rest[0] != '"') {
			continue // unquoted value — degrade
		}
		end := strings.IndexByte(rest[1:], rest[0])
		if end < 0 {
			continue // no closing quote — degrade
		}
		remainder := strings.TrimSpace(rest[1+end+1:])
		if remainder != "" && !strings.HasPrefix(remainder, "//") {
			continue // concatenation/expression — never partial (D-disc-3)
		}
		return rest[1 : 1+end]
	}
	return ""
}

// detectJavaKotlin reports a Java project when folder contains a pom.xml
// (primary) or — ONLY when no pom.xml exists — a settings.gradle with a
// rootProject.name (D-06). Language is LanguageJava for both branches
// (Kotlin-distinct value is v2 REFN-02, NOT this phase). The pom branch
// gates on manifest PRESENCE (D-09): the inline decode is
// decode-error-ignored, so a present-but-garbage pom still matches with
// empty fields — and the settings.gradle fallback does NOT fire (D-06
// asymmetry). Name: <name> → <artifactId> → folder base (D-05, DATA-02);
// Version: <version> → <parent><version> when absent (D-05, single level,
// same file); Description: <description> → README → empty (DATA-04).
// Root-scoped only (D-07); no sibling or parent POM files are ever read
// (ROBT-05). The Kotlin-DSL settings variant is NOT read (A5, D-06 literal).
func detectJavaKotlin(folder string) (ProjectData, bool) {
	data := ProjectData{Language: LanguageJava} // "Java" (project.go:45)

	if content, ok := readManifest(folder, "pom.xml"); ok { // D-09 presence gate
		var pom pomManifest
		_ = xml.Unmarshal(content, &pom) // decode error → whatever decoded before the error point survives; the rest degrades to empty (IN-01)

		// 13 WR-01: every decoded field is whitespace-trimmed BEFORE chain
		// resolution (decode hygiene, NOT version normalization) — padded
		// values report trimmed and whitespace-only elements no longer count
		// as present, so the D-05/DATA-02/DATA-03 fallbacks fire.
		data.Name = strings.TrimSpace(pom.Name) // <name> — Maven's optional display name
		if data.Name == "" {
			data.Name = strings.TrimSpace(pom.ArtifactID) // <artifactId> — the required coordinate
		}
		if data.Name == "" {
			data.Name = filepath.Base(folder) // DATA-02
		}
		data.Version = strings.TrimSpace(pom.Version)
		if data.Version == "" && pom.Parent != nil {
			data.Version = strings.TrimSpace(pom.Parent.Version) // D-05: single-level parent inheritance (same file)
		}
		data.Description = strings.TrimSpace(pom.Description)
		if data.Description == "" {
			data.Description = readmeDescription(folder) // DATA-04
		}
		return data, true
	}

	if content, ok := readManifest(folder, "settings.gradle"); ok { // D-06: ONLY when no pom.xml
		data.Name = parseRootProjectName(content)
		if data.Name == "" {
			data.Name = filepath.Base(folder) // DATA-02
		}
		data.Description = readmeDescription(folder) // settings.gradle has no description field (DATA-04)
		return data, true
	}

	return ProjectData{}, false
}
