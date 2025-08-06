/*
Copyright © 2025
*/
package cmd

import (
	"indietool/cli/dns"
	"indietool/cli/indietool"

	"github.com/spf13/cobra"
)

// DNS command flags (consolidated from subcommands)
var (
	dnsProvider   string
	dnsWideOutput bool
	dnsNoHeaders  bool
	dnsNoColor    bool
)

// DNS command state
var (
	dnsManager *dns.Manager
)

// dnsCmd represents the dns command
var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS records for your domains",
	Long: `Manage DNS records for your domains across different DNS providers.
Supports listing, setting, and deleting DNS records with automatic provider detection.

Examples:
  indietool dns list example.com
  indietool dns set example.com www A 192.168.1.1
  indietool dns delete example.com www A
  indietool dns set example.com @ MX "10 mail.example.com" --priority 10`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize DNS manager
		registry := GetProviderRegistry()
		if registry != nil {
			dnsProviders := indietool.GetProviders[dns.Provider](registry)
			dnsManager = dns.NewManager(dnsProviders)
		}

		// Send metrics with provider detection
		if metricsAgent := GetMetricsAgent(); metricsAgent != nil {
			commandName := "dns " + cmd.Name()
			metadata := make(map[string]string)

			// Detect provider if we have a domain argument
			if len(args) > 0 && args[0] != "" {
				if dnsProvider != "" {
					// Explicit provider via --provider flag
					metadata["provider"] = dnsProvider
					metadata["provider_source"] = "explicit"
				} else {
					// Try to detect provider
					if result, err := dns.DetectProvider(args[0]); err == nil && result.Provider != "" {
						metadata["provider"] = result.Provider
						metadata["provider_source"] = "detected"
					}
				}
			}

			// Track command execution asynchronously
			PendingItems(metricsAgent.Observe(commandName, args, metadata, 0))
		}
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)

	// Consolidated DNS flags (persistent across all DNS subcommands)
	dnsCmd.PersistentFlags().StringVar(&dnsProvider, "provider", "", "DNS provider to use (cloudflare, namecheap, porkbun, godaddy)")
	dnsCmd.PersistentFlags().BoolVarP(&dnsWideOutput, "wide", "w", false, "Show additional columns (ID, TTL, Priority)")
	dnsCmd.PersistentFlags().BoolVar(&dnsNoHeaders, "no-headers", false, "Don't show column headers")
	dnsCmd.PersistentFlags().BoolVar(&dnsNoColor, "no-color", false, "Disable colored output")
}

// GetDNSManager returns the initialized DNS manager for subcommands
func GetDNSManager() *dns.Manager {
	return dnsManager
}

// GetDNSProvider returns the provider flag value
func GetDNSProvider() string {
	return dnsProvider
}

// GetDNSOutputFlags returns the output formatting flags
func GetDNSOutputFlags() (wide, noHeaders, noColor bool) {
	return dnsWideOutput, dnsNoHeaders, dnsNoColor
}
