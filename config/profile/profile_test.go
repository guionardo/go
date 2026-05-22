package profile

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_findYAMLFile(t *testing.T) {
	t.Parallel()

	t.Run("existing_profile_should_return_no_error", func(t *testing.T) {
		t.Parallel()

		filename := path.Join(t.TempDir(), "sample.yaml")
		require.NoError(t, os.WriteFile(filename, []byte{}, 0600))
		found, err := findYAMLFile(filename)
		require.NoError(t, err)
		require.FileExists(t, found)
	})

	t.Run("not_existing_profile_should_return_error", func(t *testing.T) {
		t.Parallel()

		filename := path.Join(t.TempDir(), "sample.yaml")
		_, err := findYAMLFile(filename)
		require.Error(t, err)
	})
}

func Test_getProfileFiles(t *testing.T) {
	t.Parallel()

	t.Run("two_existing_files", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		defaultScope := path.Join(tmp, "default.yml")
		require.NoError(t, os.WriteFile(defaultScope, []byte{}, 0600))

		scope := path.Join(tmp, "scope.yaml")
		require.NoError(t, os.WriteFile(scope, []byte{}, 0600))

		ds, sc, err := getProfileFiles(tmp, "default", "scope")
		require.NoError(t, err)
		require.Equal(t, defaultScope, ds)
		require.Equal(t, scope, sc)
	})

	t.Run("not_existing_default_profile", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		scope := path.Join(tmp, "scope.yaml")
		require.NoError(t, os.WriteFile(scope, []byte{}, 0600))

		_, _, err := getProfileFiles(tmp, "default", "scope")
		require.Error(t, err)
	})

	t.Run("not_existing_profile", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()

		defaultScope := path.Join(tmp, "default.yml")
		require.NoError(t, os.WriteFile(defaultScope, []byte{}, 0600))

		_, _, err := getProfileFiles(tmp, "default", "scope")
		require.Error(t, err)
	})
}

func Test_readProfileMap(t *testing.T) {
	t.Parallel()

	t.Run("valid_file_should_return_data", func(t *testing.T) {
		t.Parallel()

		profile := path.Join(t.TempDir(), "profile.yml")
		require.NoError(t, os.WriteFile(profile, []byte("name: profile"), 0600))

		m, err := readProfileMap(profile)
		require.NoError(t, err)
		require.Equal(t, map[string]any{"name": "profile"}, m)
	})

	t.Run("inexistent_file_should_return_error", func(t *testing.T) {
		t.Parallel()

		_, err := readProfileMap(path.Join(t.TempDir(), "unexistent"))
		require.Error(t, err)
	})

	t.Run("invalid_file_should_return_error", func(t *testing.T) {
		t.Parallel()

		profile := path.Join(t.TempDir(), "profile.yml")
		require.NoError(t, os.WriteFile(profile, []byte(",,,,,"), 0600))

		_, err := readProfileMap(profile)
		require.Error(t, err)
	})
}
