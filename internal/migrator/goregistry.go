package migrator

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

type GoStep struct {
	Version int64
	Name    string
	Up      func(pgx.Tx) error
	Down    func(pgx.Tx) error
}

type Registry struct {
	byVersion map[int64]GoStep
}

func NewRegistry() *Registry { return &Registry{byVersion: map[int64]GoStep{}} }

func (r *Registry) Register(ver int64, name string, up func(pgx.Tx) error, down func(pgx.Tx) error) error {
	if _, exists := r.byVersion[ver]; exists {
		return fmt.Errorf("go migration %d already registered", ver)
	}
	r.byVersion[ver] = GoStep{Version: ver, Name: name, Up: up, Down: down}
	return nil
}

func (r *Registry) Steps() []GoStep {
	out := make([]GoStep, 0, len(r.byVersion))
	for _, s := range r.byVersion {
		out = append(out, s)
	}
	// упорядочивание выполняется в runner
	return out
}
