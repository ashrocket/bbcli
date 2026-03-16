package cmd

import "github.com/spf13/cobra"

// newPRCmd creates the `pr` parent command.
func newPRCmd() *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage pull requests",
		Long:  "Create, list, review, approve, merge, and decline pull requests.",
	}

	prCmd.AddCommand(newPRListCmd())
	prCmd.AddCommand(newPRViewCmd())
	prCmd.AddCommand(newPRCreateCmd())
	prCmd.AddCommand(newPRApproveCmd())
	prCmd.AddCommand(newPRMergeCmd())

	return prCmd
}
