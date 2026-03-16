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

// PR merge flags.
var (
	prMergeStrategy          string
	prMergeMessage           string
	prMergeCloseSourceBranch bool
)

// newPRMergeCmd creates the `pr merge` subcommand.
func newPRMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge <PR_ID>",
		Short: "Merge a pull request",
		Long: `Merge a pull request by ID.

Supports merge commit, squash, and fast-forward strategies.

Idempotent: if the PR is already merged, the command succeeds
without error.`,
		Args: cobra.ExactArgs(1),
		RunE: runPRMerge,
	}

	cmd.Flags().StringVar(&prMergeStrategy, "strategy", "merge_commit", "Merge strategy: merge_commit, squash, fast_forward")
	cmd.Flags().StringVarP(&prMergeMessage, "message", "m", "", "Merge commit message")
	cmd.Flags().BoolVar(&prMergeCloseSourceBranch, "close-source-branch", false, "Close the source branch after merge")

	return cmd
}

func runPRMerge(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.NewUsageError(fmt.Sprintf("PR_ID must be an integer, got %q", args[0]))
	}

	// Validate strategy.
	validStrategies := map[string]bool{
		"merge_commit": true,
		"squash":       true,
		"fast_forward": true,
	}
	if !validStrategies[prMergeStrategy] {
		return errors.NewUsageError(fmt.Sprintf("invalid merge strategy %q; must be merge_commit, squash, or fast_forward", prMergeStrategy))
	}

	// Build merge request body.
	body := map[string]any{
		"type":                "pullrequest",
		"merge_strategy":     prMergeStrategy,
		"close_source_branch": prMergeCloseSourceBranch,
	}
	if prMergeMessage != "" {
		body["message"] = prMergeMessage
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to build request body: %v", err))
	}

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/merge", prID)
	raw, reqErr := client.Request("POST", path, bodyBytes)

	// Check for already-merged (idempotency).
	if reqErr != nil {
		cliErr, ok := reqErr.(*errors.CLIError)
		if ok && strings.Contains(cliErr.Message, "MERGED") {
			result := &prMergeResult{prID: prID, state: "already merged"}
			return output.Format(os.Stdout, result, outputMode())
		}
		// Bitbucket may also return a state error when the PR is already merged.
		if ok && (cliErr.Code == "STATE_ERROR" || strings.Contains(cliErr.Message, "already merged")) {
			result := &prMergeResult{prID: prID, state: "already merged"}
			return output.Format(os.Stdout, result, outputMode())
		}
		return reqErr
	}

	// Parse the merged PR response.
	var pr pullRequest
	if err := json.Unmarshal(raw, &pr); err != nil {
		// Even if we can't parse, the merge succeeded.
		result := &prMergeResult{prID: prID, state: "merged"}
		return output.Format(os.Stdout, result, outputMode())
	}

	result := &prMergeResult{prID: pr.ID, state: pr.State, url: pr.Links.HTML.Href}
	return output.Format(os.Stdout, result, outputMode())
}

// prMergeResult implements output.Result for a merge action.
type prMergeResult struct {
	prID  int
	state string
	url   string
}

func (r *prMergeResult) Headers() []string {
	return []string{"PR", "Status", "URL"}
}

func (r *prMergeResult) Rows() [][]string {
	return [][]string{
		{fmt.Sprintf("%d", r.prID), r.state, r.url},
	}
}

func (r *prMergeResult) MinimalLines() []string {
	return []string{fmt.Sprintf("%d", r.prID)}
}
