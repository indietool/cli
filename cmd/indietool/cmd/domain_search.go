/*
Copyright © 2025 

*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"indietool/cli/domains"
	"indietool/cli/output"
)



// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [domain...]",
	Short: "Search for domain availability using RDAP",
	Long: `Search for domain availability using the Registration Data Access Protocol (RDAP).
Takes one or more domain names as arguments and checks their registration status.

Examples:
  indietool domain search example.com
  indietool domain search example.com google.com --json
  indietool domain search mydomain.org anotherdomain.net`,
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
		
		results := domains.SearchDomainsConcurrent(domainList)
		
		if jsonOutput {
			output.OutputSearchJSON(results)
		} else {
			output.OutputSearchHuman(results)
		}
	},
}





func init() {
	domainCmd.AddCommand(searchCmd)
	
}
