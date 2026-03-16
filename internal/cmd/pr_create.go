package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/git"
	"github.com/ashrocket/bbcli/internal/output"
)

// PR create flags.
var (
	prCreateTitle             string
	prCreateSource            string
	prCreateDest              string
	prCreateDescription       string
	prCreateReviewers         string
	prCreateCloseSourceBranch bool
)

// newPRCreateCmd creates the `pr create` subcommand.
func newPRCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Long: `Create a new pull request in the current repository.

The source branch defaults to the current git branch.
The destination branch defaults to the repository's main branch.

Idempotent: if a PR already exists with the same source and destination
branches, the existing PR is returned instead of creating a duplicate.`,
		RunE: runPRCreate,
	}

	cmd.Flags().StringVarP(&prCreateTitle, "title", "t", "", "Pull request title (required)")
	cmd.Flags().StringVarP(&prCreateSource, "source", "s", "", "Source branch (default: current branch)")
	cmd.Flags().StringVarP(&prCreateDest, "dest", "d", "", "Destination branch (default: repo default branch)")
	cmd.Flags().StringVar(&prCreateDescription, "description", "", "Pull request description")
	cmd.Flags().StringVar(&prCreateReviewers, "reviewers", "", "Comma-separated list of reviewer usernames")
	cmd.Flags().BoolVar(&prCreateCloseSourceBranch, "close-source-branch", false, "Close the source branch after merge")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runPRCreate(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	// Resolve source branch from flag or current git branch.
	source := prCreateSource
	if source == "" {
		info, err := git.DetectRemote()
		if err != nil || info.Branch == "" {
			return errors.NewUsageError("could not detect current branch; use --source to specify")
		}
		source = info.Branch
	}

	// Check for existing PR with same source/dest (idempotency).
	existing, err := findExistingPR(client, source, prCreateDest)
	if err != nil {
		return err
	}
	if existing != nil {
		result := &prCreateResult{pr: *existing, existed: true}
		return output.Format(os.Stdout, result, outputMode())
	}

	// Build the request body.
	body := map[string]any{
		"title": prCreateTitle,
		"source": map[string]any{
			"branch": map[string]string{"name": source},
		},
		"close_source_branch": prCreateCloseSourceBranch,
	}

	if prCreateDest != "" {
		body["destination"] = map[string]any{
			"branch": map[string]string{"name": prCreateDest},
		}
	}

	if prCreateDescription != "" {
		body["description"] = prCreateDescription
	}

	if prCreateReviewers != "" {
		names := strings.Split(prCreateReviewers, ",")
		reviewers := make([]map[string]string, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				reviewers = append(reviewers, map[string]string{"username": name})
			}
		}
		if len(reviewers) > 0 {
			body["reviewers"] = reviewers
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to build request body: %v", err))
	}

	path := "/repositories/{workspace}/{repo_slug}/pullrequests"
	raw, err := client.Request("POST", path, bodyBytes)
	if err != nil {
		return err
	}

	var pr pullRequest
	if err := json.Unmarshal(raw, &pr); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to parse created pull request: %v", err))
	}

	result := &prCreateResult{pr: pr, existed: false}
	return output.Format(os.Stdout, result, outputMode())
}

// findExistingPR checks if there is already an OPEN PR with the given source
// and optionally destination branch. Returns nil if none found.
func findExistingPR(client *api.Client, source, dest string) (*pullRequest, error) {
	var queryParts []string
	queryParts = append(queryParts, fmt.Sprintf(`state="OPEN"`))
	queryParts = append(queryParts, fmt.Sprintf(`source.branch.name=%q`, source))
	if dest != "" {
		queryParts = append(queryParts, fmt.Sprintf(`destination.branch.name=%q`, dest))
	}

	q := strings.Join(queryParts, " AND ")
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests?q=%s", q)

	rawValues, err := client.ListAll(path, 1)
	if err != nil {
		return nil, err
	}

	if len(rawValues) == 0 {
		return nil, nil
	}

	var pr pullRequest
	if err := json.Unmarshal(rawValues[0], &pr); err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to parse existing pull request: %v", err))
	}

	return &pr, nil
}

// prCreateResult implements output.Result for a created (or found) pull request.
type prCreateResult struct {
	pr      pullRequest
	existed bool
}

func (r *prCreateResult) Headers() []string {
	return []string{"ID", "Title", "State", "Source", "Destination", "URL", "Note"}
}

func (r *prCreateResult) Rows() [][]string {
	note := "created"
	if r.existed {
		note = "already existed"
	}
	pr := r.pr
	return [][]string{
		{
			fmt.Sprintf("%d", pr.ID),
			pr.Title,
			pr.State,
			pr.Source.Branch.Name,
			pr.Destination.Branch.Name,
			pr.Links.HTML.Href,
			note,
		},
	}
}

func (r *prCreateResult) MinimalLines() []string {
	return []string{fmt.Sprintf("%d", r.pr.ID)}
}
