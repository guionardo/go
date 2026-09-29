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
