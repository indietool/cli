package cmd

import (
	"context"
	"fmt"
	"indietool/cli/domains"
	"indietool/cli/indietool"
	"indietool/cli/output"
	"os"
	"sort"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	domainManager      *domains.Manager
	listProviderFilter string
	listExpiringIn     string
	listStatus         string
	listWideOutput     bool
	listNoHeaders      bool
	listShowSummary    bool
	listNoColor        bool
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

		registrars := indietool.GetProviders[domains.Registrar](registry)
		domainManager = domains.NewManager(registrars)

		// Collect domains from all registrars
		domainList := []domains.ManagedDomain{}
		wg := sync.WaitGroup{}
		domainsMux := sync.Mutex{}

		for _, registrar := range registrars {
			wg.Add(1)

			go func(reg domains.Registrar) {
				defer wg.Done()

				dlist, err := reg.ListDomains(context.TODO())
				if err != nil {
					log.Errorf("Failed to list domains from registrar: %s", err)
					return
				}

				domainsMux.Lock()
				domainList = append(domainList, dlist...)
				domainsMux.Unlock()
			}(registrar)
		}

		wg.Wait()

		sort.SliceStable(domainList, func(i, j int) bool {
			return domainList[i].Name < domainList[j].Name
		})

		// TODO: Apply additional filters (expiring-in, status) here
		// This would be implemented as part of the filtering logic

		// Determine output format and render table
		format := domains.GetOutputFormat(jsonOutput, listWideOutput)
		options := domains.DomainTableOptions(format, listWideOutput, listNoColor, listNoHeaders, os.Stdout)

		// Get appropriate table config (disable colors for tabwriter formats to avoid alignment issues)
		// For Table/Wide formats, we always disable colors to prevent ANSI codes from breaking column alignment
		useColors := !listNoColor && (format != output.FormatTable && format != output.FormatWide)
		tableConfig := domains.GetDomainTableConfig(useColors)

		table := output.NewTable(tableConfig, options)
		table.AddRows(domainList)

		if listShowSummary || (!jsonOutput && format != output.FormatJSON) {
			if err := table.RenderWithSummary(); err != nil {
				handleError(fmt.Errorf("failed to render table: %w", err))
			}
		} else {
			if err := table.Render(); err != nil {
				handleError(fmt.Errorf("failed to render table: %w", err))
			}
		}
	},
}

func init() {
	domainsCmd.AddCommand(listCmd)

	// Filtering flags
	listCmd.Flags().StringVar(&listProviderFilter, "provider", "", "Filter by provider (cloudflare, namecheap, porkbun, godaddy)")
	listCmd.Flags().StringVar(&listExpiringIn, "expiring-in", "", "Show domains expiring within timeframe (e.g., 30d, 1w)")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (healthy, warning, critical, expired)")

	// Output format flags
	listCmd.Flags().BoolVarP(&listWideOutput, "wide", "w", false, "Show additional columns (nameservers, cost, updated)")
	listCmd.Flags().BoolVar(&listNoHeaders, "no-headers", false, "Don't show column headers")
	listCmd.Flags().BoolVar(&listShowSummary, "show-summary", true, "Show summary statistics")
	listCmd.Flags().BoolVar(&listNoColor, "no-color", false, "Disable colored output")

	// These flags are inherited from the global flags defined in root.go:
	// --json: Output in JSON format
}

// calculateDomainSummary calculates summary statistics for a list of domains
func calculateDomainSummary(domainList []domains.ManagedDomain) domains.DomainSummary {
	summary := domains.DomainSummary{
		Total: len(domainList),
	}

	for _, domain := range domainList {
		switch domain.Status {
		case domains.StatusHealthy:
			summary.Healthy++
		case domains.StatusWarning:
			summary.Warning++
		case domains.StatusCritical:
			summary.Critical++
		case domains.StatusExpired:
			summary.Expired++
		}
	}

	return summary
}

// handleError is a placeholder for error handling
func handleError(err error) {
	// TODO: Implement proper error handling
	log.Errorf("Error: %v", err)
}
