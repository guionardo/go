package pathtools_test

import (
	"os"
	"path"
	"testing"

	pathtools "github.com/guionardo/go/path_tools"
	"github.com/stretchr/testify/require"
)

func TestGetRootFolder(t *testing.T) {
	t.Parallel()
	t.Run("current_test_folder_should_return_valid_folder", func(t *testing.T) {
		t.Parallel()

		got, gotErr := pathtools.GetRootFolder("")
		require.NoError(t, gotErr)
		require.True(t, pathtools.DirExists(got))
	})
	t.Run("not_a_go_project", func(t *testing.T) {
		t.Parallel()
		got, gotErr := pathtools.GetRootFolder(t.TempDir())
		require.ErrorIs(t, gotErr, pathtools.ErrNotAGoProject)
		require.Empty(t, got)
	})
	t.Run("nonexistent_folder", func(t *testing.T) {
		t.Parallel()
		got, gotErr := pathtools.GetRootFolder(path.Join(t.TempDir(), "unexistent"))
		require.ErrorIs(t, gotErr, os.ErrNotExist)
		require.Empty(t, got)
	})
}
