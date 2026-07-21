//go:build windows

package mid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMachineID_Windows(t *testing.T) {
	t.Parallel()

	id := MachineID()
	if id == "" {
		t.Log("reg query SQMClient not available — expected in CI containers")
	} else {
		assert.Len(t, id, 36) // UUID format
	}
}
