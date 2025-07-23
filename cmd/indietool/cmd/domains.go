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
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}
