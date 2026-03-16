package cmd

import "github.com/spf13/cobra"

// newPipelineCmd creates the `pipeline` parent command.
func newPipelineCmd() *cobra.Command {
	pipelineCmd := &cobra.Command{
		Use:     "pipeline",
		Short:   "Manage pipelines",
		Long:    "List pipelines, view pipeline status, and inspect step details.",
		Aliases: []string{"pl"},
	}

	pipelineCmd.AddCommand(newPipelineListCmd())
	pipelineCmd.AddCommand(newPipelineStatusCmd())
	pipelineCmd.AddCommand(newPipelineWatchCmd())

	return pipelineCmd
}
