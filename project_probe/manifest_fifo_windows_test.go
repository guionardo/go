//go:build windows

package projectprobe

import "testing"

// makeFIFO is unreachable on Windows — the fifo_blocks row skips first — and
// exists only so manifest_test.go compiles on Windows, where syscall.Mkfifo
// is not defined (WR-01 test row is Unix-only by design).
func makeFIFO(t *testing.T, _ string) {
	t.Helper()
	t.Skip("syscall.Mkfifo is Unix-only")
}
