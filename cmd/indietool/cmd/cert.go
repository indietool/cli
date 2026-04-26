package cmd

import (
	"github.com/spf13/cobra"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "View TLS certificate details and visual fingerprint",
	Long: `View TLS certificate details and generate a visual fingerprint (identicon or randomart).

Supports loading certificates from PEM files or connecting to remote hosts.

Examples:
  indietool cert show example.com
  indietool cert show /path/to/cert.pem
  indietool cert show example.com --type randomart
  indietool cert show example.com --insecure`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if metricsAgent := GetMetricsAgent(); metricsAgent != nil {
			commandName := "cert " + cmd.Name()
			metadata := make(map[string]string)
			PendingItems(metricsAgent.Observe(commandName, args, metadata, 0))
		}
	},
}

func init() {
	rootCmd.AddCommand(certCmd)
}
