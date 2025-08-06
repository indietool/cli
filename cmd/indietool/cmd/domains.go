/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// domainsCmd represents the domains management command group
var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Manage domains across multiple registrars",
	Long: `Manage your domains across multiple registrars including Cloudflare, Namecheap, Porkbun, and GoDaddy.
Provides unified view of domain expiry, auto-renewal status, and nameserver configuration.

Examples:
  indietool domains list
  indietool domains sync
  indietool domains config add cloudflare`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Send metrics for domains commands
		if metricsAgent := GetMetricsAgent(); metricsAgent != nil {
			commandName := "domains " + cmd.Name()
			metadata := make(map[string]string)

			// No specific metadata needed for domains commands yet
			// Future: could track provider info if domains commands get provider-specific features

			// Track command execution asynchronously
			PendingItems(metricsAgent.Observe(commandName, args, metadata, 0))
		}
	},
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}
