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

// TestProbe_PythonEndToEnd exercises the full chain through Probe: registry →
// detectPython → readTOMLSection([project]) → merge. A folder with a
// pyproject.toml carrying a PEP 621 [project] section yields LanguagePython
// with the name/version/description verbatim (SC1, D-04).
func TestProbe_PythonEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A Python library.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
	assert.Equal(t, "A Python library.", data.Description)
}

// TestProbe_PythonPoetryLegacy pins SC2 / D-04: a pyproject.toml with only
// [tool.poetry] (Poetry 1.x layout) still matches as Python and reports the
// legacy name/version/description.
func TestProbe_PythonPoetryLegacy(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[tool.poetry]\nname = \"poetry-app\"\nversion = \"3.4.5\"\ndescription = \"A poetry app.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "poetry-app", data.Name)
	assert.Equal(t, "3.4.5", data.Version)
	assert.Equal(t, "A poetry app.", data.Description)
}

// TestProbe_PythonProjectWins pins D-disc-3 whole-section precedence: when
// BOTH [project] and [tool.poetry] are present (migrated Poetry 2.x file),
// the [project] values win — no per-field mixing across sections.
func TestProbe_PythonProjectWins(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"modern\"\nversion = \"2.0.0\"\ndescription = \"Modern pyproject.\"\n\n"+
			"[tool.poetry]\nname = \"legacy\"\nversion = \"1.0.0\"\ndescription = \"Legacy poetry.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "modern", data.Name)
	assert.Equal(t, "2.0.0", data.Version)
	assert.Equal(t, "Modern pyproject.", data.Description)
}

// TestProbe_PythonDynamicVersion pins DATA-03: dynamic = ["version"] is an
// array the reader degrades, so Version is "" — never fabricated. There is
// no code branch for dynamic fields; the degrade is structural.
func TestProbe_PythonDynamicVersion(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"acme\"\ndynamic = [\"version\"]\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_PythonPresence pins the D-09 presence-match rule: a garbage
// pyproject.toml (no parseable TOML at all) still claims LanguagePython —
// contrast with the JS parse-success rule — with folder-base Name and empty
// fields (D-09, DATA-02).
func TestProbe_PythonPresence(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("not a toml\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
	assert.Equal(t, "", data.Description)
}

// TestProbe_PythonNameFallback pins the DATA-02 name chain for Python: a
// [project] with version but no name (malformed per PEP 621, but real files
// exist) still matches and Name falls back to the folder base.
func TestProbe_PythonNameFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nversion = \"1.0.0\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "1.0.0", data.Version)
}

// TestProbe_PythonDescriptionFallback pins the DATA-04 chain for Python: a
// [project] without description falls back to the README first real
// paragraph (badges are skipped by readmeDescription).
func TestProbe_PythonDescriptionFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"acme\"\nversion = \"1.0.0\"\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\n# Acme\n\nA Python library.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
	assert.Equal(t, "A Python library.", data.Description)
}

// TestDetectPython_MissingManifest pins the D-09 never-fail degrade at the
// detector level: a folder without pyproject.toml yields (ProjectData{},
// false), never an error.
func TestDetectPython_MissingManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	pd, ok := detectPython(folder)
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}