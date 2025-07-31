package dns

import (
	"fmt"
	"strings"
)

// Record represents a DNS record
type Record struct {
	ID       string `json:"id,omitempty"`       // Provider-specific record ID
	Type     string `json:"type"`               // A, AAAA, CNAME, MX, etc.
	Name     string `json:"name"`               // Record name (@, www, subdomain)
	Content  string `json:"content"`            // Record value (IP, target, etc.)
	TTL      int    `json:"ttl"`                // Time to live in seconds
	Priority *int   `json:"priority,omitempty"` // For MX records
	Proxied  *bool  `json:"proxied,omitempty"`  // Cloudflare-specific proxy status
}

// ValidateRecordType checks if the record type is supported
func ValidateRecordType(recordType string) error {
	validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "PTR"}
	recordType = strings.ToUpper(recordType)

	for _, validType := range validTypes {
		if recordType == validType {
			return nil
		}
	}

	return fmt.Errorf("unsupported record type: %s (supported: %s)", recordType, strings.Join(validTypes, ", "))
}

// NormalizeName normalizes the record name for consistency
func NormalizeName(name, domain string) string {
	name = strings.TrimSpace(name)

	// Handle root domain cases
	if name == "" || name == "@" {
		return "@"
	}

	// Remove trailing domain if present (e.g., "www.example.com" -> "www")
	if strings.HasSuffix(name, "."+domain) {
		name = strings.TrimSuffix(name, "."+domain)
	}

	// Remove trailing dot if present
	name = strings.TrimSuffix(name, ".")

	return name
}

// FullName returns the full DNS name (subdomain + domain)
func (r *Record) FullName(domain string) string {
	if r.Name == "@" || r.Name == "" {
		return domain
	}
	return r.Name + "." + domain
}

// String returns a human-readable representation of the record
func (r *Record) String() string {
	return fmt.Sprintf("%s %s %s (TTL: %d)", r.Name, r.Type, r.Content, r.TTL)
}
