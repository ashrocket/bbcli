// Package output formats command results into table, JSON, or minimal output.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/ashrocket/bbcli/internal/errors"
)

// Result is the interface that command results must implement to be formatted.
// Each command builds a concrete type satisfying Result, then passes it to Format.
type Result interface {
	// Headers returns column names for table output.
	Headers() []string
	// Rows returns all data rows. Each row is a slice of cell values
	// aligned positionally with Headers.
	Rows() [][]string
	// MinimalLines returns one value per result entry for minimal output
	// (e.g., just PR IDs).
	MinimalLines() []string
}

// Format writes the result to w using the given mode ("json", "table", or "minimal").
// Returns a *errors.CLIError with ExitUsage for unknown modes.
func Format(w io.Writer, r Result, mode string) error {
	switch mode {
	case "json":
		return formatJSON(w, r)
	case "table":
		return formatTable(w, r)
	case "minimal":
		return formatMinimal(w, r)
	default:
		return errors.NewUsageError(fmt.Sprintf("unknown output format: %q", mode))
	}
}

// formatJSON marshals rows as an array of objects keyed by header names.
func formatJSON(w io.Writer, r Result) error {
	headers := r.Headers()
	rows := r.Rows()

	items := make([]map[string]string, len(rows))
	for i, row := range rows {
		obj := make(map[string]string, len(headers))
		for j, h := range headers {
			if j < len(row) {
				obj[h] = row[j]
			}
		}
		items[i] = obj
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

// formatTable writes aligned columns using tabwriter. Plain text only --
// ANSI escape codes must NOT be used inside tabwriter because they break
// column alignment.
func formatTable(w io.Writer, r Result) error {
	rows := r.Rows()
	if len(rows) == 0 {
		return nil
	}

	headers := r.Headers()

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header line -- plain text, no ANSI
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	// Data rows
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}

// formatMinimal writes one value per line with no headers.
func formatMinimal(w io.Writer, r Result) error {
	for _, line := range r.MinimalLines() {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
