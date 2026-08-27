package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/indietool/cli/dns"
	"github.com/indietool/cli/indietool"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	dnsDeleteProvider string
	dnsDeleteForce    bool
	dnsDeleteType     string
	dnsDeleteID       string
	dnsDeleteDryRun   bool
)

type dnsDeleteResultJSON struct {
	Status   string       `json:"status"` // "success", "dry-run", "cancelled"
	DryRun   bool         `json:"dry_run"`
	Domain   string       `json:"domain"`
	Provider string       `json:"provider,omitempty"`
	Deleted  int          `json:"deleted"`
	Records  []dns.Record `json:"records"`
	Message  string       `json:"message"`
	Errors   []string     `json:"errors,omitempty"`
}

var dnsDeleteCmd = &cobra.Command{
	Use:   "delete <domain> <name> [type]",
	Short: "Delete DNS records by name",
	Long: `Delete DNS records from the specified domain by record name.
If no type is specified, all records for that name will be deleted.
Use --id to target a specific record when multiple records have the same name.

Examples:
  indietool dns delete example.com www A
  indietool dns delete example.com api --type CNAME
  indietool dns delete example.com test --id 123abc
  indietool dns delete example.com @ MX --force
  indietool dns delete example.com subdomain
  indietool dns delete example.com www A --dry-run
  indietool dns delete example.com www A --json`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runDNSDelete,
}

func init() {
	dnsDeleteCmd.Flags().StringVar(&dnsDeleteProvider, "provider", "", "DNS provider to use (cloudflare, namecheap, porkbun, godaddy, thelittlehost)")
	dnsDeleteCmd.Flags().BoolVarP(&dnsDeleteForce, "force", "f", false, "Delete without confirmation")
	dnsDeleteCmd.Flags().StringVar(&dnsDeleteType, "type", "", "Record type filter")
	dnsDeleteCmd.Flags().StringVar(&dnsDeleteID, "id", "", "Record ID to delete (use with --wide to find IDs)")
	dnsDeleteCmd.Flags().BoolVar(&dnsDeleteDryRun, "dry-run", false, "Show planned deletions without applying them")

	// Add to parent dns command
	dnsCmd.AddCommand(dnsDeleteCmd)
}

func runDNSDelete(cmd *cobra.Command, args []string) error {
	domain := args[0]
	name := args[1]

	// Determine record type (from positional arg or flag)
	recordType := ""
	if len(args) == 3 {
		recordType = args[2]
	} else if dnsDeleteType != "" {
		recordType = dnsDeleteType
	}

	log.Debugf("Deleting DNS records for domain=%s, name=%s, type=%s, id=%s", domain, name, recordType, dnsDeleteID)

	// Find records to delete
	recordsToDelete, resolvedProvider, err := findRecordsForDeletion(domain, name, recordType, dnsDeleteID)
	if err != nil {
		return err
	}

	// Show DNS provider (human mode only)
	if !jsonOutput && resolvedProvider != "" {
		fmt.Printf("DNS Provider: %s\n", resolvedProvider)
	}

	if len(recordsToDelete) == 0 {
		// Build descriptive error message based on filters used
		var filters []string
		if recordType != "" {
			filters = append(filters, fmt.Sprintf("type '%s'", recordType))
		}
		if dnsDeleteID != "" {
			filters = append(filters, fmt.Sprintf("ID '%s'", dnsDeleteID))
		}

		errorMsg := fmt.Sprintf("no DNS records found for '%s'", name)
		if len(filters) > 0 {
			errorMsg += " with " + strings.Join(filters, " and ")
		}
		return fmt.Errorf("%s", errorMsg)
	}

	if dnsDeleteDryRun {
		msg := fmt.Sprintf("Would delete %d DNS record(s)", len(recordsToDelete))
		if jsonOutput {
			return printJSON(dnsDeleteResultJSON{
				Status:   "dry-run",
				DryRun:   true,
				Domain:   domain,
				Provider: resolvedProvider,
				Deleted:  0,
				Records:  recordsToDelete,
				Message:  msg,
			})
		}
		fmt.Printf("[dry-run] %s:\n", msg)
		for _, record := range recordsToDelete {
			fmt.Printf("  - %s %s %s\n", record.Name, record.Type, record.Content)
		}
		return nil
	}

	// Skip confirmation in JSON mode (scriptable) or with --force
	if !dnsDeleteForce && !jsonOutput {
		if !confirmDeletion(recordsToDelete) {
			fmt.Println("Delete cancelled")
			return nil
		}
	}

	// Execute deletions
	return executeDeletions(domain, resolvedProvider, recordsToDelete)
}

func findRecordsForDeletion(domain, name, recordType, recordID string) ([]dns.Record, string, error) {
	// Get the global provider registry
	registry := GetProviderRegistry()
	if registry == nil {
		return nil, "", fmt.Errorf("provider registry not initialized")
	}

	// Get DNS providers from registry
	dnsProviders := indietool.GetProviders[dns.Provider](registry)
	if len(dnsProviders) == 0 {
		return nil, "", fmt.Errorf("no DNS providers configured")
	}

	// Create DNS manager
	manager := dns.NewManager(dnsProviders)

	providerFlag := dnsDeleteProvider
	if providerFlag == "" {
		providerFlag = GetDNSProvider()
	}

	// List all records for the domain
	records, detectionResult, err := manager.ListRecords(context.Background(), domain, providerFlag)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Resolve provider name from flag or detection
	resolvedProvider := providerFlag
	if detectionResult != nil {
		if detectionResult.Provider != "" {
			log.Debugf("Detected DNS provider: %s (confidence: %s)", detectionResult.Provider, detectionResult.Confidence)
			if resolvedProvider == "" {
				resolvedProvider = detectionResult.Provider
			}
		} else {
			log.Debugf("Failed to detect DNS provider: %s", detectionResult.Error)
		}
	}

	// Filter by name, type, and ID
	var matches []dns.Record
	for _, record := range records {
		// Exact name match
		if record.Name != name {
			continue
		}

		// Type filter (if specified) - case insensitive
		if recordType != "" && strings.ToUpper(record.Type) != strings.ToUpper(recordType) {
			continue
		}

		// ID filter (if specified) - exact match
		if recordID != "" && record.ID != recordID {
			continue
		}

		matches = append(matches, record)
	}

	return matches, resolvedProvider, nil
}

func confirmDeletion(records []dns.Record) bool {
	if len(records) == 1 {
		record := records[0]
		fmt.Printf("Found DNS record to delete:\n")
		fmt.Printf("  Name: %s\n", record.Name)
		fmt.Printf("  Type: %s\n", record.Type)
		fmt.Printf("  Content: %s\n", record.Content)
		fmt.Printf("  TTL: %d\n", record.TTL)
		if record.Priority != nil {
			fmt.Printf("  Priority: %d\n", *record.Priority)
		}
		if record.Proxied != nil && *record.Proxied {
			fmt.Printf("  Proxied: Yes ☁️\n")
		}
		if record.ID != "" {
			fmt.Printf("  ID: %s\n", record.ID)
		}
		fmt.Printf("\nDelete this record? [y/N]: ")
	} else {
		fmt.Printf("Found %d DNS records for '%s':\n", len(records), records[0].Name)
		for i, record := range records {
			proxied := ""
			if record.Proxied != nil && *record.Proxied {
				proxied = " ☁️"
			}
			priority := ""
			if record.Priority != nil {
				priority = fmt.Sprintf(" (Priority: %d)", *record.Priority)
			}
			id := ""
			if record.ID != "" {
				id = fmt.Sprintf(" [ID: %s]", record.ID)
			}
			fmt.Printf("  [%d] %s %s %s%s (TTL: %d)%s%s\n",
				i+1, record.Name, record.Type, record.Content, proxied, record.TTL, priority, id)
		}
		fmt.Printf("\nDelete all %d records? [y/N]: ", len(records))
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func executeDeletions(domain, resolvedProvider string, records []dns.Record) error {
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

	manager := dns.NewManager(dnsProviders)

	providerFlag := dnsDeleteProvider
	if providerFlag == "" {
		providerFlag = GetDNSProvider()
	}

	// Delete each record
	var errors []string
	successCount := 0
	var deleted []dns.Record

	for _, record := range records {
		err := manager.DeleteRecord(context.Background(), domain, providerFlag, record.ID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete record %s %s: %v",
				record.Name, record.Type, err))
			continue
		}
		successCount++
		deleted = append(deleted, record)
		log.Debugf("Successfully deleted DNS record: %s %s %s", record.Name, record.Type, record.Content)
	}

	if jsonOutput {
		status := "success"
		msg := fmt.Sprintf("Deleted %d DNS record(s)", successCount)
		if len(errors) > 0 {
			status = "error"
			msg = fmt.Sprintf("failed to delete %d records", len(errors))
		}
		if err := printJSON(dnsDeleteResultJSON{
			Status:   status,
			DryRun:   false,
			Domain:   domain,
			Provider: resolvedProvider,
			Deleted:  successCount,
			Records:  deleted,
			Message:  msg,
			Errors:   errors,
		}); err != nil {
			return err
		}
		if len(errors) > 0 {
			return fmt.Errorf("failed to delete %d records", len(errors))
		}
		return nil
	}

	// Report results
	if len(errors) > 0 {
		for _, errMsg := range errors {
			fmt.Printf("✗ %s\n", errMsg)
		}
		if successCount > 0 {
			fmt.Printf("✓ Deleted %d of %d DNS records\n", successCount, len(records))
		}
		return fmt.Errorf("failed to delete %d records", len(errors))
	}

	// Success message
	if len(records) == 1 {
		record := records[0]
		proxied := ""
		if record.Proxied != nil && *record.Proxied {
			proxied = " ☁️"
		}
		fmt.Printf("✓ Deleted DNS record: %s %s %s%s\n",
			record.Name, record.Type, record.Content, proxied)
	} else {
		fmt.Printf("✓ Deleted %d DNS records for '%s'\n",
			len(records), records[0].Name)
	}

	return nil
}
