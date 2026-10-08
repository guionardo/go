package sqlite

import (
	"context"
	"fmt"

	"github.com/guionardo/go/cache"
)

// Optimizable exposes explicit storage maintenance for callers that need
// to reclaim disk space. New returns a cache.BatchCache whose dynamic type
// implements this interface; callers type-assert to reach it:
//
//	if opt, ok := c.(sqlite.Optimizable); ok {
//		_ = opt.Checkpoint(ctx)
//	}
//
// Maintenance executes only when these methods are called — the provider
// never schedules checkpointing or vacuuming on its own.
type Optimizable interface {
	// Checkpoint runs PRAGMA wal_checkpoint(TRUNCATE): a cheap operation
	// that reclaims WAL space. It returns an error when another connection
	// blocks the checkpoint.
	Checkpoint(ctx context.Context) error

	// Vacuum runs a full VACUUM: an exclusive and heavy operation that
	// rewrites the database file. Do not call it on a hot path.
	Vacuum(ctx context.Context) error
}

// batchCache is the value New returns. The frozen root concreteCache does
// not forward provider-specific methods, so the Optimizable assertion
// target is built here: the adapter embeds the shared BatchCache surface
// and delegates maintenance to the provider.
//
//nolint:decorder // the exported interface and the private adapter are declared separately for discoverability
type batchCache[K comparable, V any] struct {
	cache.BatchCache[K, V]

	provider *sqliteCache[K, V]
}

const (
	// checkpointSQL reclaims WAL space. The PRAGMA reports the attempt in
	// three columns (busy, log, checkpointed).
	checkpointSQL = "PRAGMA wal_checkpoint(TRUNCATE)"

	// vacuumSQL rewrites the database file, reclaiming free pages.
	vacuumSQL = "VACUUM"
)

// Checkpoint runs PRAGMA wal_checkpoint(TRUNCATE) — see Optimizable.Checkpoint.
func (b *batchCache[K, V]) Checkpoint(ctx context.Context) error {
	return b.provider.checkpoint(ctx)
}

// Vacuum runs a full VACUUM — see Optimizable.Vacuum.
func (b *batchCache[K, V]) Vacuum(ctx context.Context) error {
	return b.provider.vacuum(ctx)
}

// checkpoint reclaims WAL space with PRAGMA wal_checkpoint(TRUNCATE). The
// PRAGMA always returns three columns; busy != 0 means another connection
// blocked the checkpoint, which is reported as an error — never a silent
// success. Memory mode returns busy=0 and is a valid no-op.
func (c *sqliteCache[K, V]) checkpoint(ctx context.Context) error {
	if err := c.check(); err != nil {
		return err
	}

	var busy, logFrames, checkpointed int

	if err := c.db.QueryRowContext(ctx, checkpointSQL).Scan(&busy, &logFrames, &checkpointed); err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	if busy != 0 {
		return fmt.Errorf("cache/sqlite: checkpoint blocked (busy=%d)", busy)
	}

	return nil
}

// vacuum rewrites the database file to reclaim free space. It is exclusive
// and heavy: callers choose when to pay for it.
func (c *sqliteCache[K, V]) vacuum(ctx context.Context) error {
	if err := c.check(); err != nil {
		return err
	}

	if _, err := c.db.ExecContext(ctx, vacuumSQL); err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	return nil
}
