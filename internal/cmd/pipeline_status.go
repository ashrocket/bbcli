package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// newPipelineStatusCmd creates the `pipeline status` subcommand.
func newPipelineStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <BUILD_NUMBER>",
		Short: "Show pipeline status and steps",
		Long: `Show detailed status for a specific pipeline build.

Displays the pipeline overview and step-by-step execution status.`,
		Args: cobra.ExactArgs(1),
		RunE: runPipelineStatus,
	}

	return cmd
}

func runPipelineStatus(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	buildNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.NewUsageError(fmt.Sprintf("invalid build number: %s", args[0]))
	}

	// List pipelines to find the one matching the build number.
	// The API doesn't support direct lookup by build_number, so we paginate.
	path := "/repositories/{workspace}/{repo_slug}/pipelines/?sort=-created_on"
	rawValues, fetchErr := client.ListAll(path, 200)
	if fetchErr != nil {
		return fetchErr
	}

	var found *pipeline
	for _, raw := range rawValues {
		var p pipeline
		if err := json.Unmarshal(raw, &p); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to parse pipeline: %v", err))
		}
		if p.BuildNumber == buildNumber {
			found = &p
			break
		}
	}

	if found == nil {
		return errors.NewNotFoundError("pipeline", buildNumber)
	}

	// Fetch steps for this pipeline.
	// Bitbucket API requires UUID with braces, URL-encoded as %7B...%7D
	uuid := found.UUID
	if !strings.HasPrefix(uuid, "{") {
		uuid = "{" + uuid + "}"
	}
	encodedUUID := strings.ReplaceAll(strings.ReplaceAll(uuid, "{", "%7B"), "}", "%7D")
	stepsPath := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pipelines/%s/steps/", encodedUUID)
	stepsRaw, err := client.ListAll(stepsPath, 100)
	if err != nil {
		return err
	}

	steps := make([]pipelineStep, 0, len(stepsRaw))
	for _, raw := range stepsRaw {
		var s pipelineStep
		if err := json.Unmarshal(raw, &s); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to parse pipeline step: %v", err))
		}
		steps = append(steps, s)
	}

	// Format and print.
	result := &pipelineStatusResult{pipeline: *found, steps: steps}
	return output.Format(os.Stdout, result, outputMode())
}

// pipelineStep represents a step within a pipeline.
type pipelineStep struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	StartedOn   string `json:"started_on"`
	CompletedOn string `json:"completed_on"`
	RunTime     int    `json:"run_time_in_seconds"`
}

// stepState returns a combined state/result string for a step.
func stepState(s pipelineStep) string {
	state := s.State.Name
	if s.State.Result != nil && s.State.Result.Name != "" {
		return state + "/" + s.State.Result.Name
	}
	return state
}

// formatDuration formats seconds into a human-readable duration.
func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "-"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", seconds)
	}
	m := int(d.Minutes())
	s := seconds - m*60
	return fmt.Sprintf("%dm%ds", m, s)
}

// pipelineStatusResult implements output.Result for pipeline status + steps.
type pipelineStatusResult struct {
	pipeline pipeline
	steps    []pipelineStep
}

func (r *pipelineStatusResult) Headers() []string {
	return []string{"Step", "Name", "State", "Duration", "Started", "Completed"}
}

func (r *pipelineStatusResult) Rows() [][]string {
	p := r.pipeline
	// First row: pipeline overview as a summary row.
	creator := p.Creator.DisplayName
	if creator == "" {
		creator = p.Creator.Nickname
	}
	rows := [][]string{
		{
			fmt.Sprintf("#%d", p.BuildNumber),
			fmt.Sprintf("[pipeline] %s", p.Target.RefName),
			pipelineState(p),
			"-",
			formatPipelineTime(p.CreatedOn),
			"",
		},
	}

	// Step rows.
	for i, s := range r.steps {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("Step %d", i+1)
		}
		rows = append(rows, []string{
			fmt.Sprintf("  %d", i+1),
			name,
			stepState(s),
			formatDuration(s.RunTime),
			formatPipelineTime(s.StartedOn),
			formatPipelineTime(s.CompletedOn),
		})
	}

	return rows
}

func (r *pipelineStatusResult) MinimalLines() []string {
	lines := []string{fmt.Sprintf("%d %s", r.pipeline.BuildNumber, pipelineState(r.pipeline))}
	for _, s := range r.steps {
		name := s.Name
		if name == "" {
			name = "step"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", name, stepState(s)))
	}
	return lines
}
