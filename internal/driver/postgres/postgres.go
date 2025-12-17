package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool        *pgxpool.Pool
	SchemaTable string
	LockKey     int64
}

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

func (d *DB) WithAdvisoryLock(ctx context.Context, fn func(context.Context) error) error {
	// блокировка на уровне сессии с использованием выделенного соединения
	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", d.LockKey); err != nil {
		return err
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", d.LockKey)
	return fn(ctx)
}

type MigrationStatus string

const (
	StatusApplied  MigrationStatus = "applied"
	StatusApplying MigrationStatus = "applying"
	StatusFailed   MigrationStatus = "failed"
)

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
