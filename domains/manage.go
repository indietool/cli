package domains

import (
	"fmt"
	"time"
)

// DomainListResult for command output
type DomainListResult struct {
	Domains     []ManagedDomain `json:"domains"`
	Summary     DomainSummary   `json:"summary"`
	LastSynced  time.Time       `json:"last_synced"`
}

type DomainSummary struct {
	Total    int `json:"total"`
	Healthy  int `json:"healthy"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
	Expired  int `json:"expired"`
}

// ListOptions for filtering domain lists
type ListOptions struct {
	Registrar  string // Filter by registrar name
	ExpiringIn string // Filter by expiry timeframe (e.g., "30d", "1w")
	Status     string // Filter by status (healthy, warning, critical, expired)
}

// SyncResult represents the result of syncing domains from a registrar
type SyncResult struct {
	Registrar    string    `json:"registrar"`
	DomainsCount int       `json:"domains_count"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	SyncedAt     time.Time `json:"synced_at"`
}

// ManagedDomain is re-exported from registrars package for convenience
// This allows the domains package to work with domain management types
// without creating circular dependencies
type ManagedDomain struct {
	Name         string            `json:"name"`
	Registrar    string            `json:"registrar"`
	ExpiryDate   time.Time         `json:"expiry_date"`
	AutoRenewal  bool              `json:"auto_renewal"`
	Nameservers  []string          `json:"nameservers"`
	Status       DomainStatus      `json:"status"`
	LastUpdated  time.Time         `json:"last_updated"`
	Cost         *DomainCost       `json:"cost,omitempty"`
	DNSRecords   []DNSRecord       `json:"dns_records,omitempty"`
}

type DomainStatus string

const (
	StatusHealthy  DomainStatus = "healthy"  // >30 days to expiry, auto-renewal on
	StatusWarning  DomainStatus = "warning"  // 7-30 days to expiry OR auto-renewal off
	StatusCritical DomainStatus = "critical" // <7 days to expiry
	StatusExpired  DomainStatus = "expired"  // Past expiry date
)

type DomainCost struct {
	Currency      string  `json:"currency"`
	RenewalPrice  float64 `json:"renewal_price"`
	TransferPrice float64 `json:"transfer_price"`
}

type DNSRecord struct {
	Type    string `json:"type"`    // A, AAAA, CNAME, MX, etc.
	Name    string `json:"name"`    // Record name
	Content string `json:"content"` // Record value
	TTL     int    `json:"ttl"`
}

// ListManagedDomains retrieves domains from all configured registrars
func ListManagedDomains(options ListOptions) (*DomainListResult, error) {
	// TODO: Implement domain listing logic
	return nil, fmt.Errorf("not implemented")
}

// SyncDomains syncs domains from specified registrars
func SyncDomains(registrarNames []string) (map[string]SyncResult, error) {
	// TODO: Implement domain sync logic
	return nil, fmt.Errorf("not implemented")
}