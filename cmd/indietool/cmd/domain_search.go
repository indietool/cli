package cmd

import (
	"fmt"
	"indietool/cli/domains"
	"indietool/cli/output"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchWide      bool
	searchNoColor   bool
	searchNoHeaders bool
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [domain...]",
	Short: "Check availability of specific domain names",
	Long: `Check the availability and registration status of one or more specific domain names
using the Registration Data Access Protocol (RDAP). This provides reliable, real-time
information about domain registration status directly from registries.

The command accepts multiple domain names and checks them concurrently for faster results.
Results include availability status, registrar information, and registration details.

Output options:
  --wide        Show additional columns (registrar, cost, expiry, error details)
  --json        Output results in JSON format
  --no-color    Disable colored output
  --no-headers  Don't show column headers

Examples:
  indietool domain search example.com
  indietool domain search example.com google.com --json
  indietool domain search mydomain.org anotherdomain.net --wide
  indietool domain search startup.dev indie.co --no-color`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		domainList := make([]string, 0, len(args))
		for _, domain := range args {
			domain = strings.TrimSpace(strings.ToLower(domain))
			if domain != "" {
				domainList = append(domainList, domain)
			}
		}

		if len(domainList) == 0 {
			fmt.Fprintf(os.Stderr, "No valid domains provided\n")
			os.Exit(1)
		}

		// Search all domains concurrently
		results := domains.SearchDomainsConcurrent(domainList)

		// Determine output format and render table
		format := domains.GetOutputFormat(jsonOutput, searchWide)
		useColors := !searchNoColor

		// Get table config and options
		tableConfig := domains.GetSearchTableConfig(useColors)
		options := domains.SearchTableOptions(format, searchWide, searchNoColor, searchNoHeaders, os.Stdout)

		// Convert results to table rows and render
		rows := domains.ConvertSearchResultsToTableRows(results)
		table := output.NewTable(tableConfig, options)
		table.AddRows(rows)

		if err := table.RenderWithSummary(); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering table: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	domainCmd.AddCommand(searchCmd)

	// Output format flags (consistent with domains list and explore commands)
	searchCmd.Flags().BoolVarP(&searchWide, "wide", "w", false, "Show additional columns (registrar, cost, expiry, error details)")
	searchCmd.Flags().BoolVar(&searchNoHeaders, "no-headers", false, "Don't show column headers")
	searchCmd.Flags().BoolVar(&searchNoColor, "no-color", true, "Disable colored output")

	// Note: --json flag is inherited from global flags in root.go
}
