package migrator

import (
	"context"
	"fmt"

	icfg "migrator/internal/config"
	ipg "migrator/internal/driver/postgres"
	im "migrator/internal/migrator"
)

// RunUp применяет все ожидающие миграции согласно конфигурации.
func RunUp(ctx context.Context, c icfg.Config) error {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return err
	}
	defer db.Close()
	r := im.NewRunner(db)
	if c.Kind == "sql" {
		steps, err := im.ParseSQLDir(c.Path)
		if err != nil {
			return err
		}
		return r.Up(ctx, steps)
	} else if c.Kind == "go" {
		return r.UpGo(ctx, goReg.Steps())
	}
	return fmt.Errorf("unknown kind: %s", c.Kind)
}

func RunDown(ctx context.Context, c icfg.Config) error {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return err
	}
	defer db.Close()
	r := im.NewRunner(db)
	if c.Kind == "sql" {
		steps, err := im.ParseSQLDir(c.Path)
		if err != nil {
			return err
		}
		return r.Down(ctx, steps)
	} else if c.Kind == "go" {
		return r.DownGo(ctx, goReg.Steps())
	}
	return fmt.Errorf("unknown kind: %s", c.Kind)
}

func RunRedo(ctx context.Context, c icfg.Config) error {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return err
	}
	defer db.Close()
	r := im.NewRunner(db)
	if c.Kind == "sql" {
		steps, err := im.ParseSQLDir(c.Path)
		if err != nil {
			return err
		}
		return r.Redo(ctx, steps)
	} else if c.Kind == "go" {
		if err := r.DownGo(ctx, goReg.Steps()); err != nil {
			return err
		}
		return r.UpGo(ctx, goReg.Steps())
	}
	return fmt.Errorf("unknown kind: %s", c.Kind)
}

func Status(ctx context.Context, c icfg.Config) ([]im.StatusRow, error) {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	r := im.NewRunner(db)
	return r.Status(ctx)
}

func DBVersion(ctx context.Context, c icfg.Config) (int64, error) {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	r := im.NewRunner(db)
	return r.DBVersion(ctx)
}
