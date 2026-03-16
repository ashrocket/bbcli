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

// Pipeline watch flags.
var (
	pipelineWatchInterval int
	pipelineWatchTimeout  int
	pipelineWatchExitCode bool
)

// newPipelineWatchCmd creates the `pipeline watch` subcommand.
func newPipelineWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch [BUILD_NUMBER]",
		Short: "Watch a pipeline until completion",
		Long: `Watch a pipeline build until it completes, polling for status updates.

If no build number is given, watches the latest pipeline.

In table mode, status updates are printed to stderr and the final
result is printed to stdout.

In json mode, output is silent until complete, then prints the final
pipeline state as JSON.

In minimal mode, prints SUCCESSFUL or FAILED when complete.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runPipelineWatch,
	}

	cmd.Flags().IntVar(&pipelineWatchInterval, "interval", 10, "Polling interval in seconds")
	cmd.Flags().IntVar(&pipelineWatchTimeout, "timeout", 900, "Timeout in seconds (default 15 minutes)")
	cmd.Flags().BoolVar(&pipelineWatchExitCode, "exit-code", false, "Exit 0 on pass, 1 on fail")

	return cmd
}

func runPipelineWatch(cmd *cobra.Command, args []string) error {
	client := resolvedClient
	if client == nil {
		return errors.NewGeneralError("API client not initialized")
	}

	var targetBuildNumber int
	if len(args) > 0 {
		var err error
		targetBuildNumber, err = strconv.Atoi(args[0])
		if err != nil {
			return errors.NewUsageError(fmt.Sprintf("BUILD_NUMBER must be an integer, got %q", args[0]))
		}
	}

	// Find the pipeline UUID to watch.
	pipelineUUID, buildNumber, err := findPipelineUUID(targetBuildNumber)
	if err != nil {
		return err
	}

	uuid := strings.Trim(pipelineUUID, "{}")
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pipelines/%s", uuid)

	mode := outputMode()
	deadline := time.Now().Add(time.Duration(pipelineWatchTimeout) * time.Second)
	interval := time.Duration(pipelineWatchInterval) * time.Second

	var lastState string
	for {
		raw, err := client.Request("GET", path, nil)
		if err != nil {
			return err
		}

		var p pipeline
		if err := json.Unmarshal(raw, &p); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to parse pipeline: %v", err))
		}

		currentState := pipelineState(p)

		// In table mode, print status updates to stderr.
		if mode == "table" && currentState != lastState {
			fmt.Fprintf(os.Stderr, "[%s] Pipeline #%d: %s\n",
				time.Now().Format("15:04:05"), buildNumber, currentState)
			lastState = currentState
		}

		// Check if pipeline is complete.
		if isPipelineComplete(p) {
			return formatWatchResult(p, mode, buildNumber)
		}

		// Check timeout.
		if time.Now().After(deadline) {
			return errors.NewGeneralError(fmt.Sprintf(
				"timeout waiting for pipeline #%d after %d seconds (last state: %s)",
				buildNumber, pipelineWatchTimeout, currentState))
		}

		time.Sleep(interval)
	}
}

// findPipelineUUID finds the UUID for a given build number, or the latest pipeline.
func findPipelineUUID(buildNumber int) (string, int, error) {
	client := resolvedClient
	path := "/repositories/{workspace}/{repo_slug}/pipelines/?sort=-created_on"
	rawValues, err := client.ListAll(path, 100)
	if err != nil {
		return "", 0, err
	}

	if len(rawValues) == 0 {
		return "", 0, errors.NewNotFoundError("pipeline", "none found")
	}

	// If no build number specified, use the latest.
	if buildNumber == 0 {
		var p pipeline
		if err := json.Unmarshal(rawValues[0], &p); err != nil {
			return "", 0, errors.NewGeneralError(fmt.Sprintf("failed to parse pipeline: %v", err))
		}
		return p.UUID, p.BuildNumber, nil
	}

	// Search for the specific build number.
	for _, raw := range rawValues {
		var p pipeline
		if err := json.Unmarshal(raw, &p); err != nil {
			continue
		}
		if p.BuildNumber == buildNumber {
			return p.UUID, p.BuildNumber, nil
		}
	}

	return "", 0, errors.NewNotFoundError("pipeline", buildNumber)
}

// isPipelineComplete returns true if the pipeline has reached a terminal state.
func isPipelineComplete(p pipeline) bool {
	state := strings.ToUpper(p.State.Name)
	return state == "COMPLETED" || state == "ERROR" || state == "STOPPED"
}

// formatWatchResult outputs the final pipeline result based on the output mode.
func formatWatchResult(p pipeline, mode string, buildNumber int) error {
	resultName := ""
	if p.State.Result != nil {
		resultName = strings.ToUpper(p.State.Result.Name)
	}

	if pipelineWatchExitCode && resultName != "SUCCESSFUL" {
		// Format output first, then return an error for exit code 1.
		result := &pipelineWatchResult{pipeline: p, buildNumber: buildNumber}
		output.Format(os.Stdout, result, mode)
		return errors.NewGeneralError(fmt.Sprintf("pipeline #%d finished with result: %s",
			buildNumber, pipelineState(p)))
	}

	result := &pipelineWatchResult{pipeline: p, buildNumber: buildNumber}
	return output.Format(os.Stdout, result, mode)
}

// pipelineWatchResult implements output.Result for the final watch output.
type pipelineWatchResult struct {
	pipeline    pipeline
	buildNumber int
}

func (r *pipelineWatchResult) Headers() []string {
	return []string{"Build", "State", "Branch", "Creator", "Created"}
}

func (r *pipelineWatchResult) Rows() [][]string {
	p := r.pipeline
	creator := p.Creator.DisplayName
	if creator == "" {
		creator = p.Creator.Nickname
	}
	return [][]string{
		{
			fmt.Sprintf("%d", r.buildNumber),
			pipelineState(p),
			p.Target.RefName,
			creator,
			formatPipelineTime(p.CreatedOn),
		},
	}
}

func (r *pipelineWatchResult) MinimalLines() []string {
	result := "FAILED"
	if r.pipeline.State.Result != nil {
		result = strings.ToUpper(r.pipeline.State.Result.Name)
	}
	return []string{result}
}
