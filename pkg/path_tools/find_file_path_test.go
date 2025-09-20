package pathtools_test

import (
	"os"
	"path"
	"strings"
	"testing"

	pathtools "github.com/guionardo/go/pkg/path_tools"
	"github.com/stretchr/testify/assert"
)

func TestFindFileInPath(t *testing.T) {
	oldPath := os.Getenv("PATH")
	t.Run("missing_path_env_should_return_error", func(t *testing.T) {
		_ = os.Setenv("PATH", "")
		_, err := pathtools.FindFileInPath("somefile.txt")
		assert.Error(t, err)
	})
	t.Run("existing_file_should_return_path", func(t *testing.T) {
		tmp1 := t.TempDir()
		tmp2 := t.TempDir()
		tmp3 := t.TempDir()
		pathEnv := strings.Join([]string{tmp1, tmp2, tmp3}, string(os.PathListSeparator))
		_ = os.Setenv("PATH", pathEnv)
		if !assert.NoError(t, os.WriteFile(path.Join(tmp2, "test.txt"), []byte{1}, 0644)) {
			return
		}
		filePath, err := pathtools.FindFileInPath("test.txt")
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(filePath, tmp2))
	})
	t.Run("unexistent_file_should_return_error", func(t *testing.T) {
		tmp := t.TempDir()
		_ = os.Setenv("PATH", tmp)
		filepath, err := pathtools.FindFileInPath("unexistent.txt")
		assert.Error(t, err)
		assert.Empty(t, filepath)
	})
	_ = os.Setenv("PATH", oldPath)
}
