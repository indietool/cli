package domains

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/openrdap/rdap"
)

// DomainSearchResult represents the result of a domain availability search
type DomainSearchResult struct {
	Domain       string     `json:"domain"`
	Available    bool       `json:"available"`
	Status       string     `json:"status,omitempty"`
	Error        string     `json:"error,omitempty"`
	CreationDate *time.Time `json:"creation_date,omitempty"`
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`
	LastUpdated  *time.Time `json:"last_updated,omitempty"`
	LastChanged  *time.Time `json:"last_changed,omitempty"`
}

// PopularTLDs contains TLDs favored by indie hackers and small startups
var PopularTLDs = []string{
	"com", "net", "org", "dev", "app", "io", "co", "me", "ai", "sh",
	"ly", "gg", "cc", "tv", "fm", "tech", "online", "site", "xyz", "lol",
	"wtf", "cool", "fun", "live", "blog", "life", "world", "cloud", "digital", "email",
	"studio", "agency", "design", "media", "social", "team", "tools", "works", "tips", "guru",
	"ninja", "expert", "pro", "biz", "info", "name", "ventures", "solutions", "services", "consulting",
}

// SearchDomain checks the availability of a single domain using RDAP with WHOIS fallback
func SearchDomain(domain string) DomainSearchResult {
	// Try RDAP first
	result := searchDomainRDAP(domain)

	// If RDAP failed with an error, fallback to WHOIS
	if result.Error != "" {
		whoisResult := searchDomainWHOIS(domain)
		// If WHOIS succeeded, use it; otherwise keep the RDAP error
		if whoisResult.Error == "" {
			return whoisResult
		}
		// Keep the original RDAP error but note the fallback attempt
		result.Error = fmt.Sprintf("RDAP failed (%s), WHOIS fallback also failed (%s)", result.Error, whoisResult.Error)
	}

	return result
}

// searchDomainRDAP checks domain availability using RDAP
func searchDomainRDAP(domain string) DomainSearchResult {
	client := &rdap.Client{}

	resp, err := client.QueryDomain(domain)
	if err != nil {
		// Check if this is an ObjectDoesNotExist error (404), which indicates domain is available
		if clientErr, ok := err.(*rdap.ClientError); ok && clientErr.Type == rdap.ObjectDoesNotExist {
			return DomainSearchResult{
				Domain:    domain,
				Available: true,
				Status:    "available",
			}
		}

		return DomainSearchResult{
			Domain:    domain,
			Available: false,
			Error:     err.Error(),
		}
	}

	// Check if domain is available based on RDAP response
	available := false
	status := "registered"

	if resp.ObjectClassName == "" || len(resp.Status) == 0 {
		available = true
		status = "available"
	} else {
		// Check status values to determine availability
		for _, s := range resp.Status {
			if strings.Contains(strings.ToLower(s), "available") ||
				strings.Contains(strings.ToLower(s), "unassigned") {
				available = true
				status = "available"
				break
			}
		}

		if !available && len(resp.Status) > 0 {
			status = strings.Join(resp.Status, ", ")
		}
	}

	// Extract date information from RDAP response
	result := DomainSearchResult{
		Domain:    domain,
		Available: available,
		Status:    status,
	}

	// Parse events for date information
	if resp.Events != nil {
		for _, event := range resp.Events {
			eventDate := parseRDAPDate(event.Date)
			if eventDate == nil {
				continue
			}

			switch strings.ToLower(event.Action) {
			case "registration":
				result.CreationDate = eventDate
			case "expiration":
				result.ExpiryDate = eventDate
			case "last updated":
				result.LastUpdated = eventDate
			case "last changed":
				result.LastChanged = eventDate
			}
		}
	}

	return result
}

// searchDomainWHOIS checks domain availability using WHOIS as fallback
func searchDomainWHOIS(domain string) DomainSearchResult {
	// Query WHOIS data
	whoisRaw, err := whois.Whois(domain)
	if err != nil {
		return DomainSearchResult{
			Domain:    domain,
			Available: false,
			Error:     fmt.Sprintf("whois query failed: %v", err),
		}
	}

	// Parse WHOIS response
	whoisInfo, err := whoisparser.Parse(whoisRaw)
	if err != nil {
		// If parsing fails, try basic text analysis
		return analyzeRawWHOIS(domain, whoisRaw)
	}

	// Check parsed WHOIS data
	available := false
	status := "registered"

	// If no registrar or registrant, likely available
	if whoisInfo.Registrar == nil && whoisInfo.Registrant == nil {
		// Double-check with common "not found" patterns
		lowerRaw := strings.ToLower(whoisRaw)
		if containsNotFoundPattern(lowerRaw) {
			available = true
			status = "available"
		}
	} else if whoisInfo.Registrar != nil {
		// Domain is registered
		status = fmt.Sprintf("registered via %s", whoisInfo.Registrar.Name)
		if whoisInfo.Domain != nil && whoisInfo.Domain.ExpirationDate != "" {
			status += fmt.Sprintf(" (expires: %s)", whoisInfo.Domain.ExpirationDate)
		}
	}

	// Extract date information from WHOIS response
	result := DomainSearchResult{
		Domain:    domain,
		Available: available,
		Status:    status,
	}

	if whoisInfo.Domain != nil {
		if whoisInfo.Domain.CreatedDate != "" {
			if t := parseWHOISDate(whoisInfo.Domain.CreatedDate); t != nil {
				result.CreationDate = t
			}
		}
		if whoisInfo.Domain.ExpirationDate != "" {
			if t := parseWHOISDate(whoisInfo.Domain.ExpirationDate); t != nil {
				result.ExpiryDate = t
			}
		}
		if whoisInfo.Domain.UpdatedDate != "" {
			if t := parseWHOISDate(whoisInfo.Domain.UpdatedDate); t != nil {
				result.LastUpdated = t
			}
		}
	}

	return result
}

// analyzeRawWHOIS performs basic text analysis on raw WHOIS data when parsing fails
func analyzeRawWHOIS(domain, whoisRaw string) DomainSearchResult {
	lowerRaw := strings.ToLower(whoisRaw)

	// Check for common "not found" patterns
	if containsNotFoundPattern(lowerRaw) {
		return DomainSearchResult{
			Domain:    domain,
			Available: true,
			Status:    "available",
		}
	}

	// Check for registration indicators
	if strings.Contains(lowerRaw, "registrar:") ||
		strings.Contains(lowerRaw, "registrant:") ||
		strings.Contains(lowerRaw, "creation date:") ||
		strings.Contains(lowerRaw, "created:") {
		return DomainSearchResult{
			Domain:    domain,
			Available: false,
			Status:    "registered (via whois)",
		}
	}

	// If we can't determine, assume it's registered to be safe
	return DomainSearchResult{
		Domain:    domain,
		Available: false,
		Status:    "unknown (whois inconclusive)",
	}
}

// containsNotFoundPattern checks for common "domain not found" patterns in WHOIS data
func containsNotFoundPattern(lowerRaw string) bool {
	notFoundPatterns := []string{
		"no match",
		"not found",
		"no entries found",
		"no data found",
		"domain not found",
		"not registered",
		"available for registration",
		"no matching record",
		"status: free",
		"status: available",
		"no such domain",
	}

	for _, pattern := range notFoundPatterns {
		if strings.Contains(lowerRaw, pattern) {
			return true
		}
	}

	return false
}

// parseRDAPDate parses RDAP date strings to time.Time
func parseRDAPDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// Try parsing common RDAP date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

// parseWHOISDate parses WHOIS date strings to time.Time
func parseWHOISDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// Try parsing common WHOIS date formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"2-Jan-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

// SearchDomainsConcurrent checks multiple domains concurrently
func SearchDomainsConcurrent(domains []string) []DomainSearchResult {
	results := make([]DomainSearchResult, len(domains))
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		go func(index int, d string) {
			defer wg.Done()
			results[index] = SearchDomain(d)
		}(i, domain)
	}

	wg.Wait()
	return results
}

// ExtractBaseDomain removes the TLD from a domain if present
func ExtractBaseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 1 {
		// Check if the last part is a known TLD
		lastPart := parts[len(parts)-1]
		for _, tld := range PopularTLDs {
			if lastPart == tld {
				// Remove the TLD and return the base domain
				return strings.Join(parts[:len(parts)-1], ".")
			}
		}
	}
	return domain
}

// ParseTLDs parses TLD input (comma-separated or @filename)
func ParseTLDs(input string) ([]string, error) {
	if strings.HasPrefix(input, "@") {
		// Read from file
		filename := input[1:]
		return readTLDsFromFile(filename)
	}

	// Parse comma-separated list
	tlds := strings.Split(input, ",")
	result := make([]string, 0, len(tlds))
	for _, tld := range tlds {
		tld = strings.TrimSpace(tld)
		if tld != "" {
			// Remove leading dot if present
			tld = strings.TrimPrefix(tld, ".")
			result = append(result, tld)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid TLDs found in input")
	}

	return result, nil
}

// readTLDsFromFile reads TLDs from a newline-delimited file
func readTLDsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	var tlds []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tld := strings.TrimSpace(scanner.Text())
		if tld != "" && !strings.HasPrefix(tld, "#") {
			// Remove leading dot if present
			tld = strings.TrimPrefix(tld, ".")
			tlds = append(tlds, tld)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	if len(tlds) == 0 {
		return nil, fmt.Errorf("no valid TLDs found in file %s", filename)
	}

	return tlds, nil
}

