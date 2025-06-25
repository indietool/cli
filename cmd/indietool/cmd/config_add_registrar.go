/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// configAddRegistrarCmd represents the config add registrar command
var configAddRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Add registrar configuration",
	Long: `Add configuration for domain registrars including API credentials
and authentication details.

Supported registrars:
  - cloudflare: Requires --api-key and --email
  - porkbun: Requires --api-key and --api-secret
  - namecheap: Requires --api-key, --api-secret, and --username
  - godaddy: Requires --api-key and --api-secret

Examples:
  indietool config add registrar cloudflare --api-key YOUR_KEY --email you@example.com
  indietool config add registrar porkbun --api-key YOUR_KEY --api-secret YOUR_SECRET`,
}

func init() {
	configAddCmd.AddCommand(configAddRegistrarCmd)
}