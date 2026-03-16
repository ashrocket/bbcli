// Package config manages bbcli's configuration file.
// v1 config is deliberately minimal: output format and default workspace.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the bbcli configuration. Stored at ~/.config/bbcli/config.yaml.
type Config struct {
	Defaults Defaults `yaml:"defaults"`
}

// Defaults holds default values for command flags.
type Defaults struct {
	Output    string `yaml:"output"`    // table | json | minimal
	Workspace string `yaml:"workspace"` // default workspace slug
}

// Default returns a config with sensible defaults.
func Default() *Config {
	return &Config{
		Defaults: Defaults{
			Output: "table",
		},
	}
}

// Dir returns the config directory path.
// Respects BBCLI_CONFIG_DIR env var, falls back to ~/.config/bbcli.
func Dir() string {
	if dir := os.Getenv("BBCLI_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".bbcli")
	}
	return filepath.Join(home, ".config", "bbcli")
}

// Path returns the full path to the config file.
func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

// Load reads the config from a file. Returns defaults if the file doesn't exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to a file, creating parent directories if needed.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
