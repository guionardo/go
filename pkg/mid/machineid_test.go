package mid_test

import (
	"testing"

	"github.com/guionardo/go/pkg/mid"
	"github.com/stretchr/testify/assert"
)

func TestMachineID(t *testing.T) {
	got := mid.MachineID()
	assert.NotEmpty(t, got)
}
