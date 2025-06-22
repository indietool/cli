package domains

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/openrdap/rdap"
)

// DomainSearchResult represents the result of a domain availability search
type DomainSearchResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PopularTLDs contains TLDs favored by indie hackers and small startups
var PopularTLDs = []string{
	"com", "net", "org", "dev", "app", "io", "co", "me", "ai", "sh",
	"ly", "gg", "cc", "tv", "fm", "tech", "online", "site", "xyz", "lol",
	"wtf", "cool", "fun", "live", "blog", "life", "world", "cloud", "digital", "email",
	"studio", "agency", "design", "media", "social", "team", "tools", "works", "tips", "guru",
	"ninja", "expert", "pro", "biz", "info", "name", "ventures", "solutions", "services", "consulting",
}

// SearchDomain checks the availability of a single domain using RDAP
func SearchDomain(domain string) DomainSearchResult {
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
	
	return DomainSearchResult{
		Domain:    domain,
		Available: available,
		Status:    status,
	}
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