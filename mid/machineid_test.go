package mid_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/guionardo/go/mid"
	"github.com/stretchr/testify/assert"
)

func TestMachineID(t *testing.T) {
	t.Parallel()

	got := mid.MachineID()
	t.Logf("MachineID [%s] = %q", runtime.GOOS, got)

	if runtime.GOOS == "linux" {
		if got == "" {
			t.Log("Linux without machine-id files/command — expected in containers")
		}

		return
	}

	assert.NotEmpty(t, got, "MachineID should be available on %s", runtime.GOOS)
}

func TestMachineID_AlwaysString(t *testing.T) {
	t.Parallel()

	got := mid.MachineID()
	assert.IsType(t, "", got, "MachineID must always return a string")
}

func ExampleMachineID() {
	id := mid.MachineID()
	fmt.Printf("Machine ID: %s\n", id)
}
