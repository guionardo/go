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

// TestProbe_RustEndToEnd exercises the full chain through Probe: registry →
// detectRust → readTOMLSection([package]) → merge. A folder with a Cargo.toml
// carrying a [package] section yields LanguageRust with the name/version/
// description verbatim (SC3, D-06).
func TestProbe_RustEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A Rust tool.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
	assert.Equal(t, "A Rust tool.", data.Description)
}

// TestProbe_RustWorkspaceVersion pins SC3 / D-07: version.workspace = true is
// a dotted key the subset reader classifies as unsupported, so Version is ""
// — never fabricated, never resolved from a workspace root.
func TestProbe_RustWorkspaceVersion(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\nversion.workspace = true\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_RustWorkspaceDescription pins DATA-04 for workspace-inherited
// descriptions: description.workspace = true degrades to "" so the README
// first real paragraph fallback fires (badges are skipped by readmeDescription).
func TestProbe_RustWorkspaceDescription(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\ndescription.workspace = true\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\nA Rust tool.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "", data.Version)
	assert.Equal(t, "A Rust tool.", data.Description)
}

// TestProbe_RustAuthorsArray pins P4: a one-line array value (authors = [...])
// is legal-but-unsupported and must degrade per-key without disturbing the
// section state — the following version keyval is still read.
func TestProbe_RustAuthorsArray(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\nauthors = [\"Alice\", \"Bob\"]\nversion = \"1.2.3\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
}

// TestProbe_RustVirtualManifest pins OQ-2: a virtual workspace manifest
// ([workspace] without [package]) still matches on presence (D-09) with
// folder-base Name and empty Version.
func TestProbe_RustVirtualManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[workspace]\nmembers = [\"crates/*\"]\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_RustPublishFalse pins P4 at detector level: an unquoted boolean
// value (publish = false) degrades without disturbing the section state —
// Name and Version stay intact.
func TestProbe_RustPublishFalse(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\nversion = \"1.2.3\"\npublish = false\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
}

// TestProbe_RustPresence pins the D-09 presence-match rule: a garbage
// Cargo.toml (no parseable TOML at all) still claims LanguageRust — contrast
// with the JS parse-success rule — with folder-base Name and empty fields.
func TestProbe_RustPresence(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("not a manifest\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
	assert.Equal(t, "", data.Description)
}

// TestDetectRust_MissingManifest pins the D-09 never-fail degrade at the
// detector level: a folder without Cargo.toml yields (ProjectData{}, false),
// never an error.
func TestDetectRust_MissingManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	pd, ok := detectRust(folder)
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}

// TestProbe_RustBOM pins Pitfall 1: a Cargo.toml prefixed with the literal
// UTF-8 BOM bytes (0xEF 0xBB 0xBF) parses identically — readManifest strips
// the BOM upstream, and the reader tolerates it directly too.
func TestProbe_RustBOM(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("\xEF\xBB\xBF[package]\nname = \"acme\"\nversion = \"1.0.0\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
}

// TestProbe_RustCRLF pins P5: Windows-edited Cargo.toml with \r\n line
// endings parses identically — the reader's per-line TrimSpace handles \r.
func TestProbe_RustCRLF(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\r\nname = \"acme\"\r\nversion = \"1.0.0\"\r\ndescription = \"CRLF file.\"\r\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.0.0", data.Version)
	assert.Equal(t, "CRLF file.", data.Description)
}

// TestProbe_RustNoVersion pins the Cargo ecosystem norm: name is the only
// field Cargo requires — a [package] without a version key is legal, so
// Version is "" (never fabricated).
func TestProbe_RustNoVersion(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_RustMultiLineAuthors pins P4 at probe level: a multi-line
// authors array spans lines, so the reader's global bracket-skip state holds
// until the closing bracket — the following version keyval is still read.
func TestProbe_RustMultiLineAuthors(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "Cargo.toml"),
		[]byte("[package]\nname = \"acme\"\nauthors = [\n    \"Alice\",\n    \"Bob\",\n]\nversion = \"1.2.3\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageRust, data.Language)
	assert.Equal(t, "acme", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
}

// TestDetectRust pins the detector-level edges (detect_go_test.go table
// shape with a folder-setup column): folder-base edges (nested temp subdir →
// short base), workspace-description README fallback (DATA-04 on the
// workspace arm), and empty-file presence match (D-09). A wantName of ""
// means the row expects the filepath.Base(folder) fallback.
func TestDetectRust(t *testing.T) { //nolint:funlen
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
			"[package]\nversion = \"1.0.0\"\n",
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
			// DATA-04 on the workspace arm: description.workspace = true
			// degrades to "" so the README first real paragraph fires.
			"workspace_description_readme_fallback",
			"[package]\nname = \"acme\"\ndescription.workspace = true\n",
			func(t *testing.T) string {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, "README.md"),
					[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\nA Rust library.\n"),
					0o600,
				))

				return folder
			},
			"acme",
			"",
			"A Rust library.",
		},
		{
			// D-09 presence: an EMPTY Cargo.toml still matches, with
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
			require.NoError(t, os.WriteFile(filepath.Join(folder, "Cargo.toml"), []byte(tt.content), 0o600))

			pd, ok := detectRust(folder)
			require.True(t, ok)
			assert.Equal(t, LanguageRust, pd.Language)
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
