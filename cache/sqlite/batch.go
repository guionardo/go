package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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
// empty input yields no chunks, so callers naturally issue no query (D-02).
func chunksOf[K comparable](keys []K, size int) [][]K {
	if len(keys) == 0 {
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

// MSetFunc stores multiple key-value pairs atomically.
//
// TRANSIENT STUB: no-op so the MSet target assertions fail.
func (c *sqliteCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	return nil
}

// MDelFunc removes multiple keys.
//
// TRANSIENT STUB: no-op so the MDel target assertions fail.
func (c *sqliteCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	return nil
}
