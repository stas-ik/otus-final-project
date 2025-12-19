// Package migrator provides tools for running SQL and Go migrations.
package migrator

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GoStep represents a single Go-based migration step.
type GoStep struct {
	Version int64
	Name    string
	Up      func(pgx.Tx) error
	Down    func(pgx.Tx) error
}

// Registry stores registered Go migrations.
type Registry struct {
	byVersion map[int64]GoStep
}

// NewRegistry creates a new Registry instance.
func NewRegistry() *Registry { return &Registry{byVersion: map[int64]GoStep{}} }

// Register adds a new Go migration to the registry.
func (r *Registry) Register(ver int64, name string, up func(pgx.Tx) error, down func(pgx.Tx) error) error {
	if _, exists := r.byVersion[ver]; exists {
		return fmt.Errorf("go migration %d already registered", ver)
	}
	r.byVersion[ver] = GoStep{Version: ver, Name: name, Up: up, Down: down}
	return nil
}

// Steps returns all registered Go migrations.
func (r *Registry) Steps() []GoStep {
	out := make([]GoStep, 0, len(r.byVersion))
	for _, s := range r.byVersion {
		out = append(out, s)
	}
	// sorting is performed in the runner
	return out
}
