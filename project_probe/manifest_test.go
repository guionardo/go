package projectprobe

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestName is the file name used by every test row. Real call sites pass
// compile-time constants ("go.mod", "package.json", ...), so a literal mirrors
// production usage (DETC-01).
const manifestName = "go.mod"

// TestReadManifest covers the readManifest boundary contract (ROBT-02): the
// 1 MB cap accepts exactly 1 MB and rejects 1 MB + 1 byte via the limit+1
// truncation probe, the UTF-8 BOM (EF BB BF) is stripped, and every failure
// mode (missing file, directory at path, empty name) yields ok=false — never
// an error, never a panic (D-03).
func TestReadManifest(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		setup  func(t *testing.T) (string, []byte) // returns folder + expected content (nil when !wantOK)
		wantOK bool
	}{
		{
			"missing_file",
			manifestName,
			func(t *testing.T) (string, []byte) { return t.TempDir(), nil },
			false,
		},
		{
			"exactly_1mb",
			manifestName,
			func(t *testing.T) (string, []byte) {
				content := bytes.Repeat([]byte("a"), maxManifestSize)
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(folder, manifestName), content, 0o600))

				return folder, content
			},
			true,
		},
		{
			"over_1mb",
			manifestName,
			func(t *testing.T) (string, []byte) {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, manifestName),
					bytes.Repeat([]byte("a"), maxManifestSize+1),
					0o600,
				))

				return folder, nil
			},
			false,
		},
		{
			"bom_stripped",
			manifestName,
			func(t *testing.T) (string, []byte) {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(folder, manifestName),
					[]byte("\xEF\xBB\xBFpackage main\n"),
					0o600,
				))

				return folder, []byte("package main\n")
			},
			true,
		},
		{
			"no_bom",
			manifestName,
			func(t *testing.T) (string, []byte) {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(folder, manifestName), []byte("package main\n"), 0o600))

				return folder, []byte("package main\n")
			},
			true,
		},
		{
			"bom_only",
			manifestName,
			func(t *testing.T) (string, []byte) {
				folder := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(folder, manifestName), []byte("\xEF\xBB\xBF"), 0o600))

				return folder, []byte{}
			},
			true,
		},
		{
			"path_is_directory",
			manifestName,
			func(t *testing.T) (string, []byte) {
				folder := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(folder, manifestName), 0o700))

				return folder, nil
			},
			false,
		},
		{
			"empty_name",
			"",
			func(t *testing.T) (string, []byte) { return t.TempDir(), nil },
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder, want := tt.setup(t)

			content, ok := readManifest(folder, tt.file)
			assert.Equal(t, tt.wantOK, ok)

			if tt.wantOK {
				assert.Equal(t, want, content)
			}
		})
	}
}
