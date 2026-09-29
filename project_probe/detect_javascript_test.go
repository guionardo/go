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

// TestProbe_JSEndToEnd exercises the full chain through Probe: registry →
// detectJS → readJSONManifest → merge. A folder with a well-formed
// package.json yields LanguageJavaScript with Name/Version/Description from
// the manifest via encoding/json (DETC-03). The Language is ALWAYS
// LanguageJavaScript for a package.json — even a TS-shaped manifest — the
// distinct typescript value is v2 backlog (D-05, REFN-01).
func TestProbe_JSEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"acme-widget","version":"2.1.0","description":"Widget library"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "acme-widget", data.Name)
	assert.Equal(t, "2.1.0", data.Version)
	assert.Equal(t, "Widget library", data.Description)
}

// TestProbe_JSPrivateNoVersion pins D-06: "private": true without a version
// yields Version "" — the zero value of the decode struct, never fabricated.
// There is deliberately no Private field in the decoder (a field would be
// dead code); this test row is the contract (RESEARCH anti-pattern).
func TestProbe_JSPrivateNoVersion(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"x","private":true}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "x", data.Name)
	assert.Equal(t, "", data.Version)
}

// TestProbe_JSDescriptionFallback pins the DATA-04 chain for JS: a manifest
// without a description falls back to the README first real paragraph.
func TestProbe_JSDescriptionFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"x"}`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("# Acme\n\nA widget library.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "x", data.Name)
	assert.Equal(t, "A widget library.", data.Description)
}

// TestProbe_JSDescriptionNull pins the verified encoding/json behavior:
// "description": null decodes to "" and the README fallback triggers.
func TestProbe_JSDescriptionNull(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"x","description":null}`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("A widget library.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "x", data.Name)
	assert.Equal(t, "A widget library.", data.Description)
}

// TestProbe_JSBOM pins Pitfall 1: a BOM-prefixed package.json still parses —
// the readManifest BOM strip is load-bearing for encoding/json, which rejects
// a leading BOM ("invalid character '\ufeff' looking for beginning of value").
func TestProbe_JSBOM(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte("\xEF\xBB\xBF{\"name\":\"acme-widget\",\"version\":\"2.1.0\"}"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "acme-widget", data.Name)
	assert.Equal(t, "2.1.0", data.Version)
}

// TestProbe_JSScopedName pins the DATA-02 byte-equality semantics for scoped
// npm names: "@scope/pkg" is preserved verbatim, never split or unescaped.
func TestProbe_JSScopedName(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"@scope/pkg"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "@scope/pkg", data.Name)
}

// TestProbe_JSMalformed pins D-04: a malformed package.json alone in the
// folder yields LanguageUnknown — the detector returns false and the cascade
// continues (no other manifest exists, so nothing matches).
func TestProbe_JSMalformed(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte("{invalid"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestProbe_JSDuplicateKeys pins the verified encoding/json semantics:
// duplicate keys are legal and LAST wins — no error, no first-wins
// ambiguity (T-11-10 pinned as a test row).
func TestProbe_JSDuplicateKeys(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"a","name":"b"}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "b", data.Name)
}

// TestProbe_JSUnknownFields pins the verified encoding/json semantics:
// unknown fields (license, scripts, ...) are silently ignored — the decode
// succeeds and the known fields populate (T-11-10).
func TestProbe_JSUnknownFields(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":"x","license":"MIT","scripts":{"build":"make"}}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJavaScript, data.Language)
	assert.Equal(t, "x", data.Name)
}

// TestProbe_JSTypeMismatch pins the verified encoding/json semantics: a type
// mismatch ("name": 123) is a decode error → detector returns false →
// LanguageUnknown (D-04 cascade continues).
func TestProbe_JSTypeMismatch(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"name":123}`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}