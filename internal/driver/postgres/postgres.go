// Package postgres provides a PostgreSQL driver for migrations.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a wrapper over the PostgreSQL connection pool.
type DB struct {
	Pool        *pgxpool.Pool
	SchemaTable string
	LockKey     int64
}

// Connect creates a new database connection and initializes service tables.
func Connect(ctx context.Context, dsn, schemaTable string, lockKey int64) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	db := &DB{Pool: pool, SchemaTable: schemaTable, LockKey: lockKey}
	if err := db.ensureTables(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the connection pool.
func (d *DB) Close() { d.Pool.Close() }

func (d *DB) ensureTables(ctx context.Context) error {
	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id              BIGSERIAL PRIMARY KEY,
    version         BIGINT NOT NULL,
    name            TEXT NOT NULL,
    checksum        TEXT NOT NULL,
    status          TEXT NOT NULL,
    applied_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    execution_ms    BIGINT DEFAULT 0,
    error_text      TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS %s_version_uq ON %s (version);
`, d.SchemaTable, d.SchemaTable, d.SchemaTable)
	_, err := d.Pool.Exec(ctx, sql)
	return err
}

// WithAdvisoryLock executes the given function within a PostgreSQL advisory lock.
func (d *DB) WithAdvisoryLock(ctx context.Context, fn func(context.Context) error) error {
	// session-level lock using a dedicated connection
	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", d.LockKey); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", d.LockKey)
	}()
	return fn(ctx)
}

// MigrationStatus represents the status of a migration.
type MigrationStatus string

const (
	// StatusApplied indicates the migration has been successfully applied.
	StatusApplied MigrationStatus = "applied"
	// StatusApplying indicates the migration is currently being applied.
	StatusApplying MigrationStatus = "applying"
	// StatusFailed indicates the migration attempt failed.
	StatusFailed MigrationStatus = "failed"
)

// Record represents a migration record in the database.
type Record struct {
	Version     int64
	Name        string
	Checksum    string
	Status      MigrationStatus
	AppliedAt   *time.Time
	UpdatedAt   time.Time
	ExecutionMs int64
	ErrorText   *string
}
