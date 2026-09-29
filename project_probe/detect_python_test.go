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

// TestProbe_PythonOptionalDepsIsolation pins P2 sub-table isolation for the
// [project] arm: keys inside [project.optional-dependencies] (a sibling
// sub-table per PEP 621) must never leak into the [project] read — the
// reader's exact-header equality is load-bearing.
func TestProbe_PythonOptionalDepsIsolation(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"acme\"\nversion = \"1.0.0\"\n\n"+
			"[project.optional-dependencies]\nname = \"evil\"\ntest = [\"pytest\"]\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
}

// TestProbe_PythonPoetryDepsIsolation pins P2 sub-table isolation for the
// [tool.poetry] arm: legacy files always carry [tool.poetry.dependencies] —
// its entries must never leak into the poetry read.
func TestProbe_PythonPoetryDepsIsolation(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[tool.poetry]\nname = \"poet\"\nversion = \"1.0.0\"\n\n"+
			"[tool.poetry.dependencies]\nrequests = \"^2.13.0\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "poet", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
}

// TestProbe_PythonEmptyProjectFallsToPoetry pins D-disc-3: a [project] whose
// values ALL degrade (name = 123 is unquoted → never stored) yields an empty
// map, so the len==0 guard falls back to [tool.poetry] — "section present
// but everything unsupported" is covered by the same guard as "section
// absent".
func TestProbe_PythonEmptyProjectFallsToPoetry(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = 123\n\n[tool.poetry]\nname = \"poet\"\nversion = \"2.0.0\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "poet", data.Name)
	assert.Equal(t, "2.0.0", data.Version)
}

// TestProbe_PythonBOM pins Pitfall 1: a pyproject.toml prefixed with the
// literal UTF-8 BOM bytes (0xEF 0xBB 0xBF) parses identically — readManifest
// strips the BOM upstream, and the reader tolerates it directly too.
func TestProbe_PythonBOM(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("\xEF\xBB\xBF[project]\nname = \"acme\"\nversion = \"1.0.0\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
}

// TestProbe_PythonCRLF pins P5: Windows-edited pyproject.toml with \r\n line
// endings parses identically — the reader's per-line TrimSpace handles \r.
func TestProbe_PythonCRLF(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\r\nname = \"acme\"\r\nversion = \"1.0.0\"\r\ndescription = \"CRLF file.\"\r\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
	assert.Equal(t, "CRLF file.", data.Description)
}

// TestProbe_PythonPoetryPackagesTable pins P4 at detector level: an inline
// table value (packages = [{include = "acme"}]) degrades without disturbing
// the poetry read — Name stays intact.
func TestProbe_PythonPoetryPackagesTable(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[tool.poetry]\nname = \"poet\"\npackages = [{include = \"acme\"}]\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, "poet", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_PythonPoetryNameAbsent pins the DATA-02 chain on the poetry arm:
// a [tool.poetry] without name still matches, Name falls back to the folder
// base.
func TestProbe_PythonPoetryNameAbsent(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[tool.poetry]\nversion = \"2.0.0\"\ndescription = \"No name here.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "2.0.0", data.Version)
	assert.Equal(t, "No name here.", data.Description)
}

// TestDetectPython pins the detector-level edges (detect_go_test.go table
// shape with a folder-setup column): folder-base edges (nested temp subdir →
// short base), poetry-description README fallback (DATA-04 on the poetry
// arm), and empty-file presence match (D-09). A wantName of "" means the row
// expects the filepath.Base(folder) fallback.
func TestDetectPython(t *testing.T) { //nolint:funlen
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
			// A7 (DATA-02 edge): a folder nested under a subdirectory yields
			// the short base name, not the full temp path.
			"folder_base_nested",
			"[tool.poetry]\nversion = \"1.0.0\"\n",
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
			// DATA-04 on the poetry arm: description absent in
			// [tool.poetry] → README first real paragraph.
			"poetry_description_readme_fallback",
			"[tool.poetry]\nname = \"poet\"\nversion = \"0.9.0\"\n",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "README.md"),
					[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\nA poet's library.\n"),
					0o600,
				))

				return folder
			},
			"poet",
			"0.9.0",
			"A poet's library.",
		},
		{
			// D-09 presence: an EMPTY pyproject.toml still matches, with
			// folder-base Name and empty fields.
			"empty_file_presence",
			"",
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
			require.NoError(t, os.WriteFile(filepath.Join(folder, "pyproject.toml"), []byte(tt.content), 0o600))

			pd, ok := detectPython(folder)
			require.True(t, ok)
			assert.Equal(t, LanguagePython, pd.Language)
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