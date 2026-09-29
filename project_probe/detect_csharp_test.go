package projectprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// t.Parallel is safe in this file: every test writes t.TempDir fixtures only
// and never mutates the package-level detectors slice (AGENTS.md global-state
// rule — the registry tests are the ones that must stay serial).

// TestProbe_CSharpEndToEnd exercises the full chain through Probe: registry →
// detectCSharp → readFirstManifest (.csproj discovery) → readManifest →
// xml.Unmarshal → merge. A folder with an SDK-style MyApp.csproj yields
// LanguageCSharp with AssemblyName/Version/Description verbatim (SC1, D-01/
// D-03). The fixture is named "MyApp.csproj", NEVER ".csproj" — the manifest
// name is project-specific (Pitfall 2).
func TestProbe_CSharpEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <AssemblyName>MyApp</AssemblyName>
    <Version>1.2.3</Version>
    <Description>A CLI for acme.</Description>
  </PropertyGroup>
</Project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "MyApp", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
	assert.Equal(t, "A CLI for acme.", data.Description)
}

// TestProbe_CSharpOldStyleNamespaced pins D-01 / probe row B: an old-style
// .csproj with xmlns="...msbuild/2003" decodes through the SAME struct —
// XMLName local-name matching is namespace-agnostic, so the expectations
// match the SDK-style row exactly.
func TestProbe_CSharpOldStyleNamespaced(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Legacy.csproj"),
		[]byte(`<?xml version="1.0" encoding="utf-8"?>
<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <PropertyGroup>
    <AssemblyName>Legacy</AssemblyName>
    <Version>2.0.0</Version>
  </PropertyGroup>
</Project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "Legacy", data.Name)
	assert.Equal(t, "2.0.0", data.Version)
}

// TestProbe_CSharpBOM pins SC1 "with BOM" (D-02): a .csproj prefixed with the
// literal UTF-8 BOM bytes (0xEF 0xBB 0xBF) parses identically — readManifest
// strips the BOM upstream, and xml.Unmarshal tolerates it directly too.
func TestProbe_CSharpBOM(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte("\xEF\xBB\xBF"+`<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <AssemblyName>MyApp</AssemblyName>
    <Version>1.2.3</Version>
  </PropertyGroup>
</Project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "MyApp", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
}

// TestProbe_CSharpMalformedPresence pins D-09: a malformed .csproj (unclosed
// XML) still matches on presence — LanguageCSharp with folder-base Name and
// empty Version/Description. xml.Unmarshal returns an error, never panics;
// the decode-error-ignored inline decode degrades to the zero struct.
func TestProbe_CSharpMalformedPresence(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Broken.csproj"),
		[]byte(`<Project><PropertyGroup><AssemblyName>X`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
	assert.Equal(t, "", data.Description)
}

// TestProbe_CSharpRootScope pins D-04 / SC4: a .csproj in a subdirectory
// never triggers the parent folder — discovery is a single-level ReadDir.
func TestProbe_CSharpRootScope(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(parent, "sub"), 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(parent, "sub", "MyApp.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>MyApp</AssemblyName></PropertyGroup></Project>`),
		0o600,
	))

	data, err := Probe(parent)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestDetectCSharp_MissingManifest pins the D-09 never-fail degrade at the
// detector level: a folder without any .csproj yields (ProjectData{}, false),
// never an error.
func TestDetectCSharp_MissingManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	pd, ok := detectCSharp(folder)
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}