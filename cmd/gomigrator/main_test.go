package main

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello_world"},
		{"MyMigration_01", "MyMigration_01"},
		{"test/file\\name", "test_file_name"},
		{"!@#$%^", "migration"},
		{"", "migration"},
		{"123-abc", "123-abc"},
	}

	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeName(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCommandSetup(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	addCommonFlags(fs)

	// Just verify commands can be created without panic
	t.Run("CreateUp", func(_ *testing.T) { _ = cmdUp(fs) })
	t.Run("CreateDown", func(_ *testing.T) { _ = cmdDown(fs) })
	t.Run("CreateRedo", func(_ *testing.T) { _ = cmdRedo(fs) })
	t.Run("CreateStatus", func(_ *testing.T) { _ = cmdStatus(fs) })
	t.Run("CreateDBVersion", func(_ *testing.T) { _ = cmdDBVersion(fs) })
	t.Run("CreateCreate", func(_ *testing.T) { _ = cmdCreate(fs) })
}
