package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/auth"
	"github.com/ashrocket/bbcli/internal/output"
)

// newAuthStatusCmd creates the `auth status` subcommand.
func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long: `Show the current authentication source, resolved workspace,
and repository detection results. No API calls are made.`,
		RunE: runAuthStatus,
	}

	return cmd
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	client := resolvedClient

	// Build the status rows from the resolved client and auth trace.
	var rows []authStatusRow

	if client != nil {
		rows = append(rows,
			authStatusRow{Key: "Workspace", Value: valueOrNone(client.Workspace())},
			authStatusRow{Key: "Repository", Value: valueOrNone(client.Repo())},
			authStatusRow{Key: "Auth Source", Value: client.Auth().Source()},
			authStatusRow{Key: "Auth Level", Value: fmt.Sprintf("%d", client.Auth().Level())},
		)
	} else {
		rows = append(rows,
			authStatusRow{Key: "Workspace", Value: valueOrNone(flagWorkspace)},
			authStatusRow{Key: "Repository", Value: valueOrNone(flagRepo)},
			authStatusRow{Key: "Auth Source", Value: "(none)"},
		)
	}

	// Show the auth chain trace.
	trace := auth.LastTrace()
	if len(trace) > 0 {
		rows = append(rows, authStatusRow{Key: "", Value: ""})
		rows = append(rows, authStatusRow{Key: "Auth Chain", Value: ""})
		for _, entry := range trace {
			status := "not found"
			if entry.Found {
				status = "found"
			}
			if entry.Err != nil {
				status = fmt.Sprintf("error: %v", entry.Err)
			}
			rows = append(rows, authStatusRow{
				Key:   fmt.Sprintf("  %d. %s", int(entry.Level), entry.Label),
				Value: status,
			})
		}
	}

	result := &authStatusResult{rows: rows}
	return output.Format(os.Stdout, result, outputMode())
}

func valueOrNone(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(not detected)"
	}
	return v
}

// authStatusRow is a key/value pair for status display.
type authStatusRow struct {
	Key   string
	Value string
}

// authStatusResult implements output.Result for auth status.
type authStatusResult struct {
	rows []authStatusRow
}

func (r *authStatusResult) Headers() []string {
	return []string{"Property", "Value"}
}

func (r *authStatusResult) Rows() [][]string {
	rows := make([][]string, len(r.rows))
	for i, row := range r.rows {
		rows[i] = []string{row.Key, row.Value}
	}
	return rows
}

func (r *authStatusResult) MinimalLines() []string {
	lines := make([]string, 0, len(r.rows))
	for _, row := range r.rows {
		if row.Key == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", row.Key, row.Value))
	}
	return lines
}
