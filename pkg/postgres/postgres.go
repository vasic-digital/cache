// Package postgres provides a PostgreSQL-backed cache implementation of the
// Cache interface defined in digital.vasic.cache/pkg/cache.
//
// The implementation stores values in a single table keyed by the cache
// key. TTL is encoded as an absolute expires_at column; expired rows are
// invisible to Get/Exists and removed by a periodic GC goroutine that the
// caller controls via Start/Stop.
//
// Design choices:
//
//   - One table, one schema. Callers may set a custom Config.SchemaName /
//     Config.TableName to namespace by application; otherwise the default
//     "lava_cache" schema and "response_cache" table are used.
//   - The implementation does NOT auto-run migrations on connect. The
//     CreateSchema method is exported so the caller can decide when (and
//     in which transaction) to apply the schema. This matches the
//     vasic-digital convention of separating data-plane and migration-plane
//     concerns.
//   - The pgx connection pool is owned by the caller. Pass an existing
//     *pgxpool.Pool from your application's database wiring, or use
//     ConnectFromURL for the simple case.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultSchema is the schema name used when Config.SchemaName is empty.
const DefaultSchema = "lava_cache"

// DefaultTable is the table name used when Config.TableName is empty.
const DefaultTable = "response_cache"

// Config configures a Postgres cache Client.
//
// At least one of (Pool, URL) MUST be set. If both are set, Pool wins.
// The pool is NOT closed by Client.Close — the caller owns its lifetime.
type Config struct {
	// Pool is an existing pgx connection pool. If non-nil, URL is ignored.
	Pool *pgxpool.Pool

	// URL is a PostgreSQL connection string used by ConnectFromURL.
	// Ignored if Pool is set. Example:
	//   postgres://user:pass@host:5432/dbname?sslmode=disable
	URL string

	// SchemaName overrides DefaultSchema. Must be a valid SQL identifier.
	SchemaName string

	// TableName overrides DefaultTable. Must be a valid SQL identifier.
	TableName string

	// GCInterval is how often the background GC sweep runs. Zero disables
	// the GC goroutine — the caller is responsible for invoking
	// PurgeExpired manually if they want expired rows reclaimed.
	GCInterval time.Duration

	// GCBatchSize bounds how many expired rows the GC sweep deletes per
	// pass. Zero defaults to 1000.
	GCBatchSize int
}

// DefaultConfig returns a Config with sensible defaults. Pool and URL
// are left empty — the caller MUST set one before calling New.
func DefaultConfig() *Config {
	return &Config{
		SchemaName:  DefaultSchema,
		TableName:   DefaultTable,
		GCInterval:  10 * time.Minute,
		GCBatchSize: 1000,
	}
}

// Client implements cache.Cache backed by a PostgreSQL table.
//
// Concurrency: all methods are safe for use by multiple goroutines.
type Client struct {
	pool       *pgxpool.Pool
	ownsPool   bool // true when ConnectFromURL created the pool
	schema     string
	table      string
	qualified  string // "schema"."table"
	gcInterval time.Duration
	gcBatch    int
	gcCancel   context.CancelFunc
	gcWG       sync.WaitGroup
	mu         sync.Mutex
	closed     bool
}

// New constructs a Client from a Config. Validates the config but does
// NOT run schema migrations (call CreateSchema for that) and does NOT
// start the GC goroutine (call Start).
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("postgres: Config is required")
	}
	if cfg.Pool == nil && cfg.URL == "" {
		return nil, errors.New("postgres: Config.Pool or Config.URL is required")
	}
	schema := cfg.SchemaName
	if schema == "" {
		schema = DefaultSchema
	}
	table := cfg.TableName
	if table == "" {
		table = DefaultTable
	}
	if !isValidIdentifier(schema) {
		return nil, fmt.Errorf("postgres: invalid SchemaName %q", schema)
	}
	if !isValidIdentifier(table) {
		return nil, fmt.Errorf("postgres: invalid TableName %q", table)
	}
	if cfg.GCInterval < 0 {
		return nil, errors.New("postgres: GCInterval must be non-negative")
	}
	if cfg.GCBatchSize < 0 {
		return nil, errors.New("postgres: GCBatchSize must be non-negative")
	}
	c := &Client{
		schema:     schema,
		table:      table,
		qualified:  fmt.Sprintf(`"%s"."%s"`, schema, table),
		gcInterval: cfg.GCInterval,
		gcBatch:    cfg.GCBatchSize,
	}
	if c.gcBatch == 0 {
		c.gcBatch = 1000
	}
	if cfg.Pool != nil {
		c.pool = cfg.Pool
		c.ownsPool = false
	}
	return c, nil
}

// ConnectFromURL is a convenience wrapper that builds a pgxpool from a
// PostgreSQL URL and returns a Client that owns it (Close will close the
// pool).
func ConnectFromURL(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("postgres: Config is required")
	}
	if cfg.URL == "" {
		return nil, errors.New("postgres: Config.URL is required")
	}
	pool, err := pgxpool.New(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres: pool: %w", err)
	}
	cfg2 := *cfg
	cfg2.Pool = pool
	c, err := New(&cfg2)
	if err != nil {
		pool.Close()
		return nil, err
	}
	c.ownsPool = true
	return c, nil
}

// CreateSchema creates the schema and table if they do not exist. Idempotent.
//
// The schema is:
//
//	CREATE SCHEMA IF NOT EXISTS <schema>;
//	CREATE TABLE IF NOT EXISTS <schema>.<table> (
//	    cache_key   TEXT PRIMARY KEY,
//	    value       BYTEA NOT NULL,
//	    expires_at  TIMESTAMPTZ
//	);
//	CREATE INDEX IF NOT EXISTS <table>_expires_idx ON <schema>.<table> (expires_at);
func (c *Client) CreateSchema(ctx context.Context) error {
	stmts := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, c.schema),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			cache_key  TEXT PRIMARY KEY,
			value      BYTEA NOT NULL,
			expires_at TIMESTAMPTZ
		)`, c.qualified),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %q ON %s (expires_at)`,
			c.table+"_expires_idx", c.qualified),
	}
	for _, s := range stmts {
		if _, err := c.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("postgres: create schema (%q): %w", s, err)
		}
	}
	return nil
}

// Get retrieves a value by key. Returns nil, nil on cache miss or when
// the entry has expired.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	q := fmt.Sprintf(`SELECT value FROM %s
		WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())`, c.qualified)
	var val []byte
	err := c.pool.QueryRow(ctx, q, key).Scan(&val)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: get %q: %w", key, err)
	}
	return val, nil
}

// Set stores a value with the given TTL. A zero TTL means the entry
// does not expire (expires_at is NULL).
//
// expires_at is computed SERVER-SIDE — `now() + <ttl> milliseconds` —
// so the expiry boundary and the `now()` compared by Get/Exists live in
// the SAME clock domain (the PostgreSQL server's). Computing the boundary
// from the Go process wall clock (time.Now().Add(ttl)) is a clock-domain
// bug: when the PG server clock and the Go client clock are skewed (common
// when PG runs in a container while the client runs on the host), a finite
// TTL shorter than the skew makes the row appear already-expired on an
// immediate read. The interval is built from an integer-millisecond
// parameter via make_interval so the TTL value itself is parameterised
// (no string interpolation of caller-influenced data).
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl > 0 {
		q := fmt.Sprintf(`INSERT INTO %s (cache_key, value, expires_at)
			VALUES ($1, $2, now() + make_interval(secs => $3::double precision))
			ON CONFLICT (cache_key) DO UPDATE
				SET value = EXCLUDED.value,
				    expires_at = EXCLUDED.expires_at`, c.qualified)
		secs := ttl.Seconds()
		if _, err := c.pool.Exec(ctx, q, key, value, secs); err != nil {
			return fmt.Errorf("postgres: set %q: %w", key, err)
		}
		return nil
	}
	// Zero TTL: never expires (expires_at stays NULL).
	q := fmt.Sprintf(`INSERT INTO %s (cache_key, value, expires_at)
		VALUES ($1, $2, NULL)
		ON CONFLICT (cache_key) DO UPDATE
			SET value = EXCLUDED.value,
			    expires_at = EXCLUDED.expires_at`, c.qualified)
	if _, err := c.pool.Exec(ctx, q, key, value); err != nil {
		return fmt.Errorf("postgres: set %q: %w", key, err)
	}
	return nil
}

// Delete removes a key. Deleting a non-existent key is not an error.
func (c *Client) Delete(ctx context.Context, key string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE cache_key = $1`, c.qualified)
	if _, err := c.pool.Exec(ctx, q, key); err != nil {
		return fmt.Errorf("postgres: delete %q: %w", key, err)
	}
	return nil
}

// Exists reports whether the key is present and not expired.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	q := fmt.Sprintf(`SELECT 1 FROM %s
		WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())`, c.qualified)
	var one int
	err := c.pool.QueryRow(ctx, q, key).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("postgres: exists %q: %w", key, err)
	}
	return true, nil
}

// PurgeExpired deletes up to GCBatchSize rows whose expires_at is in
// the past. Returns the number of rows deleted. Safe to call concurrently
// with the background GC goroutine.
func (c *Client) PurgeExpired(ctx context.Context) (int64, error) {
	q := fmt.Sprintf(`WITH expired AS (
		SELECT cache_key FROM %s
		WHERE expires_at IS NOT NULL AND expires_at <= NOW()
		LIMIT %d
	)
	DELETE FROM %s WHERE cache_key IN (SELECT cache_key FROM expired)`,
		c.qualified, c.gcBatch, c.qualified)
	tag, err := c.pool.Exec(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("postgres: purge expired: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Start launches the background GC goroutine if Config.GCInterval > 0.
// Calling Start twice is a no-op. Stop or Close terminates the goroutine.
func (c *Client) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gcInterval <= 0 || c.gcCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.gcCancel = cancel
	c.gcWG.Add(1)
	go func() {
		defer c.gcWG.Done()
		ticker := time.NewTicker(c.gcInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = c.PurgeExpired(ctx)
			}
		}
	}()
}

// Stop halts the background GC goroutine if it is running. Idempotent.
func (c *Client) Stop() {
	c.mu.Lock()
	cancel := c.gcCancel
	c.gcCancel = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.gcWG.Wait()
}

// Close stops the GC goroutine and, if this Client owns the pool
// (created via ConnectFromURL), closes it. Idempotent.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	c.Stop()
	if c.ownsPool && c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// HealthCheck verifies the database is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres health check: %w", err)
	}
	return nil
}

// Underlying returns the pgx pool for advanced operations. Returning
// it is intentional — callers may want to run their own queries on the
// same pool. Lifecycle is still owned by the Client when ownsPool is
// true.
func (c *Client) Underlying() *pgxpool.Pool {
	return c.pool
}

// isValidIdentifier returns true if s is a safe SQL identifier:
// non-empty, ASCII letters / digits / underscore, must begin with a
// letter or underscore. Rejects anything that would require quoting
// beyond the simple double-quote we apply.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
