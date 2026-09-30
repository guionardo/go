package projectprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ignoreNames mirrors the 13 D-10 ignore names for test fixtures.
var ignoreNames = []string{
	"node_modules", "vendor", ".git", "dist", ".idea",
	".venv", "venv", "__pycache__", "target", "build",
	"bin", "obj", ".cache",
}

// TestProbe covers the Probe contract end-to-end: sentinel errors for hard
// folder-level I/O failures, LanguageUnknown with nil error for content
// outcomes, and the Folder = filepath.Clean(as-given) semantics (D-08/D-09).
func TestProbe(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns a path
		wantErr error
		want    Language
	}{
		{
			"missing_folder",
			func(t *testing.T) string { return filepath.Join(t.TempDir(), "nope") },
			ErrFolderNotFound,
			"",
		},
		{"path_is_a_file", func(t *testing.T) string {
			f := filepath.Join(t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

			return f
		}, ErrNotDirectory, ""},
		{"empty_folder", func(t *testing.T) string { return t.TempDir() }, nil, LanguageUnknown},
		{"ignored_only", func(t *testing.T) string {
			dir := t.TempDir()
			for _, name := range []string{ignoreNames[0], ignoreNames[2], ignoreNames[3]} {
				require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o700))
			}

			return dir
		}, nil, LanguageUnknown},
		{"unknown_content", func(t *testing.T) string {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("hello"), 0o600))

			return dir
		}, nil, LanguageUnknown},
		{"empty_string", func(t *testing.T) string { return "" }, ErrFolderNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder := tt.setup(t)
			data, err := Probe(folder)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, data.Language)
			assert.Equal(t, filepath.Clean(folder), data.Folder) // D-08/D-09
		})
	}
}

// TestProbe_Deterministic verifies that probing the same folder twice yields
// identical ProjectData and that Folder is filepath.Clean(as-given) — the
// clean-once canonical path is reused everywhere (D-09).
func TestProbe_Deterministic(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(base, "a"), 0o700))
	require.NoError(t, os.Mkdir(filepath.Join(base, "proj"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(base, "proj", "README"), []byte("x"), 0o600))
	folder := filepath.Join(base, "a", "..", "proj")

	first, err := Probe(folder)
	require.NoError(t, err)
	second, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, filepath.Clean(folder), first.Folder)
}

// TestIgnoreList verifies all 13 D-10 ignore names match exactly and that a
// case variant does not (D-11 exact-case matching).
func TestIgnoreList(t *testing.T) {
	t.Parallel()

	for _, name := range ignoreNames {
		assert.True(t, isIgnoredDir(name), "expected %q to be ignored", name)
	}

	assert.False(t, isIgnoredDir("Node_Modules"), "exact-case match required (D-11)")
}
