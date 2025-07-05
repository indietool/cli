package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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

// OutputDomainListJSON outputs domain list results in JSON format
func OutputDomainListJSON(result *domains.DomainListResult) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// OutputDomainListHuman outputs domain list results in human-readable format
func OutputDomainListHuman(result *domains.DomainListResult) {
	fmt.Printf("Managed Domains Overview\n")
	fmt.Printf("========================\n\n")

	// Summary stats
	fmt.Printf("Summary: %d domains total\n", result.Summary.Total)
	fmt.Printf("  ✓ Healthy: %d\n", result.Summary.Healthy)
	fmt.Printf("  ⚠ Warning: %d\n", result.Summary.Warning)
	fmt.Printf("  🚨 Critical: %d\n", result.Summary.Critical)
	if result.Summary.Expired > 0 {
		fmt.Printf("  💀 Expired: %d\n", result.Summary.Expired)
	}
	fmt.Printf("Last synced: %s\n\n", result.LastSynced.Format("2006-01-02 15:04:05"))

	// Domain table
	for _, domain := range result.Domains {
		statusIcon := getStatusIcon(domain.Status)
		daysUntilExpiry := calculateDaysUntilExpiry(domain.ExpiryDate)

		fmt.Printf("%s %s (%s)\n", statusIcon, domain.Name, domain.Provider)
		fmt.Printf("    Expires: %s (%s)\n",
			domain.ExpiryDate.Format("2006-01-02"),
			formatExpiryCountdown(daysUntilExpiry))
		fmt.Printf("    Auto-renewal: %s\n", formatBool(domain.AutoRenewal))
		fmt.Printf("    Nameservers: %s\n", strings.Join(domain.Nameservers, ", "))
		fmt.Println()
	}
}

// getStatusIcon returns an emoji icon for the domain status
func getStatusIcon(status domains.DomainStatus) string {
	switch status {
	case domains.StatusHealthy:
		return "✓"
	case domains.StatusWarning:
		return "⚠"
	case domains.StatusCritical:
		return "🚨"
	case domains.StatusExpired:
		return "💀"
	default:
		return "?"
	}
}

// calculateDaysUntilExpiry calculates days until domain expiry
func calculateDaysUntilExpiry(expiryDate time.Time) int {
	return int(expiryDate.Sub(time.Now()).Hours() / 24)
}

// formatExpiryCountdown formats the expiry countdown in human-readable form
func formatExpiryCountdown(days int) string {
	if days < 0 {
		return fmt.Sprintf("%d days ago", -days)
	} else if days == 0 {
		return "today"
	} else if days == 1 {
		return "tomorrow"
	} else {
		return fmt.Sprintf("in %d days", days)
	}
}

// formatBool formats a boolean value for display
func formatBool(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

