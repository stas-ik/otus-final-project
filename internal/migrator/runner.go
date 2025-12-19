package migrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	pg "migrator/internal/driver/postgres"
)

type Runner struct {
	DB          *pg.DB
	SchemaTable string
}

func NewRunner(db *pg.DB) *Runner { return &Runner{DB: db, SchemaTable: db.SchemaTable} }

// Up применяет все ожидающие SQL-миграции, найденные в каталоге.
func (r *Runner) Up(ctx context.Context, steps []Step) error {
	return r.DB.WithAdvisoryLock(ctx, func(ctx context.Context) error {
		applied, err := r.loadApplied(ctx)
		if err != nil {
			return err
		}
		// отфильтровать ожидающие
		pending := make([]Step, 0)
		for _, s := range steps {
			if _, ok := applied[s.Version]; !ok {
				pending = append(pending, s)
			}
		}
		sort.Slice(pending, func(i, j int) bool { return pending[i].Version < pending[j].Version })
		for _, s := range pending {
			if err := r.applyOne(ctx, s, true); err != nil {
				return err
			}
		}
		return nil
	})
}

// Down откатывает последнюю применённую миграцию.
func (r *Runner) Down(ctx context.Context, steps []Step) error {
	return r.DB.WithAdvisoryLock(ctx, func(ctx context.Context) error {
		applied, err := r.loadApplied(ctx)
		if err != nil {
			return err
		}
		var lastVer int64 = -1
		for v := range applied {
			if v > lastVer {
				lastVer = v
			}
		}
		if lastVer < 0 {
			return nil
		}
		// найти шаг по версии
		var last Step
		found := false
		for _, s := range steps {
			if s.Version == lastVer {
				last = s
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("cannot find migration %d to rollback", lastVer)
		}
		return r.applyOne(ctx, last, false)
	})
}

func (r *Runner) Redo(ctx context.Context, steps []Step) error {
	return r.DB.WithAdvisoryLock(ctx, func(ctx context.Context) error {
		if err := r.Down(ctx, steps); err != nil {
			return err
		}
		return r.Up(ctx, steps)
	})
}

// Go‑миграции
func (r *Runner) UpGo(ctx context.Context, steps []GoStep) error {
	return r.DB.WithAdvisoryLock(ctx, func(ctx context.Context) error {
		applied, err := r.loadApplied(ctx)
		if err != nil {
			return err
		}
		sort.Slice(steps, func(i, j int) bool { return steps[i].Version < steps[j].Version })
		for _, s := range steps {
			if _, ok := applied[s.Version]; ok {
				continue
			}
			if err := r.applyGo(ctx, s, true); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Runner) DownGo(ctx context.Context, steps []GoStep) error {
	return r.DB.WithAdvisoryLock(ctx, func(ctx context.Context) error {
		applied, err := r.loadApplied(ctx)
		if err != nil {
			return err
		}
		var lastVer int64 = -1
		for v := range applied {
			if v > lastVer {
				lastVer = v
			}
		}
		if lastVer < 0 {
			return nil
		}
		sort.Slice(steps, func(i, j int) bool { return steps[i].Version < steps[j].Version })
		for _, s := range steps {
			if s.Version == lastVer {
				return r.applyGo(ctx, s, false)
			}
		}
		return fmt.Errorf("cannot find go migration %d to rollback", lastVer)
	})
}

func (r *Runner) applyGo(ctx context.Context, s GoStep, up bool) error {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	started := time.Now()
	if up {
		if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s(version,name,checksum,status,updated_at) VALUES($1,$2,$3,'applying',now())", r.SchemaTable), s.Version, s.Name, "go://checksum"); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := s.Up(tx); err != nil {
			_, _ = tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='failed', updated_at=now(), error_text=$2 WHERE version=$1", r.SchemaTable), s.Version, err.Error())
			_ = tx.Rollback(ctx)
			return fmt.Errorf("up %d_%s failed: %w", s.Version, s.Name, err)
		}
		dur := time.Since(started)
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='applied', applied_at=now(), updated_at=now(), execution_ms=$2, error_text=NULL WHERE version=$1", r.SchemaTable), s.Version, dur.Milliseconds()); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='applying', updated_at=now() WHERE version=$1", r.SchemaTable), s.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if s.Down != nil {
			if err := s.Down(tx); err != nil {
				_, _ = tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='failed', updated_at=now(), error_text=$2 WHERE version=$1", r.SchemaTable), s.Version, err.Error())
				_ = tx.Rollback(ctx)
				return fmt.Errorf("down %d_%s failed: %w", s.Version, s.Name, err)
			}
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE version=$1", r.SchemaTable), s.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Runner) DBVersion(ctx context.Context) (int64, error) {
	rows, err := r.DB.Pool.Query(ctx, fmt.Sprintf("SELECT version FROM %s WHERE status='applied' ORDER BY version DESC LIMIT 1", r.SchemaTable))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		return v, nil
	}
	return 0, nil
}

type StatusRow struct {
	Version   int64
	Name      string
	Status    string
	UpdatedAt time.Time
}

func (r *Runner) Status(ctx context.Context) ([]StatusRow, error) {
	q := fmt.Sprintf("SELECT version,name,status,updated_at FROM %s ORDER BY version", r.SchemaTable)
	rows, err := r.DB.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []StatusRow{}
	for rows.Next() {
		var s StatusRow
		if err := rows.Scan(&s.Version, &s.Name, &s.Status, &s.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *Runner) loadApplied(ctx context.Context) (map[int64]struct{}, error) {
	rows, err := r.DB.Pool.Query(ctx, fmt.Sprintf("SELECT version FROM %s WHERE status='applied'", r.SchemaTable))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[int64]struct{}{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		m[v] = struct{}{}
	}
	return m, rows.Err()
}

func (r *Runner) applyOne(ctx context.Context, s Step, up bool) error {
	sql := s.UpSQL
	action := "up"
	if !up {
		sql = s.DownSQL
		action = "down"
	}
	if strings.TrimSpace(sql) == "" {
		return nil
	}
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	started := time.Now()
	// пометить как выполняемую
	if up {
		if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s(version,name,checksum,status,updated_at) VALUES($1,$2,$3,'applying',now())", r.SchemaTable), s.Version, s.Name, s.Checksum); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='applying', updated_at=now() WHERE version=$1", r.SchemaTable), s.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}
	if _, err := tx.Exec(ctx, sql); err != nil {
		_, _ = tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='failed', updated_at=now(), error_text=$2 WHERE version=$1", r.SchemaTable), s.Version, err.Error())
		_ = tx.Rollback(ctx)
		return fmt.Errorf("%s %d_%s failed: %w", action, s.Version, s.Name, err)
	}
	dur := time.Since(started)
	if up {
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET status='applied', applied_at=now(), updated_at=now(), execution_ms=$2, error_text=NULL WHERE version=$1", r.SchemaTable), s.Version, dur.Milliseconds()); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE version=$1", r.SchemaTable), s.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}
	return tx.Commit(ctx)
}
