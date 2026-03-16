// Package cmd defines the Cobra command tree for bbcli.
package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/ashrocket/bbcli/internal/auth"
	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/git"
)

// Global flag values, accessible to all subcommands.
var (
	flagOutput    string
	flagJSON      bool
	flagWorkspace string
	flagRepo      string
	flagDebug     bool
	flagNoRetry   bool
	flagNoColor   bool
	flagToken     string
)

// resolvedClient is set during PersistentPreRunE so subcommands can use it.
var resolvedClient *api.Client

// outputMode returns the effective output format, resolving --json shorthand.
func outputMode() string {
	if flagJSON {
		return "json"
	}
	return flagOutput
}

// authAdapter bridges auth.Result to the api.Authenticator interface.
type authAdapter struct {
	result *auth.Result
}

func (a *authAdapter) Authenticate(req *http.Request) error {
	switch a.result.Kind {
	case auth.BasicAuth:
		encoded := base64.StdEncoding.EncodeToString([]byte(a.result.Token))
		req.Header.Set("Authorization", "Basic "+encoded)
	default:
		req.Header.Set("Authorization", "Bearer "+a.result.Token)
	}
	return nil
}

func (a *authAdapter) Source() string {
	return a.result.Source.String()
}

func (a *authAdapter) Level() int {
	return int(a.result.Source)
}

// NewRootCmd builds the root command with all global flags and subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bbcli",
		Short: "Bitbucket CLI for developers and AI agents",
		Long: `bbcli is a command-line interface for the Bitbucket Cloud API.
It supports pull request management, repository operations, and more.

Authentication is resolved automatically from environment variables,
CLI flags, OS keychain, or legacy token files.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Apply NO_COLOR / --no-color.
			if flagNoColor {
				os.Setenv("NO_COLOR", "1")
			}

			// Enable debug output.
			if flagDebug {
				api.DebugWriter = os.Stderr
			}

			// Detect workspace/repo from git if not provided via flags.
			workspace := flagWorkspace
			repo := flagRepo
			if workspace == "" || repo == "" {
				info, err := git.DetectRemote()
				if err != nil {
					if flagDebug {
						fmt.Fprintf(os.Stderr, "[DEBUG] git detect: %v\n", err)
					}
					// Not fatal — some commands may not need workspace/repo.
				} else {
					if workspace == "" {
						workspace = info.Workspace
					}
					if repo == "" {
						repo = info.Repo
					}
				}
			}

			// Resolve auth.
			authResult, err := auth.Resolve(flagToken, repo)
			if err != nil {
				return err
			}

			// Build the API client.
			resolvedClient = api.New(
				&authAdapter{result: authResult},
				api.WithWorkspaceRepo(workspace, repo),
				api.WithDebug(flagDebug),
				api.WithNoRetry(flagNoRetry),
			)

			return nil
		},
	}

	// Global flags.
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, json, minimal")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Shorthand for --output json")
	rootCmd.PersistentFlags().StringVarP(&flagWorkspace, "workspace", "w", "", "Bitbucket workspace (auto-detected from git remote)")
	rootCmd.PersistentFlags().StringVarP(&flagRepo, "repo", "r", "", "Repository slug (auto-detected from git remote)")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug output to stderr")
	rootCmd.PersistentFlags().BoolVar(&flagNoRetry, "no-retry", false, "Disable automatic retries on failure")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Authentication token (overrides env/keychain/file)")

	// Register subcommands.
	rootCmd.AddCommand(newPRCmd())

	return rootCmd
}

// Execute runs the root command and handles errors.
// It returns the exit code that main() should use with os.Exit.
func Execute() int {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err == nil {
		return errors.ExitSuccess
	}

	// Map CLIError to structured stderr output + exit code.
	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		// Wrap unexpected errors.
		cliErr = errors.NewGeneralError(err.Error())
	}

	envelope := errors.ErrorEnvelope{Error: cliErr}
	data, marshalErr := json.Marshal(envelope)
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, `{"error":{"code":"GENERAL_ERROR","message":%q}}%s`, err.Error(), "\n")
	} else {
		fmt.Fprintln(os.Stderr, string(data))
	}

	return cliErr.ExitCode
}
