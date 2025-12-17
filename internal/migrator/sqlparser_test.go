package migrator

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_splitVersionName(t *testing.T) {
	v, name, ok := splitVersionName("1700000000000_init.sql")
	if !ok || v != 1700000000000 || name != "init" {
		t.Fatalf("unexpected: %v %v %v", v, name, ok)
	}
	if _, _, ok := splitVersionName("badname.sql"); ok {
		t.Fatalf("expected not ok")
	}
}

func Test_splitUpDown(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "1_test.sql")
	content := `-- +migrate Up
CREATE TABLE x(id int);

-- +migrate Down
DROP TABLE x;`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	up, down, err := splitUpDown(file)
	if err != nil {
		t.Fatal(err)
	}
	if up == "" || down == "" {
		t.Fatalf("expected both parts, got up=%q down=%q", up, down)
	}
}
