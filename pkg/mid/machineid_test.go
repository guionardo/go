package mid_test

import (
	"runtime"
	"testing"

	"github.com/guionardo/go/pkg/mid"
	"github.com/stretchr/testify/assert"
)

func TestMachineID(t *testing.T) {
	got := mid.MachineID()
	t.Logf("MachineId [%s] () = %s", runtime.GOOS, got)
	assert.NotEmpty(t, got)
}
