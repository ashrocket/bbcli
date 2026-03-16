package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// Branch delete flags.
var (
	branchDeleteYes bool
)

// protectedBranches are branches that cannot be deleted.
var protectedBranches = map[string]bool{
	"main":   true,
	"master": true,
	"dev":    true,
}

// newBranchDeleteCmd creates the `branch delete` subcommand.
func newBranchDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <BRANCH_NAME>",
		Short: "Delete a branch",
		Long: `Delete a branch from the remote repository.

Safety: refuses to delete main, master, or dev branches.

Idempotent: if the branch does not exist (404), the command
succeeds with a note that the branch was already absent.

In non-TTY mode, the --yes flag is required to confirm deletion.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE:    runBranchDelete,
	}

	cmd.Flags().BoolVarP(&branchDeleteYes, "yes", "y", false, "Skip confirmation (required in non-TTY mode)")

	return cmd
}

func runBranchDelete(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	branchName := args[0]

	// Safety: refuse to delete protected branches.
	if protectedBranches[strings.ToLower(branchName)] {
		return errors.NewStateError(
			"PROTECTED_BRANCH",
			fmt.Sprintf("refusing to delete protected branch %q", branchName),
			"Protected branches (main, master, dev) cannot be deleted via CLI",
		)
	}

	// Check for TTY and confirmation.
	if !branchDeleteYes {
		// Check if stdin is a terminal.
		stat, _ := os.Stdin.Stat()
		isTTY := (stat.Mode() & os.ModeCharDevice) != 0
		if !isTTY {
			return errors.NewUsageError("--yes/-y flag is required in non-TTY mode to confirm branch deletion")
		}
		// In TTY mode without --yes, prompt for confirmation.
		fmt.Fprintf(os.Stderr, "Delete branch %q? [y/N] ", branchName)
		var response string
		fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
			return errors.NewGeneralError("branch deletion cancelled")
		}
	}

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/refs/branches/%s", branchName)
	_, reqErr := client.Request("DELETE", path, nil)

	if reqErr != nil {
		// Check for 404 — branch already doesn't exist (idempotent success).
		cliErr, ok := reqErr.(*errors.CLIError)
		if ok && cliErr.ExitCode == errors.ExitNotFound {
			result := &branchDeleteResult{branch: branchName, status: "already absent"}
			return output.Format(os.Stdout, result, outputMode())
		}
		return reqErr
	}

	result := &branchDeleteResult{branch: branchName, status: "deleted"}
	return output.Format(os.Stdout, result, outputMode())
}

// branchDeleteResult implements output.Result for a branch delete action.
type branchDeleteResult struct {
	branch string
	status string
}

func (r *branchDeleteResult) Headers() []string {
	return []string{"Branch", "Status"}
}

func (r *branchDeleteResult) Rows() [][]string {
	return [][]string{
		{r.branch, r.status},
	}
}

func (r *branchDeleteResult) MinimalLines() []string {
	return []string{r.branch}
}
