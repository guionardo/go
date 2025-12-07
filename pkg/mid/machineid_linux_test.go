//go:build linux

package mid

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		f           func() (string, error)
		expectError bool
	}{
		{"collectHostnamectl", collectHostnamectl, false},
		{"collectDbusMachineId", collectDbusMachineId, false},
		{"collectEtcMachineId", collectEtcMachineId, false},
		{"emptyResult", func() (string, error) { return outErr("", "empty") }, true},
	}
	previous := collectFuncs

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			collectFuncs = []func() (string, error){tt.f}

			got := MachineID()
			if tt.expectError {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
			}

			t.Logf("Collect [%s]() = %s", runtime.GOOS, got)
		})
	}

	collectFuncs = previous
}
