package cmd

import (
	"indietool/cli/indietool/secrets"
	"strings"

	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Aliases: []string{"secret"},
	Short:   "Manage encrypted secrets",
	Long:    "Secure secret storage with encryption and database management",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Send metrics with custom database detection
		if metricsAgent := GetMetricsAgent(); metricsAgent != nil {
			commandName := "secrets " + cmd.Name()
			metadata := make(map[string]string)

			// Check if any argument contains @database syntax
			for _, arg := range args {
				if strings.Contains(arg, "@") {
					// Parse to see if custom database is used
					_, database := secrets.ParseSecretIdentifier(arg)
					if database != "" {
						metadata["uses_custom_db"] = "true"
						break
					}
				}
			}

			// Track command execution asynchronously
			PendingItems(metricsAgent.Observe(commandName, args, metadata, 0))
		}
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)

	// Add subcommands
	secretsCmd.AddCommand(secretsInitCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsDbCmd)
}
