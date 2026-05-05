package pathtools

import (
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRootDirectory(t *testing.T) {
	t.Parallel()

	t.Run("temp_directory_should_return_false", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		assert.Falsef(t, IsRootDirectory(directory),
			"expected that directory %s is not the same of the base %s", directory, path.Base(directory))
	})
	t.Run("root_directory_should_return_true", func(t *testing.T) {
		t.Parallel()

		var directory string
		if runtime.GOOS == "windows" {
			directory = "C:\\"
		} else {
			directory = "/"
		}

		assert.True(t, IsRootDirectory(directory))
	})
}

func Test_windowsPathBaseFunc(t *testing.T) {
	t.Parallel()

	t.Run("normal multilevel path should return base path", func(t *testing.T) {
		t.Parallel()

		base := windowsPathBaseFunc("C:\\windows\\system32")
		require.Equal(t, "C:\\windows", base)
	})

	t.Run("base path should return same path", func(t *testing.T) {
		t.Parallel()

		base := windowsPathBaseFunc("C:\\")
		require.Equal(t, "C:\\", base)
	})
}
