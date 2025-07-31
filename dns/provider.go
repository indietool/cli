package dns

import (
	"context"
)

// Provider defines the interface for DNS operations
type Provider interface {
	// Name returns the provider name (e.g., "cloudflare")
	Name() string

	// ListRecords retrieves all DNS records for a domain
	ListRecords(ctx context.Context, domain string) ([]Record, error)

	// SetRecord creates or updates a DNS record
	SetRecord(ctx context.Context, domain string, record Record) error

	// DeleteRecord removes a DNS record by ID
	DeleteRecord(ctx context.Context, domain, recordID string) error

	// GetRecord retrieves a specific DNS record by name and type
	GetRecord(ctx context.Context, domain, name, recordType string) (*Record, error)
}

// ProviderCapabilities defines optional capabilities a provider may support
type ProviderCapabilities struct {
	SupportsProxy    bool // Cloudflare proxy mode
	SupportsPriority bool // MX record priorities
	SupportsWildcard bool // Wildcard records
	SupportsTTLRange bool // Custom TTL ranges
	MinTTL           int  // Minimum TTL value
	MaxTTL           int  // Maximum TTL value
}

// CapableProvider extends Provider with capability information
type CapableProvider interface {
	Provider
	Capabilities() ProviderCapabilities
}
