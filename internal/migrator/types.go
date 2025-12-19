package migrator

import "context"

// Direction represents the direction of a migration (Up or Down).
type Direction int

const (
	// Up represents a forward migration.
	Up Direction = iota
	// Down represents a rollback migration.
	Down
)

// Step represents a single SQL migration step.
type Step struct {
	Version  int64
	Name     string
	UpSQL    string
	DownSQL  string
	Checksum string
}

// Driver абстрагирует операции БД, используемые мигратором
type Driver interface {
	WithAdvisoryLock(ctx context.Context, fn func(context.Context) error) error
}
