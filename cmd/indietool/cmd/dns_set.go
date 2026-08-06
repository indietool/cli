package cmd

import (
	"context"
	"fmt"

	"github.com/indietool/cli/dns"
	"github.com/indietool/cli/indietool"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	dnsSetProvider string
	dnsSetTTL      int
	dnsSetPriority int
	dnsSetDryRun   bool
)

type dnsSetResultJSON struct {
	Status   string     `json:"status"` // "success" or "dry-run"
	DryRun   bool       `json:"dry_run"`
	Domain   string     `json:"domain"`
	Provider string     `json:"provider,omitempty"`
	Record   dns.Record `json:"record"`
	Message  string     `json:"message"`
}

var dnsSetCmd = &cobra.Command{
	Use:   "set <domain> <name> <type> <value>",
	Short: "Set a DNS record for a domain",
	Long: `Set or update a DNS record for a domain.
Automatically detects the DNS provider or use --provider to specify.

Examples:
  indietool dns set example.com www A 192.168.1.1
  indietool dns set example.com @ MX "10 mail.example.com"
  indietool dns set example.com --provider cloudflare www CNAME "other.example.com"
  indietool dns set example.com _dmarc TXT "v=DMARC1; p=reject"
  indietool dns set example.com www A 192.168.1.1 --dry-run
  indietool dns set example.com www A 192.168.1.1 --json`,
	Args: cobra.ExactArgs(4),
	RunE: runDNSSet,
}

func init() {
	dnsCmd.AddCommand(dnsSetCmd)

	// Provider flag
	dnsSetCmd.Flags().StringVar(&dnsSetProvider, "provider", "", "DNS provider to use (cloudflare, namecheap, porkbun, godaddy, thelittlehost)")

	// DNS record options
	dnsSetCmd.Flags().IntVar(&dnsSetTTL, "ttl", 300, "TTL (Time To Live) in seconds")
	dnsSetCmd.Flags().IntVar(&dnsSetPriority, "priority", 0, "Priority for MX records (required for MX)")
	dnsSetCmd.Flags().BoolVar(&dnsSetDryRun, "dry-run", false, "Show planned change without applying it")
}

func runDNSSet(cmd *cobra.Command, args []string) error {
	domain := args[0]
	name := args[1]
	recordType := args[2]
	value := args[3]

	// Get the global provider registry
	registry := GetProviderRegistry()
	if registry == nil {
		return fmt.Errorf("provider registry not initialized")
	}

	// Get DNS providers from registry
	dnsProviders := indietool.GetProviders[dns.Provider](registry)
	if len(dnsProviders) == 0 {
		return fmt.Errorf("no DNS providers configured")
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

	// Prefer explicit flag, fall back to parent dns --provider
	providerFlag := dnsSetProvider
	if providerFlag == "" {
		providerFlag = GetDNSProvider()
	}

	resolvedProvider := providerFlag

	if dnsSetDryRun {
		// Best-effort provider detection for display without mutating
		if resolvedProvider == "" {
			if result, err := dns.DetectProvider(domain); err == nil && result != nil && result.Provider != "" {
				resolvedProvider = result.Provider
				log.Debugf("Detected DNS provider: %s (confidence: %s)", result.Provider, result.Confidence)
			}
		}

		msg := fmt.Sprintf("Would set DNS record %s %s %s", name, recordType, value)
		if jsonOutput {
			return printJSON(dnsSetResultJSON{
				Status:   "dry-run",
				DryRun:   true,
				Domain:   domain,
				Provider: resolvedProvider,
				Record:   record,
				Message:  msg,
			})
		}
		if resolvedProvider != "" {
			fmt.Printf("DNS Provider: %s\n", resolvedProvider)
		}
		fmt.Printf("[dry-run] %s (TTL: %d)\n", msg, record.TTL)
		if record.Priority != nil {
			fmt.Printf("  Priority: %d\n", *record.Priority)
		}
		return nil
	}

	// Set DNS record
	detectionResult, err := dnsManager.SetRecord(context.TODO(), domain, providerFlag, record)
	if err != nil {
		return fmt.Errorf("failed to set DNS record: %w", err)
	}

	// Resolve provider name from flag or detection
	if detectionResult != nil {
		if detectionResult.Provider != "" {
			log.Debugf("Used DNS provider: %s (confidence: %s)", detectionResult.Provider, detectionResult.Confidence)
			if resolvedProvider == "" {
				resolvedProvider = detectionResult.Provider
			}
		} else {
			log.Debugf("Failed to detect DNS provider: %s", detectionResult.Error)
		}
	}

	msg := fmt.Sprintf("Successfully set DNS record %s %s %s", name, recordType, value)
	if jsonOutput {
		return printJSON(dnsSetResultJSON{
			Status:   "success",
			DryRun:   false,
			Domain:   domain,
			Provider: resolvedProvider,
			Record:   record,
			Message:  msg,
		})
	}

	// Success message
	if resolvedProvider != "" {
		fmt.Printf("DNS Provider: %s\n", resolvedProvider)
	}
	fmt.Printf("%s\n", msg)
	return nil
}
