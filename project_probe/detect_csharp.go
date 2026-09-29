package projectprobe

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
)

// readFirstManifest returns the content of the first READABLE *.csproj entry
// in folder (single level, root-scoped — D-04/ROBT-05). os.ReadDir sorts by
// filename, so the choice is deterministic. Exact-case suffix matching per
// the Phase 11 D-08 precedent (a plain open alone resolves case-insensitively
// on macOS/Windows volumes — the directory entries are the source of truth).
// Multiple .csproj files (monorepo layouts) degrade to the first readable
// one; the full selection rule is deferred to v2 (REFN-04). The ignore list
// is intentionally NOT consulted — same visibility as every other detector;
// hasContent already gates Probe.
func readFirstManifest(folder, suffix string) (content []byte, ok bool) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, false
	}
	for _, e := range entries { // sorted by filename
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		if content, ok := readManifest(folder, e.Name()); ok { // FIFO/regular gate applies per candidate
			return content, true
		}
	}
	return nil, false
}

// csprojManifest is the decode shape for a .csproj file. Tags carry no
// namespace: XMLName local-name matching makes the same struct work for
// SDK-style <Project Sdk="..."> and old-style <Project xmlns="...msbuild/2003">
// (D-01, verified). Unknown elements (TargetFramework, ItemGroup, ...) are
// discarded by encoding/xml; multiple PropertyGroups collect in order.
type csprojManifest struct {
	XMLName        xml.Name        `xml:"Project"`
	PropertyGroups []propertyGroup `xml:"PropertyGroup"`
}

// propertyGroup is one MSBuild <PropertyGroup> block. Plain tags (no
// namespace) are namespace-agnostic for children too (D-01, probe row I).
type propertyGroup struct {
	AssemblyName  string `xml:"AssemblyName"`
	RootNamespace string `xml:"RootNamespace"`
	Version       string `xml:"Version"`
	VersionPrefix string `xml:"VersionPrefix"`
	Description   string `xml:"Description"`
}

// detectCSharp reports a C#/.NET project when folder holds a readable
// .csproj (D-09: presence-match — malformed XML still matches, fields
// degrade to empty). The XML decode is inline and decode-error-ignored: a
// parse failure yields the zero struct, which the chains resolve to
// fallbacks. Name: AssemblyName → RootNamespace → folder base (D-03,
// DATA-02); Version: Version → VersionPrefix → empty, never fabricated
// (D-03, DATA-03 — MSBuild would default VersionPrefix to 1.0.0, we report
// ""; $(...) placeholders flow through raw, never resolved — A2);
// Description: manifest → README first paragraph → empty (DATA-04).
// Root-scoped only: discovery is a single-level ReadDir (D-04).
func detectCSharp(folder string) (ProjectData, bool) {
	content, ok := readFirstManifest(folder, ".csproj")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	var proj csprojManifest
	_ = xml.Unmarshal(content, &proj) // decode error → zero struct → presence-match degrade

	// Collect-first-then-chain (D-disc-6): first non-empty per field across
	// ALL PropertyGroups in document order — AssemblyName priority must not
	// depend on which group holds it (probe row J).
	var assemblyName, rootNamespace, version, versionPrefix, description string
	for _, pg := range proj.PropertyGroups {
		if assemblyName == "" {
			assemblyName = pg.AssemblyName
		}
		if rootNamespace == "" {
			rootNamespace = pg.RootNamespace
		}
		if version == "" {
			version = pg.Version
		}
		if versionPrefix == "" {
			versionPrefix = pg.VersionPrefix
		}
		if description == "" {
			description = pg.Description
		}
	}

	data := ProjectData{Language: LanguageCSharp} // "C#/.NET"
	data.Name = assemblyName
	if data.Name == "" {
		data.Name = rootNamespace
	}
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02
	}
	data.Version = version
	if data.Version == "" {
		data.Version = versionPrefix // D-03 chain; "" when both absent (never 1.0.0 fabrication)
	}
	data.Description = description
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04
	}
	return data, true
}