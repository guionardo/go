package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// msetPair is one pre-marshaled batch entry: the stringified cache key and
// the JSON data. Pairs are built up-front by MSetFunc so a marshal error can
// never leave a partial write (D-04).
type msetPair struct{ key, data string }

// chunkSize bounds the number of keys per IN-list query (D-01: fixed internal
// constant, no option surface). 100 keys + the bound now = 101 parameters —
// far below SQLite's 32766-variable limit.
const chunkSize = 100

// uniqueKeys deduplicates keys by their fmt.Sprint form, preserving the
// first-seen order (D-02). Two distinct key values whose stringified forms
// collide collapse to the first occurrence: the SQL keyspace is the
// stringified form, so a second occurrence would address the same row.
func uniqueKeys[K comparable](keys []K) []K {
	seen := make(map[string]struct{}, len(keys))
	unique := make([]K, 0, len(keys))

	for _, key := range keys {
		s := fmt.Sprint(key)
		if _, ok := seen[s]; ok {
			continue
		}

		seen[s] = struct{}{}

		unique = append(unique, key)
	}

	return unique
}

// chunksOf splits keys into size-bounded chunks preserving input order. An
// empty input — or a non-positive size — yields no chunks, so callers
// naturally issue no query (D-02).
func chunksOf[K comparable](keys []K, size int) [][]K {
	if len(keys) == 0 || size <= 0 {
		return nil
	}

	var chunks [][]K

	for start := 0; start < len(keys); start += size {
		end := min(start+size, len(keys))
		chunks = append(chunks, keys[start:end])
	}

	return chunks
}

// inPlaceholders returns the "?,?,..." marker list for n keys. This is the
// only text ever generated into the batch SQL — keys and values stay bound
// parameters, never interpolated.
func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// MGetFunc retrieves values for multiple keys, skipping missing or errored
// keys. It issues one chunked IN-list query per 100 deduplicated keys; expired
// rows are absent by the SQL predicate and undecodable rows are skipped at
// debug. A failed chunk is logged and skipped — remaining chunks still run
// (D-03; the frozen signature has no error channel, so a canceled context
// yields partial results by contract).
func (c *sqliteCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	result := make(map[K]V, len(keys))

	if err := c.check(); err != nil {
		logger().Warn("cache/sqlite: batch get skipped", "error", err)

		return result
	}

	now := time.Now().UnixNano()

	for _, chunk := range chunksOf(uniqueKeys(keys), chunkSize) {
		c.mgetChunk(ctx, chunk, now, result)
	}

	return result
}

// mgetChunk runs one IN-list query for a single chunk and folds the decoded
// rows into result. The original K values are recovered from the chunk's own
// input: fmt.Sprint is not invertible, so a per-chunk reverse map is the only
// correct mapping (Finding 5). The rows handle is closed before this function
// returns, so the next chunk (or any queued operation) can use the pinned
// connection — an unconsumed *sql.Rows would deadlock the single-conn pool.
func (c *sqliteCache[K, V]) mgetChunk(ctx context.Context, chunk []K, now int64, result map[K]V) {
	byKey := make(map[string]K, len(chunk))
	args := make([]any, 0, len(chunk)+1)

	for _, key := range chunk {
		s := fmt.Sprint(key)
		byKey[s] = key
		args = append(args, s)
	}

	args = append(args, now)

	rows, err := c.db.QueryContext(ctx, fmt.Sprintf(MGetSelectSQL, inPlaceholders(len(chunk))), args...)
	if err != nil {
		logger().Warn("cache/sqlite: batch get chunk failed", "keys", len(chunk), "error", err)

		return
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cacheKey, data string

		if err := rows.Scan(&cacheKey, &data); err != nil {
			logger().Warn("cache/sqlite: batch get row failed", "error", err)

			continue
		}

		var value V
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			logger().Debug("cache/sqlite: batch get undecodable entry skipped", "key", cacheKey)

			continue
		}

		result[byKey[cacheKey]] = value
	}

	if err := rows.Err(); err != nil {
		logger().Warn("cache/sqlite: batch get chunk iteration failed", "error", err)
	}
}

// MSetFunc stores multiple key-value pairs as one atomic batch (D-04,
// BATCH-02). Every value is marshaled up-front — before the transaction opens
// — so any error (marshal, begin, prepare, exec, commit) leaves zero rows from
// the batch. Exactly one TTL is resolved for the whole batch, and one prepared
// upsert runs per pair inside a single BEGIN IMMEDIATE transaction (the DSN
// carries _txlock=immediate).
func (c *sqliteCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	if err := c.check(); err != nil {
		return err
	}

	if len(items) == 0 {
		return nil // D-02: an empty batch opens no transaction
	}

	pairs := make([]msetPair, 0, len(items))

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache/sqlite: %w", err)
		}

		pairs = append(pairs, msetPair{key: fmt.Sprint(key), data: string(data)})
	}

	expiresAt, hasTTL := c.resolveTTL(ttl...)

	var exp any
	if hasTTL {
		exp = expiresAt
	}

	return c.msetTx(ctx, pairs, exp)
}

// msetTx writes every pre-marshaled pair inside one transaction with a single
// prepared upsert. Commit success commits the whole batch; any error rolls
// everything back via the deferred Rollback — the no-partial-writes invariant
// (Pitfall 4). No per-pair autocommit path exists.
func (c *sqliteCache[K, V]) msetTx(ctx context.Context, pairs []msetPair, exp any) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	defer func() { _ = tx.Rollback() }() // no-op after Commit — atomicity backstop

	stmt, err := tx.PrepareContext(ctx, UpsertSQL)
	if err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	defer func() { _ = stmt.Close() }()

	for _, p := range pairs {
		if _, err := stmt.ExecContext(ctx, p.key, p.data, exp); err != nil {
			return fmt.Errorf("cache/sqlite: %w", err) // defer rolls the batch back
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache/sqlite: %w", err)
	}

	return nil
}

// MDelFunc removes multiple keys via chunked IN-list deletes (D-05, BATCH-03).
// Duplicate and missing keys are no-ops; the first chunk error returns the
// wrapped error (fail-fast — partial deletion is acceptable because MDel is
// idempotent and a caller may safely retry).
func (c *sqliteCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	if err := c.check(); err != nil {
		return err
	}

	for _, chunk := range chunksOf(uniqueKeys(keys), chunkSize) {
		strKeys := make([]any, 0, len(chunk))

		for _, key := range chunk {
			strKeys = append(strKeys, fmt.Sprint(key))
		}

		if _, err := c.db.ExecContext(ctx, fmt.Sprintf(MDelSQL, inPlaceholders(len(chunk))), strKeys...); err != nil {
			return fmt.Errorf("cache/sqlite: %w", err)
		}
	}

	return nil
}
