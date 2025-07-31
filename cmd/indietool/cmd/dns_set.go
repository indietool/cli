package cmd

import (
	"context"
	"fmt"
	"indietool/cli/dns"
	"indietool/cli/indietool"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	dnsSetProvider string
	dnsSetTTL      int
	dnsSetPriority int
)

var dnsSetCmd = &cobra.Command{
	Use:   "set <domain> <name> <type> <value>",
	Short: "Set a DNS record for a domain",
	Long: `Set or update a DNS record for a domain.
Automatically detects the DNS provider or use --provider to specify.

Examples:
  indietool dns set example.com www A 192.168.1.1
  indietool dns set example.com @ MX "10 mail.example.com"
  indietool dns set example.com --provider cloudflare www CNAME "other.example.com"
  indietool dns set example.com _dmarc TXT "v=DMARC1; p=reject"`,
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		domain := args[0]
		name := args[1]
		recordType := args[2]
		value := args[3]

		// Get the global provider registry
		registry := GetProviderRegistry()
		if registry == nil {
			handleDNSError(fmt.Errorf("provider registry not initialized"))
			return
		}

		// Get DNS providers from registry
		dnsProviders := indietool.GetProviders[dns.Provider](registry)
		if len(dnsProviders) == 0 {
			handleDNSError(fmt.Errorf("no DNS providers configured"))
			return
		}

		// Create DNS manager
		dnsManager := dns.NewManager(dnsProviders)

		// Build DNS record
		record := dns.Record{
			Name:    name,
			Type:    recordType,
			Content: value,
			TTL:     dnsSetTTL,
		}

		// Handle priority for MX records
		if recordType == "MX" && dnsSetPriority > 0 {
			record.Priority = &dnsSetPriority
		}

		// Set DNS record
		detectionResult, err := dnsManager.SetRecord(context.TODO(), domain, dnsSetProvider, record)
		if err != nil {
			handleDNSError(fmt.Errorf("failed to set DNS record: %w", err))
			return
		}

		// Log detection result for debugging
		if detectionResult != nil {
			if detectionResult.Provider != "" {
				log.Debugf("Used DNS provider: %s (confidence: %s)", detectionResult.Provider, detectionResult.Confidence)
			} else {
				log.Debugf("Failed to detect DNS provider: %s", detectionResult.Error)
			}
		}

		// Success message
		if dnsSetProvider != "" {
			fmt.Printf("Successfully set DNS record %s %s %s via %s\n", name, recordType, value, dnsSetProvider)
		} else if detectionResult != nil && detectionResult.Provider != "" {
			fmt.Printf("Successfully set DNS record %s %s %s via %s\n", name, recordType, value, detectionResult.Provider)
		} else {
			fmt.Printf("Successfully set DNS record %s %s %s\n", name, recordType, value)
		}
	},
}

func init() {
	dnsCmd.AddCommand(dnsSetCmd)

	// Provider flag
	dnsSetCmd.Flags().StringVar(&dnsSetProvider, "provider", "", "DNS provider to use (cloudflare, namecheap, porkbun, godaddy)")

	// DNS record options
	dnsSetCmd.Flags().IntVar(&dnsSetTTL, "ttl", 300, "TTL (Time To Live) in seconds")
	dnsSetCmd.Flags().IntVar(&dnsSetPriority, "priority", 0, "Priority for MX records (required for MX)")

	// Mark priority as required for MX records - we'll validate this in the command
}
