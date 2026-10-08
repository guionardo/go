package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRoundTrip(t *testing.T) {
	c := sqlite.New[string, string](sqlite.WithMemory())
	t.Cleanup(func() { _ = c.Close() })

	err := c.Set(t.Context(), "k", "v")
	require.NoError(t, err)

	got, err := c.Get(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	err = c.Delete(t.Context(), "k")
	require.NoError(t, err)

	_, err = c.Get(t.Context(), "k")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrMiss)

	require.NoError(t, c.Close())

	_, err = c.Get(t.Context(), "k")
	require.Error(t, err)
	assert.ErrorIs(t, err, sqlite.ErrClosed)
}

func TestBatchPlaceholderSmoke(t *testing.T) {
	c := sqlite.New[string, string](sqlite.WithMemory())
	t.Cleanup(func() { _ = c.Close() })

	err := c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"})
	require.NoError(t, err)

	got := c.MGet(t.Context(), "a", "b", "missing")
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)

	err = c.MDel(t.Context(), "a")
	require.NoError(t, err)

	got = c.MGet(t.Context(), "a", "b")
	assert.Equal(t, map[string]string{"b": "2"}, got)
}

func TestFileCRUD(t *testing.T) {
	c := sqlite.New[string, string](sqlite.WithPath(filepath.Join(t.TempDir(), "cache.db")))
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "file-key", "file-value"))

	got, err := c.Get(t.Context(), "file-key")
	require.NoError(t, err)
	assert.Equal(t, "file-value", got)

	require.NoError(t, c.Delete(t.Context(), "file-key"))

	_, err = c.Get(t.Context(), "file-key")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrMiss)
}

func TestFilePathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache dir with spaces")
	c := sqlite.New[string, string](sqlite.WithPath(filepath.Join(dir, "cache.db")))
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "k", "v"))

	got, err := c.Get(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}
