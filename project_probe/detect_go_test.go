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

// TestProbe_GoEndToEnd exercises the full chain through Probe: registry →
// detectGo → parseGoMod → readmeDescription → merge. A folder with go.mod and
// README.md yields LanguageGo with Name from the module line, Version from the
// go directive (raw, toolchain floor — D-02), Description from the README
// first real paragraph (D-03/DATA-04).
func TestProbe_GoEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("module example.com/acme\n\ngo 1.26.4\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\n# Acme\n\nA CLI for acme.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, "example.com/acme", data.Name)
	assert.Equal(t, "1.26.4", data.Version)
	assert.Equal(t, "A CLI for acme.", data.Description)
}

// TestProbe_GoBOM verifies SC1: a BOM-prefixed go.mod parses identically —
// the readManifest BOM strip makes the detector more lenient than the go tool.
func TestProbe_GoBOM(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("\xEF\xBB\xBFmodule example.com/acme\n\ngo 1.26.4\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, "example.com/acme", data.Name)
	assert.Equal(t, "1.26.4", data.Version)
}

// TestProbe_GoPresence pins the D-disc-1 presence match rule: a go.mod with
// zero parseable directives still matches — Language=Go, Name falls back to
// the folder base (DATA-02), Version stays "" (never fabricated, D-02).
func TestProbe_GoPresence(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("not a go module\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "", data.Version)
	assert.Equal(t, "", data.Description)
}

// TestProbe_GoNameFallback pins the DATA-02 name chain for Go: a go.mod with
// only a go directive (no module line) still matches and Name falls back to
// the folder base while Version reports the raw directive.
func TestProbe_GoNameFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("go 1.21\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
	assert.Equal(t, "1.21", data.Version)
}