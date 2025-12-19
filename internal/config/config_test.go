package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func TestDefault(t *testing.T) {
	def := Default()
	if def.Path != "./migrations" {
		t.Errorf("expected path ./migrations, got %s", def.Path)
	}
	if def.Kind != "sql" {
		t.Errorf("expected kind sql, got %s", def.Kind)
	}
	if def.LockKey != 7243392 {
		t.Errorf("expected lock key 7243392, got %d", def.LockKey)
	}
	if def.SchemaTable != "schema_migrations" {
		t.Errorf("expected schema table schema_migrations, got %s", def.SchemaTable)
	}
}

func TestLoad(t *testing.T) {
	t.Run("minimal dsn via env", func(t *testing.T) {
		os.Setenv("GOMIGRATOR_DSN", "postgres://user:pass@localhost:5432/db")
		defer os.Unsetenv("GOMIGRATOR_DSN")

		c, err := Load(nil, "")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if c.DSN != "postgres://user:pass@localhost:5432/db" {
			t.Errorf("unexpected DSN: %s", c.DSN)
		}
	})

	t.Run("missing dsn error", func(t *testing.T) {
		_, err := Load(nil, "/non/existent/config.yaml")
		if err == nil {
			t.Fatal("expected error due to missing DSN and missing file, got nil")
		}
	})

	t.Run("from config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")
		content := `
dsn: "postgres://file:5432/db"
path: "./custom_migrations"
kind: "go"
`
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write tmp config: %v", err)
		}

		c, err := Load(nil, cfgPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if c.DSN != "postgres://file:5432/db" {
			t.Errorf("unexpected DSN: %s", c.DSN)
		}
		if c.Kind != "go" {
			t.Errorf("unexpected kind: %s", c.Kind)
		}
	})

	t.Run("with flags", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("dsn", "", "")
		fs.String("kind", "", "")

		err := fs.Parse([]string{"--dsn", "postgres://flag:5432/db", "--kind", "sql"})
		if err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		c, err := Load(fs, "")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if c.DSN != "postgres://flag:5432/db" {
			t.Errorf("unexpected DSN: %s", c.DSN)
		}
		if c.Kind != "sql" {
			t.Errorf("unexpected kind: %s", c.Kind)
		}
	})
}
