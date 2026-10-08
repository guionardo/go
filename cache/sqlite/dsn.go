package sqlite

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DSN pragma suffixes. Every per-connection pragma rides in the DSN so the
// driver applies it to every physical connection; post-open Exec pragmas are
// never used. Memory mode must not carry a journal_mode key — SQLite reports
// "memory" there and WAL is meaningless for an in-memory database.
const (
	dsnSuffixFile   = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"
	dsnSuffixMemory = "?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate"
)

// resolveLocation resolves the configured location to a database path using
// the documented precedence memory > path > name > zero value (memory).
//
// The userCacheDir seam is injected for deterministic tests; in production it
// is os.UserCacheDir. A lookup error is propagated — there is no fallback to
// another directory.
func resolveLocation(cfg *Config, userCacheDir func() (string, error)) (path string, memory bool, err error) {
	switch {
	case cfg.Memory || cfg.Path == ":memory:":
		return ":memory:", true, nil
	case cfg.Path != "":
		if strings.ContainsAny(cfg.Path, "?#") {
			return "", false, fmt.Errorf("%w: path must not contain '?' or '#'", ErrInvalidPath)
		}
		return cfg.Path, false, nil
	case cfg.Name != "":
		if !validName(cfg.Name) {
			return "", false, fmt.Errorf("%w: invalid cache name %q", ErrInvalidPath, cfg.Name)
		}
		root, err := userCacheDir()
		if err != nil {
			return "", false, fmt.Errorf("resolve user cache dir: %w", err)
		}
		return filepath.Join(root, cfg.Name, "cache.db"), false, nil
	default:
		return ":memory:", true, nil
	}
}

// validName reports whether the cache name is safe to join under the cache
// root: non-empty, not "." or "..", and only [A-Za-z0-9._-] characters (no
// separators, no spaces).
func validName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// buildDSN appends the compile-time-constant pragma suffix to the validated
// path. The only variable component is the already-validated path itself —
// pragma values are never derived from caller input.
func buildDSN(path string, memory bool) string {
	if memory {
		return path + dsnSuffixMemory
	}
	return path + dsnSuffixFile
}
