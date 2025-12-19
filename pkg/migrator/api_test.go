package migrator

import (
	"context"
	"testing"

	icfg "migrator/internal/config"
)

func TestPublicAPI_InvalidDSN(t *testing.T) {
	ctx := context.Background()
	cfg := icfg.Config{
		DSN: "invalid-dsn",
	}

	t.Run("RunUp", func(t *testing.T) {
		err := RunUp(ctx, cfg)
		if err == nil {
			t.Error("expected error with invalid DSN, got nil")
		}
	})

	t.Run("RunDown", func(t *testing.T) {
		err := RunDown(ctx, cfg)
		if err == nil {
			t.Error("expected error with invalid DSN, got nil")
		}
	})

	t.Run("RunRedo", func(t *testing.T) {
		err := RunRedo(ctx, cfg)
		if err == nil {
			t.Error("expected error with invalid DSN, got nil")
		}
	})

	t.Run("Status", func(t *testing.T) {
		_, err := Status(ctx, cfg)
		if err == nil {
			t.Error("expected error with invalid DSN, got nil")
		}
	})

	t.Run("DBVersion", func(t *testing.T) {
		_, err := DBVersion(ctx, cfg)
		if err == nil {
			t.Error("expected error with invalid DSN, got nil")
		}
	})
}

func TestPublicAPI_UnknownKind(_ *testing.T) {
	// Мы не можем легко протестировать успех без реальной БД или сложных моков,
	// но можем протестировать проверку типа миграции (kind), если DSN корректен по формату,
	// но подключение не требуется немедленно (хотя Connect в api.go вызывается сразу).
	// Однако, Connect вызывает pgxpool.Connect, который может попытаться соединиться.
}
