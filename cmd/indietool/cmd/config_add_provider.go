/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// configAddProviderCmd represents the config add provider command
var configAddProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Add provider configuration",
	Long: `Add configuration for service providers including API credentials
and authentication details.

Supported providers:
  - cloudflare: Requires --api-token and optionally --email
  - porkbun: Requires --api-key and --api-secret
  - namecheap: Requires --api-key and --username, optionally --client-ip and --sandbox
  - godaddy: Requires --api-key and --api-secret

Examples:
  indietool config add provider cloudflare --api-token YOUR_TOKEN --email you@example.com
  indietool config add provider porkbun --api-key YOUR_KEY --api-secret YOUR_SECRET
  indietool config add provider namecheap --api-key YOUR_KEY --username YOUR_USERNAME --client-ip 203.0.113.1`,
}

func init() {
	configAddCmd.AddCommand(configAddProviderCmd)
}
