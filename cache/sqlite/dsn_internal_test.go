package sqlite

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// locationCase is one row of the resolveLocation precedence matrix.
type locationCase struct {
	name       string
	cfg        *Config
	dir        func() (string, error)
	wantPath   string
	wantMemory bool
	wantErr    bool
}

// testAppName is a valid cache name reused across the location tests.
const testAppName = "app"

func locationCases(root string) []locationCase {
	ok := func() (string, error) { return root, nil }
	fail := func() (string, error) { return "", errors.New("user cache dir unavailable") }

	return []locationCase{
		{
			name:       "memory_option",
			cfg:        &Config{Memory: true},
			dir:        ok,
			wantPath:   memoryPath,
			wantMemory: true,
		},
		{
			name:       "memory_path_sentinel",
			cfg:        &Config{Path: memoryPath},
			dir:        ok,
			wantPath:   memoryPath,
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
			cfg:      &Config{Name: testAppName},
			dir:      ok,
			wantPath: filepath.Join(root, testAppName, "cache.db"),
		},
		{
			name:       "zero_value_is_memory",
			cfg:        &Config{},
			dir:        ok,
			wantPath:   memoryPath,
			wantMemory: true,
		},
		{
			name:       "memory_beats_path_and_name",
			cfg:        &Config{Memory: true, Path: filepath.Join(root, "x.db"), Name: testAppName},
			dir:        ok,
			wantPath:   memoryPath,
			wantMemory: true,
		},
		{
			name:     "path_beats_name",
			cfg:      &Config{Path: filepath.Join(root, "win.db"), Name: testAppName},
			dir:      ok,
			wantPath: filepath.Join(root, "win.db"),
		},
		{
			name:    "user_cache_dir_error_propagates",
			cfg:     &Config{Name: testAppName},
			dir:     fail,
			wantErr: true,
		},
	}
}

func TestLocation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	for _, tc := range locationCases(root) {
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

		for _, n := range []string{testAppName, "my.cache_1-x", "A"} {
			path, memory, err := resolveLocation(&Config{Name: n}, dir)
			require.NoError(t, err)
			assert.False(t, memory)
			assert.Equal(t, filepath.Join(root, n, "cache.db"), path)
		}
	})

	t.Run("valid_name_allow_list", func(t *testing.T) {
		t.Parallel()

		assert.True(t, validName(testAppName))
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
	fileSuffix := "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"

	assert.Equal(t, filePath+fileSuffix, buildDSN(filePath, false, 0))

	// pages > 0 appends the wal_autocheckpoint pragma (A6: the only variable
	// DSN component is strconv.Itoa of a caller int).
	assert.Equal(t, filePath+fileSuffix+"&_pragma=wal_autocheckpoint(200)", buildDSN(filePath, false, 200))

	// pages <= 0 appends nothing and keeps the driver default.
	assert.Equal(t, filePath+fileSuffix, buildDSN(filePath, false, -5))

	assert.Equal(t,
		memoryPath+"?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate",
		buildDSN(memoryPath, true, 0))

	// Memory mode ignores the autocheckpoint option (no WAL there).
	assert.Equal(t,
		memoryPath+"?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate",
		buildDSN(memoryPath, true, 200))

	// Memory mode must not carry the journal_mode key (SQLite reports "memory").
	assert.NotContains(t, buildDSN(memoryPath, true, 0), "journal_mode")
}
