package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// PR decline flags.
var (
	prDeclineReason string
)

// newPRDeclineCmd creates the `pr decline` subcommand.
func newPRDeclineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decline <PR_ID>",
		Short: "Decline a pull request",
		Long: `Decline a pull request by ID.

Optionally provide a reason for declining with --reason.

Idempotent: if the PR is already declined, the command succeeds
without error.`,
		Args: cobra.ExactArgs(1),
		RunE: runPRDecline,
	}

	cmd.Flags().StringVar(&prDeclineReason, "reason", "", "Reason for declining the pull request")

	return cmd
}

func runPRDecline(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.NewUsageError(fmt.Sprintf("PR_ID must be an integer, got %q", args[0]))
	}

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/decline", prID)
	raw, reqErr := client.Request("POST", path, nil)

	// Check for already-declined (idempotency).
	if reqErr != nil {
		cliErr, ok := reqErr.(*errors.CLIError)
		if ok && (strings.Contains(cliErr.Message, "DECLINED") ||
			cliErr.Code == "STATE_ERROR" ||
			strings.Contains(cliErr.Message, "already declined")) {
			result := &prDeclineResult{prID: prID, state: "already declined"}
			return output.Format(os.Stdout, result, outputMode())
		}
		return reqErr
	}

	// Parse the declined PR response.
	var pr pullRequest
	if err := json.Unmarshal(raw, &pr); err != nil {
		// Even if we can't parse, the decline succeeded.
		result := &prDeclineResult{prID: prID, state: "declined"}
		return output.Format(os.Stdout, result, outputMode())
	}

	result := &prDeclineResult{prID: pr.ID, state: pr.State, url: pr.Links.HTML.Href}
	return output.Format(os.Stdout, result, outputMode())
}

// prDeclineResult implements output.Result for a decline action.
type prDeclineResult struct {
	prID  int
	state string
	url   string
}

func (r *prDeclineResult) Headers() []string {
	return []string{"PR", "Status", "URL"}
}

func (r *prDeclineResult) Rows() [][]string {
	return [][]string{
		{fmt.Sprintf("%d", r.prID), r.state, r.url},
	}
}

func (r *prDeclineResult) MinimalLines() []string {
	return []string{fmt.Sprintf("%d", r.prID)}
}
