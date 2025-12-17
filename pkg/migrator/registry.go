package migrator

import (
	"github.com/jackc/pgx/v5"
	im "migrator/internal/migrator"
)

var goReg = im.NewRegistry()

// Register регистрирует Go‑миграцию с идентификатором <timestamp>_<name>
// Используется в приложениях, которые подключают библиотеку напрямую.
func Register(version int64, name string, up func(pgx.Tx) error, down func(pgx.Tx) error) error {
	return goReg.Register(version, name, up, down)
}
