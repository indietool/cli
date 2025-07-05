package cmd

import (
	"context"
	"fmt"

	"indietool/cli/domains"
	"indietool/cli/output"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	domainManager      *domains.Manager
	listProviderFilter string
	listExpiringIn     string
	listStatus         string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed domains from configured providers",
	Long: `List all domains managed across your configured providers.
Shows expiry dates, auto-renewal status, and nameserver information.

Examples:
  indietool domains list
  indietool domains list --provider cloudflare
  indietool domains list --expiring-in 30d
  indietool domains list --status critical --json`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the global provider registry
		registry := GetProviderRegistry()
		if registry == nil {
			handleError(fmt.Errorf("provider registry not initialized"))
			return
		}

		// Get enabled providers
		enabledProviders := registry.GetEnabledProviders()
		if len(enabledProviders) == 0 {
			log.Warnf("No providers are enabled. Please configure and enable at least one provider.")
			return
		}

		// Convert providers to registrars
		registrars := make([]domains.Registrar, 0, len(enabledProviders))
		for _, p := range enabledProviders {
			registrars = append(registrars, p.AsRegistrar())
		}

		domainManager = domains.NewManager(registrars)

		domain_list := []domains.ManagedDomain{}
		for _, registrar := range registrars {
			dlist, err := registrar.ListDomains(context.TODO())
			if err != nil {
				log.Fatalf("failed to list domains :%s", err)
			}
			domain_list = append(
				domain_list,
				dlist...,
			)
		}

		log.Infof("domains: %+v", domain_list)

		// // If a specific provider filter is requested, validate it exists
		// if listProviderFilter != "" {
		// 	if _, exists := registry.Get(listProviderFilter); !exists {
		// 		handleError(fmt.Errorf("provider '%s' is not configured", listProviderFilter))
		// 		return
		// 	}
		// }
		//
		// result, err := domains.ListManagedDomains(domains.ListOptions{
		// 	Provider:   listProviderFilter,
		// 	ExpiringIn: listExpiringIn,
		// 	Status:     listStatus,
		// })
		// if err != nil {
		// 	handleError(err)
		// 	return
		// }

		// Create a basic result for output
		result := &domains.DomainListResult{
			Domains: domain_list,
			Summary: domains.DomainSummary{
				Total: len(domain_list),
			},
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

	listCmd.Flags().StringVar(&listProviderFilter, "provider", "", "Filter by provider (cloudflare, namecheap, porkbun, godaddy)")
	listCmd.Flags().StringVar(&listExpiringIn, "expiring-in", "", "Show domains expiring within timeframe (e.g., 30d, 1w)")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (healthy, warning, critical, expired)")
}

// handleError is a placeholder for error handling
func handleError(err error) {
	// TODO: Implement proper error handling
	log.Errorf("Error: %v", err)
}
