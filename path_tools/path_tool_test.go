package pathtools

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirExists(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	t.Run("Existent", func(t *testing.T) {
		t.Parallel()
		assert.True(t, DirExists(tmp))
	})

	t.Run("Unexistent", func(t *testing.T) {
		t.Parallel()

		unexistent := path.Join(tmp, "unexistent")
		assert.False(t, DirExists(unexistent))
	})
}

func TestCreatePath(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	tryWrite := func(base, filename string) error {
		return os.WriteFile(path.Join(base, filename), []byte{}, 0600)
	}

	t.Run("Existing", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, CreatePath(tmp))
	})
	t.Run("Writable", func(t *testing.T) {
		t.Parallel()

		writable := path.Join(tmp, "writable")
		if !assert.NoError(t, CreatePath(writable)) {
			return
		}

		assert.NoError(t, tryWrite(writable, "test.txt"))
	})

	t.Run("Unwritable", func(t *testing.T) {
		t.Parallel()

		unwritable := path.Join(tmp, "unwritable")
		require.NoError(t, CreatePath(unwritable))
		require.Error(t, tryWrite(unwritable, ""))
	})
}

func TestFileExists(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	t.Run("Existent", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, os.WriteFile(path.Join(tmp, "exist"), []byte{}, 0600))
		assert.True(t, FileExists(path.Join(tmp, "exist")))
	})
	t.Run("Unexistent", func(t *testing.T) {
		t.Parallel()
		assert.False(t, FileExists(path.Join(tmp, "unexist")))
	})
}
