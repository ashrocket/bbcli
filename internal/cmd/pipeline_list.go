package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// Pipeline list flags.
var (
	pipelineListBranch string
	pipelineListLimit  int
)

// newPipelineListCmd creates the `pipeline list` subcommand.
func newPipelineListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipelines",
		Long: `List pipelines for the current repository.

By default, lists the most recent pipelines sorted by creation date.
Use --branch to filter by branch name.`,
		Aliases: []string{"ls"},
		RunE:    runPipelineList,
	}

	cmd.Flags().StringVarP(&pipelineListBranch, "branch", "b", "", "Filter by branch name")
	cmd.Flags().IntVarP(&pipelineListLimit, "limit", "l", 50, "Maximum number of results")

	return cmd
}

func runPipelineList(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	path := "/repositories/{workspace}/{repo_slug}/pipelines/?sort=-created_on"

	// Fetch all pages up to the limit.
	rawValues, err := client.ListAll(path, pipelineListLimit)
	if err != nil {
		return err
	}

	// Parse into pipeline structs and optionally filter by branch.
	pipelines := make([]pipeline, 0, len(rawValues))
	for _, raw := range rawValues {
		var p pipeline
		if err := json.Unmarshal(raw, &p); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to parse pipeline: %v", err))
		}
		if pipelineListBranch != "" && !strings.EqualFold(p.Target.RefName, pipelineListBranch) {
			continue
		}
		pipelines = append(pipelines, p)
	}

	// Format and print.
	result := &pipelineListResult{pipelines: pipelines}
	return output.Format(os.Stdout, result, outputMode())
}

// pipeline represents the fields we extract from the Bitbucket Pipelines API response.
type pipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	Target struct {
		RefName string `json:"ref_name"`
		RefType string `json:"ref_type"`
	} `json:"target"`
	CreatedOn string `json:"created_on"`
	Creator   struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	} `json:"creator"`
}

// pipelineState returns a combined state/result string.
func pipelineState(p pipeline) string {
	state := p.State.Name
	if p.State.Result != nil && p.State.Result.Name != "" {
		return state + "/" + p.State.Result.Name
	}
	return state
}

// formatPipelineTime formats an ISO 8601 time string for display.
func formatPipelineTime(t string) string {
	parsed, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		return t
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

// pipelineListResult implements output.Result for a list of pipelines.
type pipelineListResult struct {
	pipelines []pipeline
}

func (r *pipelineListResult) Headers() []string {
	return []string{"Build", "State", "Branch", "Creator", "Created"}
}

func (r *pipelineListResult) Rows() [][]string {
	rows := make([][]string, len(r.pipelines))
	for i, p := range r.pipelines {
		creator := p.Creator.DisplayName
		if creator == "" {
			creator = p.Creator.Nickname
		}
		rows[i] = []string{
			fmt.Sprintf("%d", p.BuildNumber),
			pipelineState(p),
			p.Target.RefName,
			creator,
			formatPipelineTime(p.CreatedOn),
		}
	}
	return rows
}

func (r *pipelineListResult) MinimalLines() []string {
	lines := make([]string, len(r.pipelines))
	for i, p := range r.pipelines {
		lines[i] = fmt.Sprintf("%d", p.BuildNumber)
	}
	return lines
}
