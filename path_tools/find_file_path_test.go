package pathtools_test

import (
	"os"
	"path"
	"strings"
	"testing"

	pathtools "github.com/guionardo/go/path_tools"
	"github.com/stretchr/testify/require"
)

func TestFindFileInPath(t *testing.T) {
	t.Run("missing_path_env_should_return_error", func(t *testing.T) {
		t.Setenv("PATH", "")

		_, err := pathtools.FindFileInPath("somefile.txt")
		require.Error(t, err)
	})
	t.Run("existing_file_should_return_path", func(t *testing.T) {
		tmp1 := t.TempDir()
		tmp2 := t.TempDir()
		tmp3 := t.TempDir()
		pathEnv := strings.Join([]string{tmp1, tmp2, tmp3}, string(os.PathListSeparator))
		t.Setenv("PATH", pathEnv)

		require.NoError(t, os.WriteFile(path.Join(tmp2, "test.txt"), []byte{1}, 0600))

		filePath, err := pathtools.FindFileInPath("test.txt")
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(filePath, tmp2))
	})
	t.Run("unexistent_file_should_return_error", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("PATH", tmp)

		filepath, err := pathtools.FindFileInPath("unexistent.txt")
		require.Error(t, err)
		require.Empty(t, filepath)
	})
}
