package sqlite

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ok := func() (string, error) { return root, nil }
	fail := func() (string, error) { return "", errors.New("user cache dir unavailable") }

	tests := []struct {
		name       string
		cfg        *Config
		dir        func() (string, error)
		wantPath   string
		wantMemory bool
		wantErr    bool
	}{
		{
			name:       "memory_option",
			cfg:        &Config{Memory: true},
			dir:        ok,
			wantPath:   ":memory:",
			wantMemory: true,
		},
		{
			name:       "memory_path_sentinel",
			cfg:        &Config{Path: ":memory:"},
			dir:        ok,
			wantPath:   ":memory:",
			wantMemory: true,
		},
		{
			name:     "explicit_path_as_given",
			cfg:      &Config{Path: filepath.Join(root, "custom.db")},
			dir:      ok,
			wantPath: filepath.Join(root, "custom.db"),
		},
		{
			name:     "name_resolves_under_user_cache_dir",
			cfg:      &Config{Name: "app"},
			dir:      ok,
			wantPath: filepath.Join(root, "app", "cache.db"),
		},
		{
			name:       "zero_value_is_memory",
			cfg:        &Config{},
			dir:        ok,
			wantPath:   ":memory:",
			wantMemory: true,
		},
		{
			name:       "memory_beats_path_and_name",
			cfg:        &Config{Memory: true, Path: filepath.Join(root, "x.db"), Name: "app"},
			dir:        ok,
			wantPath:   ":memory:",
			wantMemory: true,
		},
		{
			name:     "path_beats_name",
			cfg:      &Config{Path: filepath.Join(root, "win.db"), Name: "app"},
			dir:      ok,
			wantPath: filepath.Join(root, "win.db"),
		},
		{
			name:    "user_cache_dir_error_propagates",
			cfg:     &Config{Name: "app"},
			dir:     fail,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, memory, err := resolveLocation(tc.cfg, tc.dir)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "user cache dir")
				assert.NotErrorIs(t, err, ErrInvalidPath)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, path)
			assert.Equal(t, tc.wantMemory, memory)
		})
	}
}

func TestInvalidPathAndName(t *testing.T) {
	t.Parallel()

	t.Run("invalid_paths", func(t *testing.T) {
		t.Parallel()

		for _, p := range []string{"bad?path", "bad#path", filepath.Join("dir", "a?b", "c.db")} {
			_, _, err := resolveLocation(&Config{Path: p}, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidPath)
		}
	})

	t.Run("invalid_names", func(t *testing.T) {
		t.Parallel()

		// nil userCacheDir proves validation runs before the lookup.
		for _, n := range []string{".", "..", "a/b", "a b", `a\b`, "na?me", "na#me"} {
			_, _, err := resolveLocation(&Config{Name: n}, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidPath)
		}
	})

	t.Run("valid_names", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		dir := func() (string, error) { return root, nil }

		for _, n := range []string{"app", "my.cache_1-x", "A"} {
			path, memory, err := resolveLocation(&Config{Name: n}, dir)
			require.NoError(t, err)
			assert.False(t, memory)
			assert.Equal(t, filepath.Join(root, n, "cache.db"), path)
		}
	})

	t.Run("valid_name_allow_list", func(t *testing.T) {
		t.Parallel()

		assert.True(t, validName("app"))
		assert.True(t, validName("my.cache_1-x"))
		assert.False(t, validName(""))
		assert.False(t, validName("."))
		assert.False(t, validName(".."))
		assert.False(t, validName("a/b"))
		assert.False(t, validName("a b"))
		assert.False(t, validName("a?b"))
		assert.False(t, validName("a#b"))
		assert.False(t, validName("acentuação"))
	})
}

func TestDSNBuildDSN(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "cache.db")

	assert.Equal(t,
		filePath+"?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate",
		buildDSN(filePath, false))

	assert.Equal(t,
		":memory:?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate",
		buildDSN(":memory:", true))

	// Memory mode must not carry the journal_mode key (SQLite reports "memory").
	assert.NotContains(t, buildDSN(":memory:", true), "journal_mode")
}
