package cmd

import "github.com/spf13/cobra"

// newBranchCmd creates the `branch` parent command.
func newBranchCmd() *cobra.Command {
	branchCmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage branches",
		Long:  "Delete and manage repository branches.",
	}

	branchCmd.AddCommand(newBranchDeleteCmd())

	return branchCmd
}
