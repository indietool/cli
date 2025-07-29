package cmd

import (
	"github.com/spf13/cobra"
)

var secretsDbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage secrets databases",
	Long:  "Commands for managing secrets databases including listing and deleting databases.",
}

func init() {
	// Add subcommands to db command
	secretsDbCmd.AddCommand(secretsDbListCmd)
	secretsDbCmd.AddCommand(secretsDbDeleteCmd)
}
