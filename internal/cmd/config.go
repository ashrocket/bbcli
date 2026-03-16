package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/config"
	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// validConfigKeys lists the config keys that can be set.
var validConfigKeys = map[string]bool{
	"defaults.output":    true,
	"defaults.workspace": true,
}

// newConfigCmd creates the `config` parent command with set/get/list subcommands.
func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Get, set, and list bbcli configuration values.",
		// Override PersistentPreRunE — config commands work without
		// authentication since they only manage local files.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigListCmd())

	return configCmd
}

// newConfigSetCmd creates the `config set` subcommand.
func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Valid keys:
  defaults.output     Output format (table, json, minimal)
  defaults.workspace  Default Bitbucket workspace slug`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

// newConfigGetCmd creates the `config get` subcommand.
func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value by key.

Valid keys:
  defaults.output     Output format (table, json, minimal)
  defaults.workspace  Default Bitbucket workspace slug`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}
}

// newConfigListCmd creates the `config list` subcommand.
func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all configuration values",
		Long:    "Show all current configuration values and the config file path.",
		Aliases: []string{"ls"},
		RunE:    runConfigList,
	}
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if !validConfigKeys[key] {
		return errors.NewUsageError(fmt.Sprintf(
			"unknown config key %q; valid keys: %s",
			key, strings.Join(configKeyList(), ", ")))
	}

	// Validate output format values.
	if key == "defaults.output" {
		validFormats := map[string]bool{"table": true, "json": true, "minimal": true}
		if !validFormats[value] {
			return errors.NewUsageError(fmt.Sprintf(
				"invalid output format %q; must be table, json, or minimal", value))
		}
	}

	cfgPath := config.Path()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	switch key {
	case "defaults.output":
		cfg.Defaults.Output = value
	case "defaults.workspace":
		cfg.Defaults.Workspace = value
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to save config: %v", err))
	}

	fmt.Fprintf(os.Stdout, "%s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	if !validConfigKeys[key] {
		return errors.NewUsageError(fmt.Sprintf(
			"unknown config key %q; valid keys: %s",
			key, strings.Join(configKeyList(), ", ")))
	}

	cfgPath := config.Path()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	var value string
	switch key {
	case "defaults.output":
		value = cfg.Defaults.Output
	case "defaults.workspace":
		value = cfg.Defaults.Workspace
	}

	fmt.Fprintln(os.Stdout, value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfgPath := config.Path()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}

	rows := []configRow{
		{Key: "defaults.output", Value: cfg.Defaults.Output},
		{Key: "defaults.workspace", Value: cfg.Defaults.Workspace},
		{Key: "", Value: ""},
		{Key: "config file", Value: cfgPath},
	}

	result := &configListResult{rows: rows}
	return output.Format(os.Stdout, result, outputMode())
}

func configKeyList() []string {
	keys := make([]string, 0, len(validConfigKeys))
	for k := range validConfigKeys {
		keys = append(keys, k)
	}
	return keys
}

// configRow is a key/value pair for config display.
type configRow struct {
	Key   string
	Value string
}

// configListResult implements output.Result for config list.
type configListResult struct {
	rows []configRow
}

func (r *configListResult) Headers() []string {
	return []string{"Key", "Value"}
}

func (r *configListResult) Rows() [][]string {
	rows := make([][]string, len(r.rows))
	for i, row := range r.rows {
		rows[i] = []string{row.Key, row.Value}
	}
	return rows
}

func (r *configListResult) MinimalLines() []string {
	lines := make([]string, 0, len(r.rows))
	for _, row := range r.rows {
		if row.Key == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", row.Key, row.Value))
	}
	return lines
}
