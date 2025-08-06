package cmd

import (
	"context"
	"fmt"
	"indietool/cli/dns"
	"indietool/cli/output"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// DNS list command no longer needs its own flags - uses parent flags

var dnsListCmd = &cobra.Command{
	Use:   "list <domain>",
	Short: "List DNS records for a domain",
	Long: `List all DNS records for a domain from the DNS hosting provider.
Automatically detects the DNS provider or use --provider to specify.

Examples:
  indietool dns list example.com
  indietool dns list example.com --provider cloudflare
  indietool dns list example.com --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		domain := args[0]

		// Get DNS manager from parent command
		dnsManager := GetDNSManager()
		if dnsManager == nil {
			handleDNSError(fmt.Errorf("DNS manager not initialized"))
			return
		}

		// List DNS records using parent provider flag
		records, detectionResult, err := dnsManager.ListRecords(context.TODO(), domain, GetDNSProvider())
		if err != nil {
			handleDNSError(fmt.Errorf("failed to list DNS records: %w", err))
			return
		}

		// Log detection result for debugging
		if detectionResult != nil {
			if detectionResult.Provider != "" {
				log.Debugf("Detected DNS provider: %s (confidence: %s)", detectionResult.Provider, detectionResult.Confidence)
			} else {
				log.Debugf("Failed to detect DNS provider: %s", detectionResult.Error)
			}
		}

		// Output records
		// if jsonOutput {
		// 	output.PrintJSON(map[string]interface{}{"records": records})
		// } else {
		// 	outputDNSRecordsTable(records, domain)
		// }
		outputDNSRecordsTable(records, domain)
	},
}

func init() {
	dnsCmd.AddCommand(dnsListCmd)
	// Flags are now handled by parent dns command
}

func outputDNSRecordsTable(records []dns.Record, domain string) {
	if len(records) == 0 {
		fmt.Printf("No DNS records found for domain: %s\n", domain)
		return
	}

	// Get output flags from parent DNS command
	wide, noHeaders, noColor := GetDNSOutputFlags()

	// Create table configuration
	options := output.TableOptions{
		Wide:      wide,
		NoHeaders: noHeaders,
		NoColor:   noColor,
		Format:    output.FormatTable,
		Writer:    os.Stdout,
	}

	if wide {
		options.Format = output.FormatWide
	}

	config := output.TableConfig{
		DefaultColumns: []output.Column{
			{Name: "TYPE", JSONPath: "type", Width: 10},
			{Name: "NAME", JSONPath: "name"},
			{Name: "CONTENT", JSONPath: "content"},
		},

		WideColumns: []output.Column{
			{Name: "TTL", JSONPath: "ttl"},
			{Name: "PRIORITY", JSONPath: "priority"},
			{Name: "ID", JSONPath: "id"},
		},
	}

	table := output.NewTable(config, options)

	// Add rows
	for _, record := range records {
		// Determine display name with proxy indicator for Cloudflare
		displayName := record.Name
		if record.Proxied != nil && *record.Proxied {
			displayName = "☁️" + record.Name
		}

		// var row []interface{}
		var row map[string]any
		if wide {
			priority := ""
			if record.Priority != nil {
				priority = fmt.Sprintf("%d", *record.Priority)
			}

			row = map[string]any{
				"type":     record.Type,
				"name":     displayName,
				"content":  record.Content,
				"ttl":      record.TTL,
				"priority": priority,
				"id":       record.ID,
			}
		} else {
			row = map[string]any{
				"type":    record.Type,
				"name":    displayName,
				"content": record.Content,
			}
		}
		table.AddRow(row)
	}

	// Show summary
	if !noHeaders {
		fmt.Printf("\nDNS Records for %s (%d total)\n\n", domain, len(records))
	}

	if err := table.Render(); err != nil {
		handleDNSError(fmt.Errorf("failed to render table: %w", err))
	}
}

func handleDNSError(err error) {
	log.Errorf("Error: %v", err)
}
