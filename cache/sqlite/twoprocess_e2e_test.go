//go:build e2e

package sqlite_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite" // registers the "sqlite" driver for the raw test-only handles
)

// Two-process E2E (Phase 16, D-09/D-10): the parent spawns two child processes
// through the TestMain re-exec pattern; each child drives the SHIPPED provider
// (the public sqlite.New + WithPath API) against one shared, fresh cache file
// — 100 point Set/Get pairs, a point Delete with an absence check on the
// tenth-iteration cadence, a 20-key MSet + MGet every 10th iteration, and a
// chunked MDel every 15th — and exits 0. The provider's own open() owns the
// bounded busy-only retry (Plan 16-02, D-06), so the child path contains no
// retry logic at all: the production path is what the E2E exercises.
const (
	helperEnv = "SQLITE_E2E_HELPER"
	dbEnv     = "SQLITE_E2E_DB"
	roleEnv   = "SQLITE_E2E_ROLE"

	// e2eDSN mirrors the dsnSuffixFile literal in dsn.go. It belongs only to
	// the raw test-only handles — the parent's verification handle and (Plan
	// 16-04 Task 2) the crash child — never to the provider-driven children.
	e2eDSN = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"

	// iterations is the per-child point-operation count.
	iterations = 100

	// batchSize is the number of keys in each periodic MSet.
	batchSize = 20

	// perRoleRows is the expected per-role row count after the
	// deletion-bearing workload: 100 point keys − 14 distinct deletions
	// (MDel at i%15==0 targets 0,7,15,22,30,37,45; point Delete at i%10==0
	// with i>=10 targets 0,10,20,30,40,50,60,70,80; 0 and 30 are deleted by
	// both paths) + 10×20 batch keys + 1 final marker.
	perRoleRows = 287

	// childTimeout bounds every parent-child interaction: no child wait is
	// unbounded (T-16-10).
	childTimeout = 60 * time.Second

	// e2eCreateTableSQL / e2eCreateIndexSQL / e2eUpsertSQL mirror schema.go's
	// constants for the raw crash-child handle (same test-only mirror note as
	// e2eDSN). The DDL bootstraps the shared fresh file so the crash child
	// (Plan 16-04 Task 2) can insert inside an open write transaction.
	e2eCreateTableSQL = `
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key  TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    expires_at INTEGER
) WITHOUT ROWID;`

	e2eCreateIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;`

	e2eUpsertSQL = `INSERT INTO cache_entries (cache_key, value, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`
)

// TestMain intercepts the helper child before the test runner starts: when the
// parent re-execs this binary with SQLITE_E2E_HELPER=1 the process acts as a
// cache client only and never enters m.Run().
func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) == "1" {
		os.Exit(helperMain())
	}

	os.Exit(m.Run())
}

// helperMain is the child entry point. It drives the shipped provider (the
// public API only — the DI-15-01 bounded open retry lives inside the
// provider's open()) against the shared database, runs the mixed workload,
// and finishes with the role's final marker. It returns the process exit code.
func helperMain() int {
	role := os.Getenv(roleEnv)

	c := sqlite.New[string, string](sqlite.WithPath(os.Getenv(dbEnv)))
	defer func() { _ = c.Close() }()

	ctx := context.Background()

	if err := runContentionWorkload(ctx, c, role); err != nil {
		fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: %v\n", role, err)

		return 1
	}

	return 0
}

// runContentionWorkload executes the D-10A mixed workload through the shipped
// provider: per-iteration Set + read-back, a periodic 20-key MSet + MGet,
// point Delete with an absence check from the 10th iteration on, and a
// chunked MDel every 15th iteration. Any failure is returned wrapped for the
// child to print and exit 1.
func runContentionWorkload(ctx context.Context, c cache.BatchCache[string, string], role string) error {
	for i := range iterations {
		key := pointKey(role, i)
		value := fmt.Sprintf("v-%d", i)

		if err := c.Set(ctx, key, value); err != nil {
			return fmt.Errorf("set %d: %w", i, err)
		}

		got, err := c.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("get %d: %w", i, err)
		}

		if got != value {
			return fmt.Errorf("get %d: got %q want %q", i, got, value)
		}

		if i%10 == 0 {
			if err := batchRoundTrip(ctx, c, role, i); err != nil {
				return fmt.Errorf("batch %d: %w", i, err)
			}
		}

		if i%10 == 0 && i >= 10 {
			deleted := pointKey(role, i-10)

			if err := c.Delete(ctx, deleted); err != nil {
				return fmt.Errorf("delete %d: %w", i, err)
			}

			if _, err := c.Get(ctx, deleted); !errors.Is(err, cache.ErrMiss) {
				return fmt.Errorf("delete %d: %q still present (err=%v)", i, deleted, err)
			}
		}

		if i%15 == 0 {
			if err := c.MDel(ctx, pointKey(role, i/2)); err != nil {
				return fmt.Errorf("mdel %d: %w", i, err)
			}
		}
	}

	if err := c.Set(ctx, role+":final", "done"); err != nil {
		return fmt.Errorf("final marker: %w", err)
	}

	return nil
}

// batchRoundTrip writes one 20-key MSet and reads one of its keys back through
// MGet.
func batchRoundTrip(ctx context.Context, c cache.BatchCache[string, string], role string, iteration int) error {
	items := make(map[string]string, batchSize)

	for j := range batchSize {
		items[batchKey(role, iteration, j)] = "b"
	}

	if err := c.MSet(ctx, items); err != nil {
		return fmt.Errorf("mset: %w", err)
	}

	probe := batchKey(role, iteration, 0)

	found := c.MGet(ctx, probe)
	if got, ok := found[probe]; !ok || got != "b" {
		return fmt.Errorf("mget: got %q ok=%v want \"b\"", got, ok)
	}

	return nil
}

// pointKey is the per-role point-operation key (role:%04d).
func pointKey(role string, index int) string {
	return fmt.Sprintf("%s:%04d", role, index)
}

// batchKey is the per-role batch key written by the periodic MSet.
func batchKey(role string, iteration, item int) string {
	return fmt.Sprintf("%s:batch:%04d:%02d", role, iteration, item)
}

// e2eBootstrap creates the fixed schema mirroring the provider's bootstrap():
// one BEGIN IMMEDIATE transaction (via _txlock=immediate) around the
// idempotent CREATE ... IF NOT EXISTS statements. It is consumed by the raw
// crash child (Plan 16-04 Task 2).
func e2eBootstrap(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	if _, err := tx.ExecContext(ctx, e2eCreateTableSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, e2eCreateIndexSQL); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return tx.Commit()
}

// startChild launches the re-executed test binary as a helper child driving
// the shared database with the given role. Child stdout/stderr are captured
// into buffers for failure diagnostics; the caller registers Kill+Wait
// cleanup for the returned process handle.
func startChild(t *testing.T, dbPath, role string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		helperEnv+"=1",
		dbEnv+"="+dbPath,
		roleEnv+"="+role,
	)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start())

	return cmd, &stderr
}

// waitChild waits for a child up to timeout. On expiry the child is killed,
// the wait is drained, and an error is returned — no child interaction is
// unbounded (T-16-10).
func waitChild(cmd *exec.Cmd, timeout time.Duration) error {
	done := make(chan error, 1)

	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done

		return fmt.Errorf("child timed out after %s", timeout)
	}
}

// cleanupChildren kills and waits any child that is still running, so no
// process, handle, or zombie survives the test.
func cleanupChildren(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}
}

// assertJournalWAL requires the reopened database to be in WAL mode.
func assertJournalWAL(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var mode string

	require.NoError(t, db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode))
	assert.Equal(t, "wal", mode)
}

// assertIntegrityOK requires SQLite's own integrity check to pass.
func assertIntegrityOK(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var integrity string

	require.NoError(t, db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity))
	assert.Equal(t, "ok", integrity)
}

// assertKeyAbsent requires the key to have no row.
func assertKeyAbsent(t *testing.T, ctx context.Context, db *sql.DB, key string) {
	t.Helper()

	var count int

	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM cache_entries WHERE cache_key = ?", key).Scan(&count))
	assert.Zero(t, count, "deleted key %q must be absent", key)
}

// TestTwoProcessContention is the parent: two OS processes drive the shipped
// provider against one fresh database file. Both children must exit 0, and
// the reopened file must be uncorrupted (integrity_check, WAL), carry each
// role's final marker, keep the deleted keys absent, and hold exactly
// perRoleRows rows per role (D-10A).
func TestTwoProcessContention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shared.db")

	roles := []string{"alpha", "beta"}

	cmds := make([]*exec.Cmd, 0, len(roles))
	stderrs := make([]*bytes.Buffer, 0, len(roles))

	for _, role := range roles {
		cmd, stderr := startChild(t, dbPath, role)
		cmds = append(cmds, cmd)
		stderrs = append(stderrs, stderr)
	}

	t.Cleanup(func() { cleanupChildren(cmds) })

	for i, cmd := range cmds {
		err := waitChild(cmd, childTimeout)
		require.NoErrorf(t, err, "child %q exited non-zero\nstderr:\n%s", roles[i], stderrs[i].String())
	}

	db, err := sql.Open("sqlite", dbPath+e2eDSN)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := t.Context()

	assertJournalWAL(t, ctx, db)
	assertIntegrityOK(t, ctx, db)

	for _, role := range roles {
		var count int

		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM cache_entries WHERE cache_key LIKE ?", role+":%").Scan(&count))
		assert.Equal(t, perRoleRows, count, "role %q row count", role)

		var raw string

		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT value FROM cache_entries WHERE cache_key = ?", role+":final").Scan(&raw))

		// The provider stores JSON-encoded values; decode the raw column the
		// same way a Get would.
		var marker string

		require.NoError(t, json.Unmarshal([]byte(raw), &marker))
		assert.Equal(t, "done", marker, "role %q final marker", role)

		// Deleted-key spot checks: 0000 and 0007 are MDel targets; 0050 is a
		// point-Delete target.
		for _, deleted := range []string{pointKey(role, 0), pointKey(role, 7), pointKey(role, 50)} {
			assertKeyAbsent(t, ctx, db, deleted)
		}
	}
}
