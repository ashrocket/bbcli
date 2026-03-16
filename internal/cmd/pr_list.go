package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// PR list flags.
var (
	prListState  string
	prListAuthor string
	prListDest   string
	prListLimit  int
)

// newPRListCmd creates the `pr list` subcommand.
func newPRListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
		Long: `List pull requests for the current repository.

By default, lists open pull requests. Use --state to filter by state
(OPEN, MERGED, DECLINED, SUPERSEDED).`,
		Aliases: []string{"ls"},
		RunE:    runPRList,
	}

	cmd.Flags().StringVarP(&prListState, "state", "s", "OPEN", "Filter by state: OPEN, MERGED, DECLINED, SUPERSEDED")
	cmd.Flags().StringVarP(&prListAuthor, "author", "a", "", "Filter by author username")
	cmd.Flags().StringVarP(&prListDest, "dest", "d", "", "Filter by destination branch")
	cmd.Flags().IntVarP(&prListLimit, "limit", "l", 50, "Maximum number of results")

	return cmd
}

func runPRList(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	// Build the query string.
	var queryParts []string
	state := strings.ToUpper(prListState)
	queryParts = append(queryParts, fmt.Sprintf("state=%q", state))
	if prListAuthor != "" {
		queryParts = append(queryParts, fmt.Sprintf("author.username=%q", prListAuthor))
	}
	if prListDest != "" {
		queryParts = append(queryParts, fmt.Sprintf("destination.branch.name=%q", prListDest))
	}

	q := strings.Join(queryParts, " AND ")
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests?q=%s", q)

	// Fetch all pages up to the limit.
	rawValues, err := client.ListAll(path, prListLimit)
	if err != nil {
		return err
	}

	// Parse into PR structs.
	prs := make([]pullRequest, 0, len(rawValues))
	for _, raw := range rawValues {
		var pr pullRequest
		if err := json.Unmarshal(raw, &pr); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to parse pull request: %v", err))
		}
		prs = append(prs, pr)
	}

	// Format and print.
	result := &prListResult{prs: prs}
	return output.Format(os.Stdout, result, outputMode())
}

// pullRequest represents the fields we extract from the Bitbucket API response.
type pullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
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
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// prListResult implements output.Result for a list of pull requests.
type prListResult struct {
	prs []pullRequest
}

func (r *prListResult) Headers() []string {
	return []string{"ID", "Title", "State", "Author", "Source", "Destination", "URL"}
}

func (r *prListResult) Rows() [][]string {
	rows := make([][]string, len(r.prs))
	for i, pr := range r.prs {
		author := pr.Author.DisplayName
		if author == "" {
			author = pr.Author.Nickname
		}
		rows[i] = []string{
			fmt.Sprintf("%d", pr.ID),
			pr.Title,
			pr.State,
			author,
			pr.Source.Branch.Name,
			pr.Destination.Branch.Name,
			pr.Links.HTML.Href,
		}
	}
	return rows
}

func (r *prListResult) MinimalLines() []string {
	lines := make([]string, len(r.prs))
	for i, pr := range r.prs {
		lines[i] = fmt.Sprintf("%d", pr.ID)
	}
	return lines
}
