package cmd

import (
	"github.com/spf13/cobra"
	"indietool/cli/domains"
	"indietool/cli/output"
)

var (
	listRegistrarFilter string
	listExpiringIn      string
	listStatus          string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed domains from configured registrars",
	Long: `List all domains managed across your configured registrars.
Shows expiry dates, auto-renewal status, and nameserver information.

Examples:
  indietool domains list
  indietool domains list --registrar cloudflare
  indietool domains list --expiring-in 30d
  indietool domains list --status critical --json`,
	Run: func(cmd *cobra.Command, args []string) {
		result, err := domains.ListManagedDomains(domains.ListOptions{
			Registrar:  listRegistrarFilter,
			ExpiringIn: listExpiringIn,
			Status:     listStatus,
		})
		if err != nil {
			handleError(err)
			return
		}
		
		if jsonOutput {
			output.OutputDomainListJSON(result)
		} else {
			output.OutputDomainListHuman(result)
		}
	},
}

func init() {
	domainsCmd.AddCommand(listCmd)
	
	listCmd.Flags().StringVar(&listRegistrarFilter, "registrar", "", "Filter by registrar (cloudflare, namecheap, porkbun, godaddy)")
	listCmd.Flags().StringVar(&listExpiringIn, "expiring-in", "", "Show domains expiring within timeframe (e.g., 30d, 1w)")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (healthy, warning, critical, expired)")
}

// handleError is a placeholder for error handling
func handleError(err error) {
	// TODO: Implement proper error handling
	panic(err)
}