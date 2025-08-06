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
	customTLDs       string
	exploreWide      bool
	exploreNoColor   bool
	exploreNoHeaders bool
)

// exploreCmd represents the explore command
var exploreCmd = &cobra.Command{
	Use:   "explore [domain-name]",
	Short: "Explore domain availability across multiple TLDs",
	Long: `Explore the availability of a domain name across multiple Top-Level Domains (TLDs).
Perfect for finding the right domain extension for your project or business.

Takes a base domain name (with or without TLD) and checks availability across
popular TLDs favored by indie hackers, startups, and developers. By default,
it checks popular extensions like .com, .org, .dev, .io, .co, and more.

You can customize the TLD list using the --tlds flag with a comma-separated list
or reference a file containing TLDs. The command automatically extracts the base
domain name if you provide a full domain.

Output options:
  --tlds        Comma-separated list of TLDs or @filename for file input
  --wide        Show additional columns (cost, expiry, error details)
  --json        Output results in JSON format
  --no-color    Disable colored output
  --no-headers  Don't show column headers

Examples:
  indietool domain explore kopitiam
  indietool domain explore kopitiam.dev
  indietool domain explore mycompany --json
  indietool domain explore startup --tlds com,org,dev,ai
  indietool domain explore webapp --tlds @tlds.txt
  indietool domain explore myapp --wide --no-color`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := strings.TrimSpace(strings.ToLower(args[0]))
		if input == "" {
			fmt.Fprintf(os.Stderr, "Domain name cannot be empty\n")
			os.Exit(1)
		}

		// Extract base domain name (remove TLD if present)
		baseDomain := domains.ExtractBaseDomain(input)

		// Determine which TLDs to use
		tlds := domains.PopularTLDs
		if customTLDs != "" {
			var err error
			tlds, err = domains.ParseTLDs(customTLDs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing TLDs: %v\n", err)
				os.Exit(1)
			}
		}

		// Generate domains to check
		domainList := make([]string, 0, len(tlds))
		for _, tld := range tlds {
			domainList = append(domainList, baseDomain+"."+tld)
		}

		// Search all domains concurrently
		results := domains.SearchDomainsConcurrent(domainList)

		// Organize results
		exploreResult := domains.OrganizeExploreResults(baseDomain, results)

		// Determine output format and render table
		format := domains.GetOutputFormat(jsonOutput, exploreWide)
		useColors := !exploreNoColor

		// Get table config and options
		tableConfig := domains.GetExploreTableConfig(useColors)
		options := domains.ExploreTableOptions(format, exploreWide, exploreNoColor, exploreNoHeaders, os.Stdout)

		// Convert results to table rows and render
		rows := exploreResult.ConvertToTableRows()
		table := output.NewTable(tableConfig, options)
		table.AddRows(rows)

		if err := table.RenderWithSummary(); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering table: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	domainCmd.AddCommand(exploreCmd)

	exploreCmd.Flags().StringVar(&customTLDs, "tlds", "", "Comma-separated list of TLDs or @filename for file input")

	// Output format flags (consistent with domains list command)
	exploreCmd.Flags().BoolVarP(&exploreWide, "wide", "w", false, "Show additional columns (cost, expiry, error details)")
	exploreCmd.Flags().BoolVar(&exploreNoHeaders, "no-headers", false, "Don't show column headers")
	exploreCmd.Flags().BoolVar(&exploreNoColor, "no-color", true, "Disable colored output")

	// Note: --json flag is inherited from global flags in root.go
}
