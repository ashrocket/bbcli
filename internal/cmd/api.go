package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
)

// API command flags.
var (
	apiMethod string
	apiInput  string
)

// newAPICmd creates the `api` escape-hatch subcommand.
func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api <PATH>",
		Short: "Make an authenticated API request",
		Long: `Make an authenticated request to any Bitbucket API endpoint.

The path supports {workspace} and {repo} placeholders, which are
substituted with the detected or flag-provided values.

Examples:
  bbcli api /repositories/{workspace}/{repo}/refs/branches
  bbcli api -X POST /repositories/{workspace}/{repo}/pullrequests --input body.json
  echo '{"title":"test"}' | bbcli api -X POST /repositories/{workspace}/{repo}/pullrequests --input -`,
		Args: cobra.ExactArgs(1),
		RunE: runAPI,
	}

	cmd.Flags().StringVarP(&apiMethod, "method", "X", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH)")
	cmd.Flags().StringVar(&apiInput, "input", "", "Request body: file path or - for stdin")

	return cmd
}

func runAPI(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	path := args[0]

	// Read request body if provided.
	var body []byte
	if apiInput != "" {
		var reader io.Reader
		if apiInput == "-" {
			reader = os.Stdin
		} else {
			f, err := os.Open(apiInput)
			if err != nil {
				return errors.NewUsageError(fmt.Sprintf("cannot open input file %q: %v", apiInput, err))
			}
			defer f.Close()
			reader = f
		}

		var err error
		body, err = io.ReadAll(reader)
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to read input: %v", err))
		}
	}

	raw, err := client.Request(apiMethod, path, body)
	if err != nil {
		return err
	}

	// Pretty-print the JSON response.
	var pretty json.RawMessage
	if json.Unmarshal(raw, &pretty) == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(pretty)
	}

	// Fallback: write raw bytes.
	_, writeErr := os.Stdout.Write(raw)
	if writeErr != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to write output: %v", writeErr))
	}
	fmt.Fprintln(os.Stdout)
	return nil
}
