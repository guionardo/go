//go:build !windows

package projectprobe

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// makeFIFO creates a named pipe at path. Unix-only: the fifo_blocks test row
// skips on Windows (runtime.GOOS guard) and this build tag keeps the
// syscall.Mkfifo reference out of Windows builds, where the symbol does not
// exist.
func makeFIFO(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, syscall.Mkfifo(path, 0o600))
}