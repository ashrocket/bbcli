package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// newPRApproveCmd creates the `pr approve` subcommand.
func newPRApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <PR_ID>",
		Short: "Approve a pull request",
		Long: `Approve a pull request by ID.

Idempotent: if the PR is already approved by the current user,
the command succeeds without error.`,
		Args: cobra.ExactArgs(1),
		RunE: runPRApprove,
	}

	return cmd
}

func runPRApprove(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.NewUsageError(fmt.Sprintf("PR_ID must be an integer, got %q", args[0]))
	}

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/approve", prID)
	_, reqErr := client.Request("POST", path, nil)

	alreadyApproved := false
	if reqErr != nil {
		// Bitbucket returns 409 when already approved. The client maps
		// non-standard 4xx to a general error containing the status code.
		cliErr, ok := reqErr.(*errors.CLIError)
		if ok && strings.Contains(cliErr.Message, "409") {
			alreadyApproved = true
		} else {
			return reqErr
		}
	}

	result := &prApproveResult{prID: prID, alreadyApproved: alreadyApproved}
	return output.Format(os.Stdout, result, outputMode())
}

// prApproveResult implements output.Result for an approval action.
type prApproveResult struct {
	prID            int
	alreadyApproved bool
}

func (r *prApproveResult) Headers() []string {
	return []string{"PR", "Status"}
}

func (r *prApproveResult) Rows() [][]string {
	status := "approved"
	if r.alreadyApproved {
		status = "already approved"
	}
	return [][]string{
		{fmt.Sprintf("%d", r.prID), status},
	}
}

func (r *prApproveResult) MinimalLines() []string {
	return []string{fmt.Sprintf("%d", r.prID)}
}
