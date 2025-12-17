package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config содержит конфигурацию приложения
type Config struct {
	DSN         string `mapstructure:"dsn"`
	Path        string `mapstructure:"path"`
	Kind        string `mapstructure:"kind"` // sql|go
	LockKey     int64  `mapstructure:"lock_key"`
	SchemaTable string `mapstructure:"schema_table"`
}

func Default() Config {
	return Config{
		Path:        "./migrations",
		Kind:        "sql",
		LockKey:     7243392,
		SchemaTable: "schema_migrations",
	}
}

// Load загружает конфигурацию из файла + переменных окружения + флагов
func Load(flags *pflag.FlagSet, configFile string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("GOMIGRATOR")
	v.AutomaticEnv()

	def := Default()
	_ = v.MergeConfigMap(map[string]any{
		"dsn":          def.DSN,
		"path":         def.Path,
		"kind":         def.Kind,
		"lock_key":     def.LockKey,
		"schema_table": def.SchemaTable,
	})

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := readAndExpandFile(v, configFile); err != nil {
			return Config{}, err
		}
	} else {
		// искать конфиг по умолчанию
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if err := tryReadAndExpand(v); err != nil {
			return Config{}, err
		}
	}

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return Config{}, err
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return Config{}, err
	}
	if c.DSN == "" {
		return Config{}, fmt.Errorf("dsn is required (env GOMIGRATOR_DSN or config dsn)")
	}
	if c.Path == "" {
		c.Path = def.Path
	}
	if !filepath.IsAbs(c.Path) {
		if p, err := filepath.Abs(c.Path); err == nil {
			c.Path = p
		}
	}
	c.Kind = strings.ToLower(strings.TrimSpace(c.Kind))
	if c.Kind != "sql" && c.Kind != "go" {
		c.Kind = def.Kind
	}
	if c.SchemaTable == "" {
		c.SchemaTable = def.SchemaTable
	}
	if c.LockKey == 0 {
		c.LockKey = def.LockKey
	}
	return c, nil
}

func readAndExpandFile(v *viper.Viper, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	expanded := os.ExpandEnv(string(b))
	return v.MergeConfig(strings.NewReader(expanded))
}

func tryReadAndExpand(v *viper.Viper) error {
	// необязательно
	if err := v.ReadInConfig(); err != nil {
		// игнорировать отсутствие файла
		var viperConfigFileNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &viperConfigFileNotFound) {
			return nil
		}
		// попытаться вручную подставить переменные, если чтение не удалось по другой причине
		// но путь неизвестен, поэтому вернуть ошибку
		return nil
	}
	// Если файл конфигурации найден, повторно прочитать его с подстановкой окружения
	path := v.ConfigFileUsed()
	if path == "" {
		return nil
	}
	return readAndExpandFile(v, path)
}
