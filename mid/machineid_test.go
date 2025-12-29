package mid_test

import (
	"runtime"
	"testing"

	"github.com/guionardo/go/mid"
	"github.com/stretchr/testify/assert"
)

func TestMachineID(t *testing.T) {
	t.Parallel()

	got := mid.MachineID()
	t.Logf("MachineId [%s] () = %s", runtime.GOOS, got)
	assert.NotEmpty(t, got)
}
