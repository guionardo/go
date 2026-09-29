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

// TestDetectGo pins every verified go.mod grammar fact (RESEARCH Pattern 2 +
// Pitfall 6, live-verified): quoted/backtick module values, trailing
// comments, raw go-directive forms, toolchain/gopher/modulex isolation,
// first-occurrence-wins duplicates, and the DATA-02 folder-base fallback.
// A wantName of "" means the row expects the filepath.Base(folder) fallback.
func TestDetectGo(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name     string
		mod      string
		folder   func(t *testing.T) string // nil → t.TempDir()
		wantName string                   // "" → filepath.Base(folder)
		wantVer  string
	}{
		{
			// D-01: the double-quoted module form is tool-legal and the
			// outer quotes are stripped.
			"quoted_module",
			"module \"example.com/dq\"\n",
			nil,
			"example.com/dq",
			"",
		},
		{
			// A4 leniency: the go tool rejects backticks, we accept them.
			"backtick_module",
			"module `example.com/bt`\n",
			nil,
			"example.com/bt",
			"",
		},
		{
			// A2: trailing comments are stripped before tokenizing; "//"
			// cannot occur inside a valid module path.
			"trailing_comment",
			"module example.com/qux // my comment\n",
			nil,
			"example.com/qux",
			"",
		},
		{
			// D-02: go directive forms reported verbatim, never normalized.
			"go_1_21",
			"go 1.21\n",
			nil,
			"",
			"1.21",
		},
		{
			"go_1_21rc1",
			"module example.com/rc\ngo 1.21rc1\n",
			nil,
			"example.com/rc",
			"1.21rc1",
		},
		{
			"go_1_21_4",
			"module example.com/p\ngo 1.21.4\n",
			nil,
			"example.com/p",
			"1.21.4",
		},
		{
			// Pitfall 6: toolchain is not the go directive — Version stays
			// "" when no go directive exists.
			"toolchain_isolation",
			"toolchain go1.26.4\n",
			nil,
			"",
			"",
		},
		{
			// Pitfall 6: directive-like tokens ("gopher", "modulex") never
			// match; Name falls back to the folder base (DATA-02).
			"non_directive_lines",
			"gopher 1.2\nmodulex y\n",
			nil,
			"",
			"",
		},
		{
			// DATA-02: module absent → folder base.
			"module_absent",
			"go 1.21\n",
			nil,
			"",
			"1.21",
		},
		{
			// D-02: go directive absent → Version "".
			"go_absent",
			"module example.com/mo\n",
			nil,
			"example.com/mo",
			"",
		},
		{
			// Pattern 2: first occurrence wins per directive.
			"duplicate_modules_first_wins",
			"module example.com/first\nmodule example.com/second\n",
			nil,
			"example.com/first",
			"",
		},
		{
			// A7 (DATA-02 edge): a folder nested under a subdirectory yields
			// the short base name, not the full temp path.
			"folder_base_nested",
			"go 1.21\n",
			func(t *testing.T) string {
				sub := filepath.Join(t.TempDir(), "sub")
				require.NoError(t, os.Mkdir(sub, 0o700))

				return sub
			},
			"",
			"1.21",
		},
		{
			// Real-world composite: comments, module + go + toolchain +
			// require block — only module/go populate the fields.
			"combined_real_world",
			"// Copyright 2026 Example\nmodule example.com/full // trailing\n\ngo 1.23.0\n\n" +
				"toolchain go1.26.4\n\nrequire (\n\tgithub.com/x/y v1.0.0\n)\n",
			nil,
			"example.com/full",
			"1.23.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			if tt.folder != nil {
				folder = tt.folder(t)
			}
			require.NoError(t, os.WriteFile(filepath.Join(folder, "go.mod"), []byte(tt.mod), 0o600))

			pd, ok := detectGo(folder)
			require.True(t, ok)
			assert.Equal(t, LanguageGo, pd.Language)
			wantName := tt.wantName
			if wantName == "" {
				wantName = filepath.Base(folder)
			}
			assert.Equal(t, wantName, pd.Name)
			assert.Equal(t, tt.wantVer, pd.Version)

			// A7: the folder-base fallback is stdlib filepath.Base on the
			// cleaned path — pin the "."-like edge here too.
			if tt.name == "folder_base_nested" {
				assert.Equal(t, ".", filepath.Base(filepath.Clean(".")))
			}
		})
	}
}