package main

import (
	"context"
	"fmt"
	"os"
	"time"

	cfg "migrator/internal/config"
	pub "migrator/pkg/migrator"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	cfgFile string
)

func main() {
	root := &cobra.Command{
		Use:   "gomigrator",
		Short: "Database migration tool for PostgreSQL (SQL & Go)",
	}

	flags := root.PersistentFlags()
	addCommonFlags(flags)
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to config YAML")

	root.AddCommand(cmdCreate(flags), cmdUp(flags), cmdDown(flags), cmdRedo(flags), cmdStatus(flags), cmdDBVersion(flags))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addCommonFlags(fs *pflag.FlagSet) {
	fs.String("dsn", "", "PostgreSQL DSN")
	fs.String("path", "./migrations", "Path to migrations directory")
	fs.String("kind", "sql", "Migration kind: sql|go")
	fs.Int64("lock_key", 7243392, "Advisory lock key")
	fs.String("schema_table", "schema_migrations", "Schema table name")
}

func loadConfig(flags *pflag.FlagSet) (cfg.Config, error) {
	return cfg.Load(flags, cfgFile)
}

func cmdCreate(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := loadConfig(flags)
			if err != nil {
				return err
			}
			name := args[0]
			var path string
			if c.Kind == "go" {
				path, err = createGoTemplate(c.Path, name)
			} else {
				path, err = createSQLTemplate(c.Path, name)
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
			return nil
		},
	}
}

func cmdUp(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{Use: "up", Short: "Apply all pending migrations", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig(flags)
		if err != nil {
			return err
		}
		return pub.RunUp(context.Background(), c)
	}}
}

func cmdDown(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{Use: "down", Short: "Rollback the last migration", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig(flags)
		if err != nil {
			return err
		}
		return pub.RunDown(context.Background(), c)
	}}
}

func cmdRedo(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{Use: "redo", Short: "Redo the last migration (down+up)", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig(flags)
		if err != nil {
			return err
		}
		return pub.RunRedo(context.Background(), c)
	}}
}

func cmdStatus(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Show migration status table", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig(flags)
		if err != nil {
			return err
		}
		rows, err := pub.Status(context.Background(), c)
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		fmt.Fprintln(w, "STATUS\tUPDATED_AT\tVERSION\tNAME")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", r.Status, r.UpdatedAt.Format(time.RFC3339), r.Version, r.Name)
		}
		return nil
	}}
}

func cmdDBVersion(flags *pflag.FlagSet) *cobra.Command {
	return &cobra.Command{Use: "dbversion", Short: "Print the last applied version", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig(flags)
		if err != nil {
			return err
		}
		v, err := pub.DBVersion(context.Background(), c)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), v)
		return nil
	}}
}

// createSQLTemplate создаёт файл SQL‑миграции с разделителями Up/Down.
func createSQLTemplate(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ts := time.Now().UnixMilli()
	file := fmt.Sprintf("%d_%s.sql", ts, sanitizeName(name))
	full := fmt.Sprintf("%s%c%s", dir, os.PathSeparator, file)
	content := "-- +migrate Up\n\n\n-- +migrate Down\n"
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", err
	}
	return full, nil
}

func sanitizeName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			out = append(out, r)
		} else if r == ' ' || r == '.' || r == '/' || r == '\\' {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "migration"
	}
	return string(out)
}

// без дополнительной прослойки времени

// createGoTemplate создаёт шаблон Go‑миграции и регистрирует функции.
func createGoTemplate(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ts := time.Now().UnixMilli()
	base := fmt.Sprintf("%d_%s.go", ts, sanitizeName(name))
	full := fmt.Sprintf("%s%c%s", dir, os.PathSeparator, base)
	pkg := "migrations"
	content := fmt.Sprintf(`package %s

import (
    "context"
    "github.com/jackc/pgx/v5"
    lib "migrator/pkg/migrator"
)

func init() {
    // Зарегистрировать Go‑миграцию %d_%s
    _ = lib.Register(%d, "%s", Up_%d_%s, Down_%d_%s)
}

func Up_%d_%s(tx pgx.Tx) error {
    // TODO: напишите здесь логику применения (up) миграции
    // Пример:
    // _, err := tx.Exec(context.Background(), "SELECT 1")
    _, _ = tx.Exec(context.Background(), "SELECT 1")
    return nil
}

func Down_%d_%s(tx pgx.Tx) error {
    // TODO: напишите здесь логику отката (down) миграции
    _, _ = tx.Exec(context.Background(), "SELECT 1")
    return nil
}
`, pkg, ts, sanitizeName(name), ts, sanitizeName(name), ts, sanitizeName(name), ts, sanitizeName(name), ts, sanitizeName(name), ts, sanitizeName(name))
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", err
	}
	return full, nil
}
