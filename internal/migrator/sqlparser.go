package migrator

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ParseSQLDir сканирует каталог на наличие файлов *.sql формата: <version>_<name>.sql
// и разделяет содержимое по маркерам: `-- +migrate Up` и `-- +migrate Down`.
func ParseSQLDir(dir string) ([]Step, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	steps := make([]Step, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		ver, title, ok := splitVersionName(name)
		if !ok {
			continue
		}
		full := filepath.Join(dir, name)
		up, down, err := splitUpDown(full)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		sum := checksum(up + "\n--DOWN--\n" + down)
		steps = append(steps, Step{Version: ver, Name: title, UpSQL: up, DownSQL: down, Checksum: sum})
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].Version < steps[j].Version })
	return steps, nil
}

func splitVersionName(filename string) (int64, string, bool) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	idx := strings.IndexByte(base, '_')
	if idx <= 0 {
		return 0, "", false
	}
	v, err := strconv.ParseInt(base[:idx], 10, 64)
	if err != nil {
		return 0, "", false
	}
	name := base[idx+1:]
	if name == "" {
		name = "migration"
	}
	return v, name, true
}

func splitUpDown(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	var b strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	mode := ""
	var up, down strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		ltrim := strings.TrimSpace(strings.ToLower(line))
		switch ltrim {
		case "-- +migrate up", "--+migrate up":
			mode = "up"
			continue
		case "-- +migrate down", "--+migrate down":
			mode = "down"
			continue
		}
		if mode == "up" {
			up.WriteString(line)
			up.WriteByte('\n')
		} else if mode == "down" {
			down.WriteString(line)
			down.WriteByte('\n')
		} else {
			// игнорировать строки до первого маркера
			b.WriteString("")
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(up.String()), strings.TrimSpace(down.String()), nil
}

func checksum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
