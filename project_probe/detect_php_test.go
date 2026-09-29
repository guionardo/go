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

// TestProbe_PHPEndToEnd exercises the full chain through Probe: registry →
// detectPHP → readJSONManifest → merge. A folder with a well-formed
// composer.json yields LanguagePHP with Name/Version/Description from the
// manifest (DETC-04). Name is the FULL vendor/package string (D-07), never
// the last segment.
func TestProbe_PHPEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":"acme/logger","version":"1.0.0","description":"Logging library"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, "acme/logger", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
	assert.Equal(t, "Logging library", data.Description)
}

// TestProbe_PHPVersionAbsent pins the composer ecosystem norm: version is
// usually absent (Packagist infers it from VCS tags) — empty Version is
// correct, never fabricated.
func TestProbe_PHPVersionAbsent(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":"acme/logger"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, "acme/logger", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_PHPNameFallback pins the DATA-02 name chain for PHP: a manifest
// without a name falls back to the folder base.
func TestProbe_PHPNameFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"description":"d"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
}

// TestProbe_PHPDescriptionFallback pins the DATA-04 chain for PHP: a
// manifest without a description falls back to the README first real
// paragraph — the common case for project packages.
func TestProbe_PHPDescriptionFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":"acme/logger"}`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("# Logger\n\nA logging library.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, "acme/logger", data.Name)
	assert.Equal(t, "A logging library.", data.Description)
}

// TestProbe_PHPMalformed pins the D-04 mirror rule for PHP: a malformed
// composer.json alone in the folder yields LanguageUnknown — parse failure →
// false → cascade continues (no other manifest exists, so nothing matches).
func TestProbe_PHPMalformed(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte("{oops"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestProbe_PHPDuplicateKeys pins the verified encoding/json semantics for
// the PHP mirror: duplicate keys are legal and LAST wins.
func TestProbe_PHPDuplicateKeys(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":"acme/a","name":"acme/b"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, "acme/b", data.Name)
}

// TestProbe_PHPTypeMismatch pins the verified encoding/json semantics for
// the PHP mirror: a type mismatch ("name": 123) is a decode error →
// detector returns false → LanguageUnknown (D-04 cascade continues).
func TestProbe_PHPTypeMismatch(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":123}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestProbe_PHPVersionNull pins the verified encoding/json semantics:
// "version": null decodes to "" — no error, no fabricated value.
func TestProbe_PHPVersionNull(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "composer.json"),
		[]byte(`{"name":"acme/logger","version":null}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePHP, data.Language)
	assert.Equal(t, "acme/logger", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_CascadePrecedence pins the DETC-01 first-match cascade across
// the real ordered registry through Probe (Go@0 beats JS@3 beats PHP@6) and
// the D-04 parse-success rule: a broken package.json at position 3 does NOT
// match, so the cascade continues to a valid composer.json at position 6
// (T-11-09 — the RESEARCH match-rule asymmetry locked behavior).
func TestProbe_CascadePrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns the folder
		want    Language
		wantErr bool
	}{
		{
			// D-04 parse-success asymmetry: broken JS manifest falls through
			// to the valid PHP manifest — LanguagePHP wins.
			"broken_package_json_valid_composer_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "package.json"),
					[]byte("{invalid"),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "composer.json"),
					[]byte(`{"name":"acme/logger"}`),
					0o600,
				))

				return folder
			},
			LanguagePHP,
			false,
		},
		{
			// DETC-01 first-match: Go at index 0 beats JS at index 3.
			"go_mod_and_package_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "go.mod"),
					[]byte("module example.com/acme\n"),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "package.json"),
					[]byte(`{"name":"acme-widget"}`),
					0o600,
				))

				return folder
			},
			LanguageGo,
			false,
		},
		{
			// DETC-01 first-match: JS at index 3 beats PHP at index 6.
			"package_json_and_composer_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "package.json"),
					[]byte(`{"name":"acme-widget"}`),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "composer.json"),
					[]byte(`{"name":"acme/logger"}`),
					0o600,
				))

				return folder
			},
			LanguageJavaScript,
			false,
		},
		{
			// D-08 first-match across the Phase 12 detectors: Python at
			// index 1 beats Rust at index 4 — the ordered cascade proven
			// across both TOML detectors.
			"pyproject_toml_and_cargo_toml",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "pyproject.toml"),
					[]byte("[project]\nname = \"acme\"\n"),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "Cargo.toml"),
					[]byte("[package]\nname = \"acme\"\n"),
					0o600,
				))

				return folder
			},
			LanguagePython,
			false,
		},
		{
			// D-09/D-04 match-rule asymmetry: Python at index 1 matches on
			// presence (garbage or valid, the file IS the marker) and beats
			// JS at index 3, which needs parse success — a broken
			// package.json falls through to the valid pyproject.toml.
			"broken_package_json_valid_pyproject_toml",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "package.json"),
					[]byte("{invalid"),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "pyproject.toml"),
					[]byte("[project]\nname = \"acme\"\n"),
					0o600,
				))

				return folder
			},
			LanguagePython,
			false,
		},
		{
			// D-08 first-match: C#/.NET at index 2 beats JS/TS at index 3.
			"csproj_and_package_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "MyApp.csproj"),
					[]byte(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <AssemblyName>MyApp</AssemblyName>
  </PropertyGroup>
</Project>
`),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "package.json"),
					[]byte(`{"name":"acme-widget"}`),
					0o600,
				))

				return folder
			},
			LanguageCSharp,
			false,
		},
		{
			// D-08 first-match: C#/.NET at index 2 beats Java/Kotlin at
			// index 5.
			"csproj_and_pom_xml",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "MyApp.csproj"),
					[]byte(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <AssemblyName>MyApp</AssemblyName>
  </PropertyGroup>
</Project>
`),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "pom.xml"),
					[]byte(`<project><name>AcmeJava</name></project>`),
					0o600,
				))

				return folder
			},
			LanguageCSharp,
			false,
		},
		{
			// D-08 first-match: Java/Kotlin at index 5 (pom presence, D-09)
			// beats PHP at index 6.
			"pom_xml_and_composer_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "pom.xml"),
					[]byte(`<project><name>AcmeJava</name></project>`),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "composer.json"),
					[]byte(`{"name":"acme/logger"}`),
					0o600,
				))

				return folder
			},
			LanguageJava,
			false,
		},
		{
			// D-08 first-match via the D-06 FALLBACK: settings.gradle (no
			// pom.xml) at index 5 still beats PHP at index 6 — pins that the
			// gradle fallback participates in cascade order.
			"settings_gradle_and_composer_json",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "settings.gradle"),
					[]byte("rootProject.name = 'acme-tool'\n"),
					0o600,
				))
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "composer.json"),
					[]byte(`{"name":"acme/logger"}`),
					0o600,
				))

				return folder
			},
			LanguageJava,
			false,
		},
		{
			// D-06 fallback end-to-end: settings.gradle only → LanguageJava.
			"settings_gradle_only",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "settings.gradle"),
					[]byte("rootProject.name = 'acme-tool'\n"),
					0o600,
				))

				return folder
			},
			LanguageJava,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := tt.setup(t)

			data, err := Probe(folder)
			require.NoError(t, err)
			assert.Equal(t, tt.want, data.Language)
		})
	}
}
