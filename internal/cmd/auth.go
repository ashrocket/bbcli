package cmd

import "github.com/spf13/cobra"

// newAuthCmd creates the `auth` parent command.
func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  "View authentication status and manage credentials.",
	}

	authCmd.AddCommand(newAuthStatusCmd())

	return authCmd
}
