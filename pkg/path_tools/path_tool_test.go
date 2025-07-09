package pathtools

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirExists(t *testing.T) {
	tmp := t.TempDir()
	assert.True(t, DirExists(tmp))

	unexistent := path.Join(tmp, "unexistent")
	assert.False(t, DirExists(unexistent))
}
