package pathtools

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirExists(t *testing.T) {
	tmp := t.TempDir()
	t.Run("Existent", func(t *testing.T) {
		assert.True(t, DirExists(tmp))
	})

	t.Run("Unexistent", func(t *testing.T) {
		unexistent := path.Join(tmp, "unexistent")
		assert.False(t, DirExists(unexistent))
	})
}

func TestCreatePath(t *testing.T) {
	tmp := t.TempDir()
	tryWrite := func(base, filename string) error {
		return os.WriteFile(path.Join(base, filename), []byte{}, 0644)
	}
	t.Run("Writable", func(t *testing.T) {
		writable := path.Join(tmp, "writable")
		if !assert.NoError(t, CreatePath(writable)) {
			return
		}
		assert.NoError(t, tryWrite(writable, "test.txt"))
	})
	t.Run("Unwritable",func(t *testing.T){
		unwritable=
	})

}
