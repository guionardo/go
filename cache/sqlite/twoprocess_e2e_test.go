//go:build e2e

package sqlite_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite" // registers the "sqlite" driver for the raw harness handles
)

// Two-process WAL spike (Phase 16, D-11): the parent spawns two child
// processes through the TestMain re-exec pattern; each child drives a raw
// database/sql handle against one shared, fresh cache file (100 point
// upsert/read-back pairs plus a 20-key prepared-upsert transaction every 10th
// iteration) and exits 0. The policy arm wraps the single connection
// establishing PingContext in the candidate D-06 bounded busy-only retry; the
// raw arm (SQLITE_E2E_RETRY=0) pings once and exists to reproduce the DI-15-01
// first-open race as observational counter-evidence, never as a gate.
const (
	helperEnv = "SQLITE_E2E_HELPER"
	dbEnv     = "SQLITE_E2E_DB"
	roleEnv   = "SQLITE_E2E_ROLE"
	retryEnv  = "SQLITE_E2E_RETRY"

	// e2eDSN mirrors the dsnSuffixFile literal in dsn.go. This mirror belongs
	// to the spike only and is eliminated in Plan 16-04, when the children
	// rewire to the shipped provider (whose open() owns the real DSN).
	e2eDSN = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"

	// iterations is the per-child point-operation count. The parent asserts
	// iterations point rows + (iterations/10)*20 batch rows + 1 final marker
	// per role (301 for iterations = 100).
	iterations = 100

	// retryBudget matches the candidate D-06 bound: the same 5 s that
	// _busy_timeout=5000 sets in the DSN.
	retryBudget = 5 * time.Second

	// retryBackoff is the tiny probe-proven wait between busy retries.
	retryBackoff = 5 * time.Millisecond

	// e2eCreateTableSQL / e2eCreateIndexSQL / e2eUpsertSQL / e2eSelectSQL
	// mirror schema.go's constants for the raw harness handles (same
	// elimination note as e2eDSN). The DDL is what bootstraps the shared fresh
	// file; both children race it idempotently (CREATE ... IF NOT EXISTS under
	// BEGIN IMMEDIATE).
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

	e2eSelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`
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

// helperMain is the child entry point. It opens a raw handle on the shared
// database, establishes the connection through the candidate retry policy (or
// one plain ping in the raw arm), runs the write/read workload, and finishes
// with the role's final marker. It returns the process exit code.
func helperMain() int {
	dbPath := os.Getenv(dbEnv)
	role := os.Getenv(roleEnv)

	db, err := sql.Open("sqlite", dbPath+e2eDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: open: %v\n", role, err)

		return 1
	}
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx := context.Background()

	// Connection establishment is the DI-15-01 seam: the fresh-file WAL
	// conversion returns an immediate SQLITE_BUSY that bypasses the busy
	// handler. The policy arm wraps exactly this one PingContext; the raw arm
	// pings exactly once and surfaces the failure as-is.
	ping := db.PingContext
	if os.Getenv(retryEnv) == "1" {
		ping = func(ctx context.Context) error { return retryPing(ctx, db) }
	}

	if err := ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: ping: %v\n", role, err)

		return 1
	}

	// The shared fresh file needs its schema; both children race this
	// idempotent DDL exactly like the provider's bootstrap() does.
	if err := e2eBootstrap(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: bootstrap: %v\n", role, err)

		return 1
	}

	for i := range iterations {
		key := fmt.Sprintf("%s:%04d", role, i)
		value := fmt.Sprintf("v-%d", i)

		if _, err := db.ExecContext(ctx, e2eUpsertSQL, key, value, nil); err != nil {
			fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: point upsert %d: %v\n", role, i, err)

			return 1
		}

		var got string
		if err := db.QueryRowContext(ctx, e2eSelectSQL, key, time.Now().UnixNano()).Scan(&got); err != nil {
			fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: read-back %d: %v\n", role, i, err)

			return 1
		}

		if got != value {
			fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: read-back %d: got %q want %q\n", role, i, got, value)

			return 1
		}

		if i%10 == 0 {
			if err := e2eBatchUpsert(ctx, db, role, i); err != nil {
				fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: batch %d: %v\n", role, i, err)

				return 1
			}
		}
	}

	if _, err := db.ExecContext(ctx, e2eUpsertSQL, role+":final", "done", nil); err != nil {
		fmt.Fprintf(os.Stderr, "sqlite e2e helper %s: final marker: %v\n", role, err)

		return 1
	}

	return 0
}

// e2eBootstrap creates the fixed schema mirroring the provider's bootstrap():
// one BEGIN IMMEDIATE transaction (via _txlock=immediate) around the idempotent
// CREATE ... IF NOT EXISTS statements.
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

// e2eBatchUpsert writes one 20-key prepared-upsert transaction: the batch
// shape BATCH-02 ships, exercised cross-process for the spike's row counts.
// _txlock=immediate makes BeginTx issue BEGIN IMMEDIATE.
func e2eBatchUpsert(ctx context.Context, db *sql.DB, role string, iteration int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	stmt, err := tx.PrepareContext(ctx, e2eUpsertSQL)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for j := range 20 {
		key := fmt.Sprintf("%s:batch:%04d:%02d", role, iteration, j)
		if _, err := stmt.ExecContext(ctx, key, "b", nil); err != nil {
			return fmt.Errorf("exec %d: %w", j, err)
		}
	}

	return tx.Commit()
}

// busyError is the minimal typed view of SQLITE_BUSY errors — the same
// interface shape Plan 16-02 ships in retry.go (the D-06 policy).
type busyError interface{ Code() int }

// isBusyError classifies SQLITE_BUSY by primary result code. The driver
// enables extended result codes, so compare code&0xff (SQLITE_BUSY_SNAPSHOT
// 517 etc. still classify as busy).
func isBusyError(err error) bool {
	var be busyError

	return errors.As(err, &be) && be.Code()&0xff == 5
}

// retryPing wraps exactly one connection establishment (PingContext) in the
// candidate D-06 policy: busy-only classification, 5 ms backoff, 5 s absolute
// deadline. A failed ping discards the connection, so each retry re-dials.
func retryPing(ctx context.Context, db *sql.DB) error {
	deadline := time.Now().Add(retryBudget)

	for {
		err := db.PingContext(ctx)
		if err == nil || !isBusyError(err) || time.Now().After(deadline) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryBackoff):
		}
	}
}

// TestTwoProcessSpikeContention is the parent: two OS processes share one
// fresh database file. The policy arm (default, SQLITE_E2E_RETRY=1) must be
// deterministically clean. Setting SQLITE_E2E_RETRY=0 selects the raw arm,
// which exists to observe the DI-15-01 first-open race and is not expected to
// pass reliably — the test never gate-keeps on probabilistic raw failures.
func TestTwoProcessSpikeContention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shared.db")

	retry := os.Getenv(retryEnv)
	if retry == "" {
		retry = "1"
	}

	roles := []string{"alpha", "beta"}

	cmds := make([]*exec.Cmd, 0, len(roles))
	stderrs := make([]*bytes.Buffer, 0, len(roles))

	for _, role := range roles {
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(),
			helperEnv+"=1",
			dbEnv+"="+dbPath,
			roleEnv+"="+role,
			retryEnv+"="+retry,
		)

		var stdout, stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Start())

		cmds = append(cmds, cmd)
		stderrs = append(stderrs, &stderr)
	}

	t.Cleanup(func() {
		for _, cmd := range cmds {
			if cmd.ProcessState == nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}
	})

	for i, cmd := range cmds {
		err := cmd.Wait()
		require.NoErrorf(t, err, "child %q exited non-zero\nstderr:\n%s", roles[i], stderrs[i].String())
	}

	db, err := sql.Open("sqlite", dbPath+e2eDSN)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := t.Context()

	var journalMode string

	require.NoError(t, db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
	assert.Equal(t, "wal", journalMode)

	var integrity string

	require.NoError(t, db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity))
	assert.Equal(t, "ok", integrity)

	// Per-role row count: iterations point keys + (iterations/10)*20 batch
	// keys + 1 final marker (301 for iterations = 100).
	const perRoleRows = iterations + (iterations/10)*20 + 1

	for _, role := range roles {
		var count int

		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM cache_entries WHERE cache_key LIKE ?", role+":%").Scan(&count))
		assert.Equal(t, perRoleRows, count, "role %q row count", role)

		var marker string

		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT value FROM cache_entries WHERE cache_key = ?", role+":final").Scan(&marker))
		assert.Equal(t, "done", marker, "role %q final marker", role)
	}
}
