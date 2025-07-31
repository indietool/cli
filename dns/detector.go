package dns

import (
	"fmt"
	"net"
	"strings"
)

// nameserverPatterns maps provider names to their nameserver patterns
var nameserverPatterns = map[string][]string{
	"cloudflare": {
		".ns.cloudflare.com",
		"cloudflare.com",
	},
	"namecheap": {
		".registrar-servers.com",
		".namecheaphosting.com",
		".namecheap.com",
	},
	"porkbun": {
		".porkbun.com",
		"curitiba.porkbun.com",
		"fortaleza.porkbun.com",
		"maceio.porkbun.com",
		"salvador.porkbun.com",
	},
	"godaddy": {
		".domaincontrol.com",
		".godaddy.com",
	},
}

// DetectorResult contains the result of DNS provider detection
type DetectorResult struct {
	Provider    string   `json:"provider"`
	Confidence  string   `json:"confidence"` // "high", "medium", "low"
	Nameservers []string `json:"nameservers"`
	Error       string   `json:"error,omitempty"`
}

// DetectProvider attempts to detect the DNS hosting provider for a domain
func DetectProvider(domain string) (*DetectorResult, error) {
	// Query nameservers for the domain
	nameservers, err := net.LookupNS(domain)
	if err != nil {
		return &DetectorResult{
			Error: fmt.Sprintf("failed to lookup nameservers: %v", err),
		}, fmt.Errorf("failed to lookup nameservers for %s: %w", domain, err)
	}

	// Convert to string slice for easier processing
	nsHosts := make([]string, len(nameservers))
	for i, ns := range nameservers {
		nsHosts[i] = strings.ToLower(strings.TrimSuffix(ns.Host, "."))
	}

	result := &DetectorResult{
		Nameservers: nsHosts,
	}

	// Check nameserver patterns
	for _, nsHost := range nsHosts {
		provider := matchNameserverPattern(nsHost)
		if provider != "" {
			result.Provider = provider
			result.Confidence = "high"
			return result, nil
		}
	}

	// If no exact match found, try partial matching with lower confidence
	for _, nsHost := range nsHosts {
		for providerName, patterns := range nameserverPatterns {
			for _, pattern := range patterns {
				if strings.Contains(nsHost, strings.TrimPrefix(pattern, ".")) {
					result.Provider = providerName
					result.Confidence = "medium"
					return result, nil
				}
			}
		}
	}

	return result, fmt.Errorf("unable to detect DNS provider for %s (nameservers: %s)", domain, strings.Join(nsHosts, ", "))
}

// matchNameserverPattern checks if a nameserver matches any known provider pattern
func matchNameserverPattern(nameserver string) string {
	for providerName, patterns := range nameserverPatterns {
		for _, pattern := range patterns {
			if strings.HasSuffix(nameserver, pattern) || nameserver == strings.TrimPrefix(pattern, ".") {
				return providerName
			}
		}
	}
	return ""
}

// GetSupportedProviders returns a list of providers that can be auto-detected
func GetSupportedProviders() []string {
	providers := make([]string, 0, len(nameserverPatterns))
	for provider := range nameserverPatterns {
		providers = append(providers, provider)
	}
	return providers
}

// GetProviderPatterns returns the nameserver patterns for a specific provider
func GetProviderPatterns(provider string) ([]string, bool) {
	patterns, exists := nameserverPatterns[provider]
	return patterns, exists
}
