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