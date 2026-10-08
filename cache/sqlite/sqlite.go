package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver (CGO-free)

	"github.com/guionardo/go/cache"
)

// sqliteCache is the SQLite-backed provider. It implements the cache.cacher
// primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc and the batch
// methods); New embeds it in the batchCache adapter built on
// cache.NewConcreteCache, which supplies the shared Cache surface (singleflight
// GetOrSet dedup, Cache interface) plus the sqlite.Optimizable maintenance
// surface.
//
// The database pool is pinned to a single connection in both modes: it is
// mandatory for :memory: correctness (each pooled connection would otherwise
// open a private database) and provides in-process serialization.
type sqliteCache[K comparable, V any] struct {
	db            *sql.DB
	defaultTTL    time.Duration
	sweepInterval time.Duration
	initErr       error
	closed        atomic.Bool
	stop          chan struct{}
	done          chan struct{}
}

// cacheDirPerm is the permission mode for cache directories.
const cacheDirPerm = 0o700

var (
	// ErrClosed is returned by operations attempted on a closed cache.
	ErrClosed = errors.New("cache/sqlite: cache is closed")

	// ErrInvalidPath is returned when a path or cache name fails validation.
	ErrInvalidPath = errors.New("cache/sqlite: invalid path")

	logger = sync.OnceValue[*slog.Logger](func() *slog.Logger {
		return slog.With(slog.String("module", "cache/sqlite"))
	})
)

// New creates a new SQLite cache provider with optional functional options.
//
// New never fails: open or validation errors are recorded and deferred, and
// the first operation that needs the database returns them wrapped. Returns a
// cache.BatchCache sharing the in-memory singleflight GetOrSet; the returned
// value also implements Optimizable (type-assert to reach Checkpoint/Vacuum).
func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	c := &sqliteCache[K, V]{
		defaultTTL:    cfg.DefaultTTL,
		sweepInterval: cfg.SweepInterval,
	}
	c.initErr = c.open(context.Background(), cfg)

	return &batchCache[K, V]{
		BatchCache: cache.NewConcreteCache[K, V](c),
		provider:   c,
	}
}

// open resolves the location, creates the cache directory for file mode, opens
// the database with the DSN-carried pragmas, pins the pool to one connection,
// and bootstraps the schema.
func (c *sqliteCache[K, V]) open(ctx context.Context, cfg *Config) error {
	path, memory, err := resolveLocation(cfg, os.UserCacheDir)
	if err != nil {
		return err
	}

	if !memory {
		if err := os.MkdirAll(filepath.Dir(path), cacheDirPerm); err != nil {
			return err
		}
	}

	db, err := sql.Open("sqlite", buildDSN(path, memory, cfg.AutoCheckpoint))
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	c.db = db

	if err := c.bootstrap(ctx); err != nil {
		_ = db.Close()
		c.db = nil

		return err
	}

	if !memory {
		c.verifyJournalMode(ctx)
	}

	// TTL-03: best-effort sweep, synchronous before New returns (D-10).
	c.sweep(ctx)

	// TTL-04: opt-in periodic sweeper (D-08). A non-positive interval leaves
	// stop/done nil and starts no goroutine.
	if c.sweepInterval > 0 {
		c.startSweeper()
	}

	return nil
}

// verifyJournalMode reads back the effective journal mode on the pinned
// connection. A mismatch only logs a warning — construction never fails on the
// read-back (WAL silently degrades on filesystems without shared memory).
func (c *sqliteCache[K, V]) verifyJournalMode(ctx context.Context) {
	var mode string
	if err := c.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		logger().Warn("cache/sqlite: journal_mode is not wal", "mode", mode, "error", err)
	}
}

// bootstrap creates the fixed schema inside one transaction. Because the DSN
// carries _txlock=immediate, BeginTx issues BEGIN IMMEDIATE, so concurrent
// first-opens serialize at BEGIN and the CREATE ... IF NOT EXISTS statements
// are idempotent.
func (c *sqliteCache[K, V]) bootstrap(ctx context.Context) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	if _, err := tx.ExecContext(ctx, CreateTableSQL); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, CreateIndexSQL); err != nil {
		return err
	}

	return tx.Commit()
}

// check returns the deferred open/validation error, or ErrClosed once the
// cache has been closed. It is the first statement of every operation.
func (c *sqliteCache[K, V]) check() error {
	if c.initErr != nil {
		return fmt.Errorf("cache/sqlite: %w", c.initErr)
	}

	if c.closed.Load() {
		return fmt.Errorf("cache/sqlite: %w", ErrClosed)
	}

	return nil
}

// GetFunc retrieves a value by key. Returns cache.ErrMiss if not found or
// expired. Only sql.ErrNoRows maps to ErrMiss; every other failure (including
// malformed JSON) is returned as itself.
func (c *sqliteCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
	var zero V
	if err := c.check(); err != nil {
		return zero, err
	}

	var data string

	err := c.db.QueryRowContext(ctx, SelectSQL, fmt.Sprint(key), time.Now().UnixNano()).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)
	}

	if err != nil {
		return zero, fmt.Errorf("cache/sqlite: %w", err)
	}

	var value V
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return zero, fmt.Errorf("cache/sqlite: %w", err)
	}

	return value, nil
}

// SetFunc stores a value with optional per-key TTL. The expiry is stored as
// an absolute UnixNano integer; a missing TTL stores SQL NULL. A marshal
// failure returns an error and stores nothing.
func (c *sqliteCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	if err := c.check(); err != nil {
		return err
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	expiresAt, hasTTL := c.resolveTTL(ttl...)

	var exp any
	if hasTTL {
		exp = expiresAt
	}

	if _, err := c.db.ExecContext(ctx, UpsertSQL, fmt.Sprint(key), string(data), exp); err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	return nil
}

// DeleteFunc removes a key. Deleting a missing key is a no-op.
func (c *sqliteCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
	if err := c.check(); err != nil {
		return err
	}

	if _, err := c.db.ExecContext(ctx, DeleteSQL, fmt.Sprint(key)); err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	return nil
}

// CloseFunc cancels and waits for the periodic sweeper (when present), then
// closes the database (the last connection checkpoints the WAL and removes
// the -wal/-shm sidecars). It is idempotent and returns nil even when the
// handle deferred an open error.
func (c *sqliteCache[K, V]) CloseFunc() error {
	if c.closed.CompareAndSwap(false, true) {
		if c.stop != nil {
			close(c.stop)
			<-c.done
		}

		if c.db != nil {
			return c.db.Close()
		}
	}

	return nil
}

// MGetFunc retrieves values for multiple keys, skipping missing or errored
// keys.
//
// Phase 16: replace with a chunked IN-list query (BATCH-01).
func (c *sqliteCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	result := make(map[K]V, len(keys))
	for _, key := range keys {
		value, err := c.GetFunc(ctx, key)
		if err != nil {
			continue // missing or errored key — skip
		}

		result[key] = value
	}

	return result
}

// MSetFunc stores multiple key-value pairs, one point operation per pair.
//
// Phase 16: replace with a single-transaction prepared upsert (BATCH-02).
func (c *sqliteCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	for key, value := range items {
		if err := c.SetFunc(ctx, key, value, ttl...); err != nil {
			return err
		}
	}

	return nil
}

// MDelFunc removes multiple keys, one point operation per key.
//
// Phase 16: replace with a chunked IN-list delete (BATCH-03).
func (c *sqliteCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	for _, key := range keys {
		if err := c.DeleteFunc(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

// resolveTTL resolves the effective expiration for a Set operation.
// Precedence: per-call TTL > provider-level default > no expiry.
func (c *sqliteCache[K, V]) resolveTTL(ttl ...time.Duration) (int64, bool) {
	if len(ttl) > 0 && ttl[0] > 0 {
		return time.Now().Add(ttl[0]).UnixNano(), true
	}

	if c.defaultTTL > 0 {
		return time.Now().Add(c.defaultTTL).UnixNano(), true
	}

	return 0, false
}
