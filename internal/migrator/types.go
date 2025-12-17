package migrator

import "context"

// Направление миграции
type Direction int

const (
	Up Direction = iota
	Down
)

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
