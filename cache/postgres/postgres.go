package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guionardo/go/cache"
)

// postgresCache is the PostgreSQL-backed provider. It implements the
// cache.cacher primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc);
// New wraps it in a cache.NewConcreteCache, which supplies the shared Cache
// surface (singleflight GetOrSet dedup, Cache interface).
type postgresCache[K comparable, V any] struct {
	pool          *pgxpool.Pool
	tableName     string
	defaultTTL    time.Duration
	sweepInterval time.Duration
	stop          chan struct{}
}

var logger = sync.OnceValue[*slog.Logger](func() *slog.Logger {
	return slog.With(slog.String("module", "cache/postgres"))
})

// New creates a new Postgres cache provider with optional functional options.
// The constructor creates the cache table (if not exists) and starts the
// background sweep goroutine.
func New[K comparable, V any](opts ...Option) (cache.BatchCache[K, V], error) {
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
		logger().Warn("pg_prewarm not available, skipping prewarm", "error", err)
	}

	c := &postgresCache[K, V]{
		pool:          pool,
		tableName:     cfg.TableName,
		defaultTTL:    cfg.DefaultTTL,
		sweepInterval: cfg.SweepInterval,
		stop:          make(chan struct{}),
	}

	go c.sweepLoop()

	return cache.NewConcreteCache(c), nil
}

// GetFunc retrieves a value by key. Returns cache.ErrMiss if not found or expired.
func (c *postgresCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
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

// SetFunc stores a value with optional per-key TTL.
func (c *postgresCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
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

// DeleteFunc removes a key from the cache. Idempotent — deleting a missing key succeeds.
func (c *postgresCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE cache_key = $1",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	if _, err := c.pool.Exec(ctx, query, fmt.Sprint(key)); err != nil {
		return fmt.Errorf("cache/postgres: %w", err)
	}

	return nil
}

// CloseFunc shuts down the background sweep goroutine and closes the connection pool.
// Safe to call multiple times (idempotent).
func (c *postgresCache[K, V]) CloseFunc() error {
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}

	c.pool.Close()
	return nil
}

// MGetFunc retrieves values for multiple keys via pgx SendBatch.
// Each key is a parameterized SELECT query in the batch; missing keys
// are silently excluded from the result (D-06 best-effort).
func (c *postgresCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	query := fmt.Sprintf(
		"SELECT value FROM %s WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	batch := &pgx.Batch{}
	for _, key := range keys {
		batch.Queue(query, fmt.Sprint(key))
	}

	br := c.pool.SendBatch(ctx, batch)
	defer br.Close()

	result := make(map[K]V, len(keys))
	for _, key := range keys {
		var valueJSON string
		err := br.QueryRow().Scan(&valueJSON)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // missing key — skip
		}
		if err != nil {
			continue // query error — skip (D-06 best-effort)
		}
		var value V
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			continue // deserialization error — skip
		}
		result[key] = value
	}
	return result
}

// MSetFunc stores multiple key-value pairs via pgx SendBatch.
// Uses INSERT ON CONFLICT (upsert) for each key. Errors are accumulated
// via errors.Join (D-06 best-effort).
func (c *postgresCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (cache_key, value, expires_at) VALUES ($1, $2, $3) ON CONFLICT (cache_key) DO UPDATE SET value = $2, expires_at = $3",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	batch := &pgx.Batch{}
	expiresAt := c.resolveTTL(ttl...)

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache/postgres: %w", err) // hard failure on marshal
		}
		batch.Queue(query, fmt.Sprint(key), string(data), expiresAt)
	}

	br := c.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Consume results (pgx requires reading all results from SendBatch)
	var errs []error
	for range items {
		if _, err := br.Exec(); err != nil {
			errs = append(errs, fmt.Errorf("cache/postgres: %w", err))
		}
	}

	return errors.Join(errs...)
}

// MDelFunc removes multiple keys via pgx SendBatch.
// Uses a parameterized DELETE query for each key. Errors are accumulated
// via errors.Join (D-06 best-effort). Deleting already-missing keys is
// idempotent (DELETE on non-existent rows affects 0 rows — no error).
func (c *postgresCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE cache_key = $1",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	batch := &pgx.Batch{}
	for _, key := range keys {
		batch.Queue(query, fmt.Sprint(key))
	}

	br := c.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Consume results (pgx requires reading all results)
	var errs []error
	for range keys {
		if _, err := br.Exec(); err != nil {
			errs = append(errs, fmt.Errorf("cache/postgres: %w", err))
		}
	}

	return errors.Join(errs...)
}

// resolveTTL converts the optional TTL to an expiration timestamp.
// Returns nil for no expiry.
func (c *postgresCache[K, V]) resolveTTL(ttl ...time.Duration) *time.Time {
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
