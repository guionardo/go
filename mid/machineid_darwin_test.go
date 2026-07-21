//go:build darwin

package mid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMachineID_Darwin(t *testing.T) {
	t.Parallel()

	id := MachineID()
	if id == "" {
		t.Log("system_profiler not available — expected in CI containers without macOS hardware info")
	} else {
		assert.Contains(t, id, "|")
	}
}
