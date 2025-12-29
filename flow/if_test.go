package flow_test

import (
	"testing"

	"github.com/guionardo/go/flow"
	"github.com/stretchr/testify/assert"
)

func TestIf(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "A", flow.If(true, "A", "B"))
	assert.Equal(t, "B", flow.If(false, "A", "B"))
}
