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

// TestProbe_CSharpMultiGroup pins Pitfall 3 / probe row J: MSBuild files
// legitimately carry several PropertyGroups (Debug/Release Condition groups),
// and Version can sit in the LAST one. Collect-first-then-chain across ALL
// groups in document order is load-bearing (D-disc-6): AssemblyName from
// group 1 and Version from group 3 are both collected.
func TestProbe_CSharpMultiGroup(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup Condition="'$(Configuration)' == 'Debug'">
    <AssemblyName>MyApp</AssemblyName>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <PropertyGroup Condition="'$(Configuration)' == 'Release'">
    <Optimize>true</Optimize>
  </PropertyGroup>
  <PropertyGroup>
    <Version>3.1.4</Version>
  </PropertyGroup>
</Project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "MyApp", data.Name)
	assert.Equal(t, "3.1.4", data.Version)
}

// TestProbe_CSharpVersionPrefixChain pins D-03 / DATA-03: with only
// VersionPrefix present, Version reports its value; with NEITHER Version nor
// VersionPrefix, Version is "" — the MSBuild-implicit 1.0.0 default is never
// fabricated.
func TestProbe_CSharpVersionPrefixChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantVer string
	}{
		{
			"version_prefix_only",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><VersionPrefix>2.1.0</VersionPrefix></PropertyGroup></Project>`,
			"2.1.0",
		},
		{
			"neither_version_nor_prefix",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>MyApp</AssemblyName></PropertyGroup></Project>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(folder, "MyApp.csproj"), []byte(tt.content), 0o600))

			data, err := Probe(folder)
			require.NoError(t, err)
			assert.Equal(t, LanguageCSharp, data.Language)
			assert.Equal(t, tt.wantVer, data.Version)
		})
	}
}

// TestProbe_CSharpRootNamespaceChain pins D-03 / DATA-02: with only
// RootNamespace present, Name reports it; with neither AssemblyName nor
// RootNamespace, Name falls back to the folder base.
func TestProbe_CSharpRootNamespaceChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  string
		wantName string // "" → filepath.Base(folder)
	}{
		{
			"root_namespace_only",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><RootNamespace>Acme.Tool</RootNamespace></PropertyGroup></Project>`,
			"Acme.Tool",
		},
		{
			"neither_assembly_nor_root_namespace",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><Version>1.0.0</Version></PropertyGroup></Project>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(folder, "MyApp.csproj"), []byte(tt.content), 0o600))

			data, err := Probe(folder)
			require.NoError(t, err)
			assert.Equal(t, LanguageCSharp, data.Language)
			wantName := tt.wantName
			if wantName == "" {
				wantName = filepath.Base(folder)
			}
			assert.Equal(t, wantName, data.Name)
		})
	}
}

// TestProbe_CSharpPlaceholderRaw pins A2 / DATA-03: an MSBuild $(...)
// placeholder version is reported RAW, never resolved, never stripped — there
// is no code branch for placeholder forms (Phase 12 no-branch lesson).
func TestProbe_CSharpPlaceholderRaw(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><Version>$(MyVersion)</Version></PropertyGroup></Project>`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "$(MyVersion)", data.Version)
}

// TestProbe_CSharpDescriptionFromManifest pins A1 / DATA-04: when the csproj
// carries a <Description>, it wins over the README first paragraph — the
// full DATA-04 chain is manifest first.
func TestProbe_CSharpDescriptionFromManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>MyApp</AssemblyName><Description>Manifest description.</Description></PropertyGroup></Project>`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("README paragraph.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "Manifest description.", data.Description)
}

// TestProbe_CSharpDescriptionReadmeFallback pins A1 / DATA-04: without a
// <Description> element, the README first real paragraph fires (badge lines
// are skipped by readmeDescription).
func TestProbe_CSharpDescriptionReadmeFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>MyApp</AssemblyName></PropertyGroup></Project>`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\nA C# library.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "A C# library.", data.Description)
}

// TestProbe_CSharpSubdirNamedCsproj pins D-04 root-scope hygiene: a DIRECTORY
// named foo.csproj in the root is skipped by e.IsDir() inside
// readFirstManifest — it can never trigger a match.
func TestProbe_CSharpSubdirNamedCsproj(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(folder, "foo.csproj"), 0o700))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestProbe_CSharpEmptyFile pins D-09 presence-match: an EMPTY (0-byte)
// .csproj still claims LanguageCSharp — xml.Unmarshal("") errors, the zero
// struct degrades to folder-base Name and empty fields.
func TestProbe_CSharpEmptyFile(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, "MyApp.csproj"), nil, 0o600))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
}

// TestDetectCSharp pins the detector-level edges (detect_rust_test.go table
// shape with a folder-setup column): folder-base edge (nested temp subdir →
// short base) and the old-style + BOM combined row (belt-and-suspenders).
// A wantName of "" means the row expects the filepath.Base(folder) fallback.
//
// FIFO candidate rejection is NOT re-pinned here: FIFO fixtures are
// Unix-only (manifest_fifo_test.go carries the FIFO-creating helper under a
// !windows build tag), readManifest's WR-01 gate is already pinned by
// manifest_fifo_test.go, and a cross-platform FIFO row in an untagged file
// would break the Windows CI build. First-readable-candidate continues past a
// rejected entry by construction of the readFirstManifest loop (A9).
func TestDetectCSharp(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		folder   func(t *testing.T) string // nil → t.TempDir()
		wantName string                    // "" → filepath.Base(folder)
		wantVer  string
		wantDesc string
	}{
		{
			// DATA-02 edge: a folder nested under a subdirectory yields the
			// short base name, not the full temp path.
			"folder_base_nested",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><Version>1.0.0</Version></PropertyGroup></Project>`,
			func(t *testing.T) string {
				sub := filepath.Join(t.TempDir(), "sub")
				require.NoError(t, os.Mkdir(sub, 0o700))

				return sub
			},
			"",
			"1.0.0",
			"",
		},
		{
			// Belt-and-suspenders: old-style namespaced root AND a BOM prefix
			// in the same file — both boundaries handled at once.
			"old_style_and_bom",
			"\xEF\xBB\xBF" + `<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003"><PropertyGroup><AssemblyName>Legacy</AssemblyName><Version>2.0.0</Version></PropertyGroup></Project>`,
			nil,
			"Legacy",
			"2.0.0",
			"",
		},
		{
			// 13 WR-01: padded XML element text is decode-hygiene-trimmed —
			// values report trimmed, never verbatim with surrounding spaces.
			"padded_values",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>  MyApp  </AssemblyName><Version> 1.2.3 </Version></PropertyGroup></Project>`,
			nil,
			"MyApp",
			"1.2.3",
			"",
		},
		{
			// 13 WR-01: a whitespace-only <Version> no longer counts as
			// present — the Version→VersionPrefix chain fires (DATA-03).
			"whitespace_only_version_falls_to_prefix",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><Version> </Version><VersionPrefix>2.0.0</VersionPrefix></PropertyGroup></Project>`,
			nil,
			"",
			"2.0.0",
			"",
		},
		{
			// 13 WR-01: a whitespace-only <AssemblyName> no longer counts as
			// present — the folder-base Name fallback fires (DATA-02).
			"whitespace_only_assembly_name",
			`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName> </AssemblyName></PropertyGroup></Project>`,
			nil,
			"",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			if tt.folder != nil {
				folder = tt.folder(t)
			}
			require.NoError(t, os.WriteFile(filepath.Join(folder, "MyApp.csproj"), []byte(tt.content), 0o600))

			pd, ok := detectCSharp(folder)
			require.True(t, ok)
			assert.Equal(t, LanguageCSharp, pd.Language)
			wantName := tt.wantName
			if wantName == "" {
				wantName = filepath.Base(folder)
			}
			assert.Equal(t, wantName, pd.Name)
			assert.Equal(t, tt.wantVer, pd.Version)
			assert.Equal(t, tt.wantDesc, pd.Description)
		})
	}
}

// TestDetectCSharp_ExactNameGuard pins 13 IN-03: a file literally named
// ".csproj" (no project name) can never match the suffix filter — the
// exact-name guard len(e.Name()) <= len(suffix) precedes the HasSuffix
// check, so the folder falls through to the next detector (DATA-03 flagged
// adjacency assumption: exactly-equal names separate, never match).
func TestDetectCSharp_ExactNameGuard(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, ".csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>X</AssemblyName></PropertyGroup></Project>`),
		0o600,
	))

	pd, ok := detectCSharp(folder)
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}