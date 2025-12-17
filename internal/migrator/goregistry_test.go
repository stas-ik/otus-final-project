package migrator

import (
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestRegistry_RegisterAndSteps(t *testing.T) {
	r := NewRegistry()
	up := func(pgx.Tx) error { return nil }
	down := func(pgx.Tx) error { return nil }
	if err := r.Register(1, "one", up, down); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := r.Register(2, "two", up, nil); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := r.Register(1, "dup", up, down); err == nil {
		t.Fatalf("expected duplicate error")
	}
	steps := r.Steps()
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}
