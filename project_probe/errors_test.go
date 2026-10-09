package projectprobe

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// opStat is the synthetic PathError.Op used across the mapping table rows.
const opStat = "stat"

// TestMapFolderError verifies the D-04 errno mapping: ENOENT →
// ErrFolderNotFound, ENOTDIR → ErrNotDirectory, EACCES or EPERM →
// ErrPermissionDenied (amended 2026-09-28), everything else → wrapped raw
// error. Every row also asserts the D-02 wrap shape "probe <folder>:".
func TestMapFolderError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"enoent", &os.PathError{Op: opStat, Path: "x", Err: syscall.ENOENT}, ErrFolderNotFound},
		{"enotdir", &os.PathError{Op: opStat, Path: "x", Err: syscall.ENOTDIR}, ErrNotDirectory},
		{"eacces", &os.PathError{Op: opStat, Path: "x", Err: syscall.EACCES}, ErrPermissionDenied},
		{"eperm", &os.PathError{Op: opStat, Path: "x", Err: syscall.EPERM}, ErrPermissionDenied},
		{"raw", errors.New("boom"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := mapFolderError("x", tt.err)
			require.Contains(t, err.Error(), "probe x:")

			if tt.wantErr == nil {
				require.NotErrorIs(t, err, ErrFolderNotFound)
				require.NotErrorIs(t, err, ErrNotDirectory)
				require.NotErrorIs(t, err, ErrPermissionDenied)

				return
			}

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestProbePermissionDenied exercises the real filesystem permission path.
// OS-gated per RESEARCH Pitfall 3: chmod-based permission tests fail on
// Windows (ACLs) and under root (modes ignored).
func TestProbePermissionDenied(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission semantics differ on windows or as root")
	}

	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o600) })

	_, err := Probe(dir)
	require.ErrorIs(t, err, ErrPermissionDenied)
}
