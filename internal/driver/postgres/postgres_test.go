package postgres

import (
	"context"
	"testing"
)

func TestConnect_Error(t *testing.T) {
	ctx := context.Background()
	_, err := Connect(ctx, "invalid-dsn", "test_table", 12345)
	if err == nil {
		t.Error("expected error with invalid DSN, got nil")
	}
}

func TestMigrationStatus(t *testing.T) {
	if StatusApplied != "applied" {
		t.Errorf("expected applied, got %s", StatusApplied)
	}
	if StatusApplying != "applying" {
		t.Errorf("expected applying, got %s", StatusApplying)
	}
	if StatusFailed != "failed" {
		t.Errorf("expected failed, got %s", StatusFailed)
	}
}
