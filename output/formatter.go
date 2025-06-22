package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"indietool/cli/domains"
)

// OutputSearchJSON outputs domain search results in JSON format
func OutputSearchJSON(results []domains.DomainSearchResult) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// OutputSearchHuman outputs domain search results in human-readable format
func OutputSearchHuman(results []domains.DomainSearchResult) {
	fmt.Printf("Domain Availability Search Results\n")
	fmt.Printf("==================================\n\n")
	
	for _, result := range results {
		fmt.Printf("Domain: %s\n", result.Domain)
		
		if result.Error != "" {
			fmt.Printf("  Status: ERROR - %s\n", result.Error)
		} else {
			if result.Available {
				fmt.Printf("  Status: ✓ AVAILABLE\n")
			} else {
				fmt.Printf("  Status: ✗ NOT AVAILABLE\n")
			}
			
			if result.Status != "" {
				fmt.Printf("  Details: %s\n", result.Status)
			}
		}
		
		fmt.Println()
	}
}

// OutputExploreJSON outputs domain exploration results in JSON format
func OutputExploreJSON(result domains.ExploreResult) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// OutputExploreHuman outputs domain exploration results in human-readable format
func OutputExploreHuman(result domains.ExploreResult) {
	fmt.Printf("Domain Exploration Results for \"%s\"\n", result.BaseDomain)
	fmt.Printf(strings.Repeat("=", 40+len(result.BaseDomain)) + "\n\n")
	
	// Summary
	totalChecked := len(result.Results)
	fmt.Printf("Summary: %d domains checked\n", totalChecked)
	fmt.Printf("  ✓ Available: %d\n", len(result.Available))
	fmt.Printf("  ✗ Taken: %d\n", len(result.Taken))
	if len(result.Errors) > 0 {
		fmt.Printf("  ⚠ Errors: %d\n", len(result.Errors))
	}
	fmt.Println()
	
	// Available domains
	if len(result.Available) > 0 {
		fmt.Printf("✓ AVAILABLE DOMAINS:\n")
		for _, domain := range result.Available {
			fmt.Printf("  %s\n", domain.Domain)
		}
		fmt.Println()
	}
	
	// Taken domains
	if len(result.Taken) > 0 {
		fmt.Printf("✗ TAKEN DOMAINS:\n")
		for _, domain := range result.Taken {
			status := domain.Status
			if status == "" {
				status = "registered"
			}
			fmt.Printf("  %s (%s)\n", domain.Domain, status)
		}
		fmt.Println()
	}
	
	// Errors
	if len(result.Errors) > 0 {
		fmt.Printf("⚠ ERRORS:\n")
		for _, domain := range result.Errors {
			fmt.Printf("  %s - %s\n", domain.Domain, domain.Error)
		}
		fmt.Println()
	}
}