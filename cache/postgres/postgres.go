// Package postgres provides a Postgres backend for cache.Cache[K, V].
//
// Uses pgx/v5 with connection pooling and automatic table creation
// (CREATE UNLOGGED TABLE for performance). The schema is a simple
// key-value table with an optional expires_at column. A background
// sweep goroutine removes expired rows periodically. Uses pg_prewarm
// to preload cache data into shared buffers after restarts (best-effort,
// silently skipped if the extension is not installed).
// Values are serialized via encoding/json.
//
// Usage:
//
//	c, err := postgres.New[string, string](
//	    postgres.WithConnString("postgres://user:pass@localhost:5432/cache?sslmode=disable"),
//	)
//	c.Set(ctx, "key", "value")
//	v, err := c.Get(ctx, "key")
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guionardo/go/cache"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
)

func log() *slog.Logger {
	loggerOnce.Do(func() {
		logger = slog.With(slog.String("module", "cache/postgres"))
	})
	return logger
}

// Cache implements cache.Cache[K, V] using a PostgreSQL backend.
type Cache[K comparable, V any] struct {
	pool          *pgxpool.Pool
	tableName     string
	defaultTTL    time.Duration
	sweepInterval time.Duration
	stop          chan struct{}
}

// New creates a new Postgres cache provider with optional functional options.
// The constructor creates the cache table (if not exists) and starts the
// background sweep goroutine.
func New[K comparable, V any](opts ...Option) (*Cache[K, V], error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	pool, err := pgxpool.New(context.Background(), cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("cache/postgres: %w", err)
	}

	if _, err := pool.Exec(context.Background(), CreateTableSQL); err != nil {
		return nil, fmt.Errorf("cache/postgres: %w", err)
	}

	if _, err := pool.Exec(context.Background(), PrewarmSQL); err != nil {
		log().Warn("pg_prewarm not available, skipping prewarm", "error", err)
	}

	c := &Cache[K, V]{
		pool:          pool,
		tableName:     cfg.TableName,
		defaultTTL:    cfg.DefaultTTL,
		sweepInterval: cfg.SweepInterval,
		stop:          make(chan struct{}),
	}

	go c.sweepLoop()

	return c, nil
}

// Get retrieves a value by key. Returns cache.ErrMiss if not found or expired.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	query := fmt.Sprintf(
		"SELECT value FROM %s WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	var valueJSON string
	err := c.pool.QueryRow(ctx, query, fmt.Sprint(key)).Scan(&valueJSON)
	if err == pgx.ErrNoRows {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", cache.ErrMiss)
	}
	if err != nil {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", err)
	}

	var value V
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", err)
	}

	return value, nil
}

// Set stores a value with optional per-key TTL.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/postgres: %w", err)
	}

	expiresAt := c.resolveTTL(ttl...)

	query := fmt.Sprintf(
		"INSERT INTO %s (cache_key, value, expires_at) VALUES ($1, $2, $3) ON CONFLICT (cache_key) DO UPDATE SET value = $2, expires_at = $3",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	if _, err := c.pool.Exec(ctx, query, fmt.Sprint(key), string(data), expiresAt); err != nil {
		return fmt.Errorf("cache/postgres: %w", err)
	}

	return nil
}

// Delete removes a key from the cache. Idempotent — deleting a missing key succeeds.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE cache_key = $1",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	if _, err := c.pool.Exec(ctx, query, fmt.Sprint(key)); err != nil {
		return fmt.Errorf("cache/postgres: %w", err)
	}

	return nil
}

// GetOrSet returns the existing value or computes, stores, and returns it.
func (c *Cache[K, V]) GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error) {
	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	computed, err := setter()
	if err != nil {
		var zero V
		return zero, err
	}

	if err := c.Set(ctx, key, computed, ttl...); err != nil {
		var zero V
		return zero, err
	}

	return computed, nil
}

// Close shuts down the background sweep goroutine and closes the connection pool.
// Safe to call multiple times (idempotent).
func (c *Cache[K, V]) Close() error {
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}

	c.pool.Close()
	return nil
}

// resolveTTL converts the optional TTL to an expiration timestamp.
// Returns nil for no expiry.
func (c *Cache[K, V]) resolveTTL(ttl ...time.Duration) *time.Time {
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().Add(ttl[0])
		return &t
	}
	if c.defaultTTL > 0 {
		t := time.Now().Add(c.defaultTTL)
		return &t
	}
	return nil
}

// compile-time interface assertion
var _ cache.Cache[string, any] = (*Cache[string, any])(nil)
