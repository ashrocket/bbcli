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

// newPRViewCmd creates the `pr view` subcommand.
func newPRViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <PR_ID>",
		Short: "View a pull request",
		Long: `View detailed information about a specific pull request.

Shows the PR title, state, author, source/destination branches,
description, reviewers, and URL.`,
		Args: cobra.ExactArgs(1),
		RunE: runPRView,
	}

	return cmd
}

func runPRView(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.NewUsageError(fmt.Sprintf("PR_ID must be an integer, got %q", args[0]))
	}

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d", prID)
	raw, err := client.Request("GET", path, nil)
	if err != nil {
		return err
	}

	var pr pullRequestDetail
	if err := json.Unmarshal(raw, &pr); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to parse pull request: %v", err))
	}

	result := &prViewResult{pr: pr}
	return output.Format(os.Stdout, result, outputMode())
}

// pullRequestDetail has more fields than pullRequest for single-PR view.
type pullRequestDetail struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Description string `json:"description"`
	Author      struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		Nickname    string `json:"nickname"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CloseSourceBranch bool `json:"close_source_branch"`
	Reviewers         []struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		Nickname    string `json:"nickname"`
	} `json:"reviewers"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

// prViewResult implements output.Result for a single pull request.
type prViewResult struct {
	pr pullRequestDetail
}

func (r *prViewResult) Headers() []string {
	return []string{"Field", "Value"}
}

func (r *prViewResult) Rows() [][]string {
	pr := r.pr
	author := pr.Author.DisplayName
	if author == "" {
		author = pr.Author.Nickname
	}

	reviewerNames := make([]string, len(pr.Reviewers))
	for i, rev := range pr.Reviewers {
		name := rev.DisplayName
		if name == "" {
			name = rev.Nickname
		}
		reviewerNames[i] = name
	}

	return [][]string{
		{"ID", fmt.Sprintf("%d", pr.ID)},
		{"Title", pr.Title},
		{"State", pr.State},
		{"Author", author},
		{"Source", pr.Source.Branch.Name},
		{"Destination", pr.Destination.Branch.Name},
		{"Description", pr.Description},
		{"Reviewers", strings.Join(reviewerNames, ", ")},
		{"Close Source Branch", fmt.Sprintf("%v", pr.CloseSourceBranch)},
		{"Created", pr.CreatedOn},
		{"Updated", pr.UpdatedOn},
		{"URL", pr.Links.HTML.Href},
	}
}

func (r *prViewResult) MinimalLines() []string {
	return []string{fmt.Sprintf("%d", r.pr.ID)}
}
