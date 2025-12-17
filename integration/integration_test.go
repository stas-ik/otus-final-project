package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	icfg "migrator/internal/config"
	pub "migrator/pkg/migrator"
)

func dsn() string {
	if v := os.Getenv("DB_DSN"); v != "" {
		return v
	}
	return "postgres://test:test@localhost:54329/testdb?sslmode=disable"
}

func Test_SQL_Migrations_EndToEnd(t *testing.T) {
	ctx := context.Background()
	// пробуем подключиться, пропускаем тест, если БД недоступна
	pool, err := pgxpool.New(ctx, dsn())
	if err != nil {
		t.Skipf("pg not available: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("pg not available: %v", err)
	}

	dir := t.TempDir()
	// создать две миграции
	mustWrite(t, filepath.Join(dir, "1000_init.sql"), `-- +migrate Up
CREATE TABLE IF NOT EXISTS foo(id INT PRIMARY KEY);

-- +migrate Down
DROP TABLE IF EXISTS foo;`)

	mustWrite(t, filepath.Join(dir, "2000_seed.sql"), `-- +migrate Up
INSERT INTO foo(id) VALUES (1);

-- +migrate Down
DELETE FROM foo WHERE id=1;`)

	cfg := icfg.Config{DSN: dsn(), Path: dir, Kind: "sql", LockKey: 7243392, SchemaTable: "schema_migrations"}

	// up
	if err := pub.RunUp(ctx, cfg); err != nil {
		t.Fatalf("up failed: %v", err)
	}

	// проверка
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM foo").Scan(&n); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row, got %d", n)
	}

	// версия
	v, err := pub.DBVersion(ctx, cfg)
	if err != nil {
		t.Fatalf("dbversion: %v", err)
	}
	if v != 2000 {
		t.Fatalf("expected version 2000, got %d", v)
	}

	// статус
	rows, err := pub.Status(ctx, cfg)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 status rows, got %d", len(rows))
	}

	// down
	if err := pub.RunDown(ctx, cfg); err != nil {
		t.Fatalf("down failed: %v", err)
	}
	// убеждаемся, что таблица удалена
	if _, err := pool.Exec(ctx, "SELECT 1 FROM foo"); err == nil {
		t.Fatalf("expected error querying dropped table")
	}

	// redo
	if err := pub.RunRedo(ctx, cfg); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM foo").Scan(&n); err != nil {
		t.Fatalf("verify2 failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row after redo, got %d", n)
	}
}

func mustWrite(t *testing.T, path string, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
