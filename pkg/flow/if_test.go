package flow_test

import (
	"testing"

	"github.com/guionardo/go/pkg/flow"
	"github.com/stretchr/testify/assert"
)

func TestIf(t *testing.T) {
	assert.Equal(t, "A", flow.If(true, "A", "B"))
	assert.Equal(t, "B", flow.If(false, "A", "B"))
}
