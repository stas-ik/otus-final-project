// Package migrator provides the public API for running migrations.
package migrator

import (
	"context"
	"fmt"

	icfg "migrator/internal/config"
	ipg "migrator/internal/driver/postgres"
	im "migrator/internal/migrator"
)

// RunUp applies all pending migrations according to the configuration.
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

// RunDown rolls back the last applied migration.
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

// RunRedo rolls back and then reapplies the last migration.
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

// Status returns the migration status for all migrations.
func Status(ctx context.Context, c icfg.Config) ([]im.StatusRow, error) {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	r := im.NewRunner(db)
	return r.Status(ctx)
}

// DBVersion returns the current database migration version.
func DBVersion(ctx context.Context, c icfg.Config) (int64, error) {
	db, err := ipg.Connect(ctx, c.DSN, c.SchemaTable, c.LockKey)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	r := im.NewRunner(db)
	return r.DBVersion(ctx)
}
