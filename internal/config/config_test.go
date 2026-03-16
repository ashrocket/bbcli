package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ashrocket/bbcli/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	if cfg.Defaults.Output != "table" {
		t.Errorf("Output = %q, want table", cfg.Defaults.Output)
	}
	if cfg.Defaults.Workspace != "" {
		t.Errorf("Workspace = %q, want empty", cfg.Defaults.Workspace)
	}
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if cfg.Defaults.Output != "table" {
		t.Errorf("should return defaults when file missing")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &config.Config{
		Defaults: config.Defaults{
			Output:    "json",
			Workspace: "kureapp",
		},
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Defaults.Output != "json" {
		t.Errorf("Output = %q, want json", loaded.Defaults.Output)
	}
	if loaded.Defaults.Workspace != "kureapp" {
		t.Errorf("Workspace = %q, want kureapp", loaded.Defaults.Workspace)
	}
}

func TestConfigDir(t *testing.T) {
	// Default path
	dir := config.Dir()
	if dir == "" {
		t.Error("Dir() should not be empty")
	}

	// Override via env var
	t.Setenv("BBCLI_CONFIG_DIR", "/custom/path")
	dir = config.Dir()
	if dir != "/custom/path" {
		t.Errorf("Dir() = %q, want /custom/path", dir)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	cfg := config.Default()
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save should create parent dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist after Save: %v", err)
	}
}
