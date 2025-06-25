package registrars

import (
	"context"
	"time"
)

// ManagedDomain represents a domain from a registrar
type ManagedDomain struct {
	Name        string       `json:"name"`
	Registrar   string       `json:"registrar"`
	ExpiryDate  time.Time    `json:"expiry_date"`
	AutoRenewal bool         `json:"auto_renewal"`
	Nameservers []string     `json:"nameservers"`
	Status      DomainStatus `json:"status"`
	LastUpdated time.Time    `json:"last_updated"`
	Cost        *DomainCost  `json:"cost,omitempty"`
	DNSRecords  []DNSRecord  `json:"dns_records,omitempty"`
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

// Registrar defines the interface for domain registrar integrations
type Registrar interface {
	// Identification
	Name() string

	// Authentication & Setup
	Configure(config Config) error
	Validate(ctx context.Context) error

	// Domain Operations
	ListDomains(ctx context.Context) ([]ManagedDomain, error)
	GetDomain(ctx context.Context, name string) (*ManagedDomain, error)

	// Renewal Operations
	UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error
	GetRenewalInfo(ctx context.Context, name string) (*DomainCost, error)

	// DNS Operations (basic)
	GetNameservers(ctx context.Context, name string) ([]string, error)
	UpdateNameservers(ctx context.Context, name string, nameservers []string) error
}

// Config holds registrar-specific configuration
type Config struct {
	APIKey      string            `yaml:"api_key"`
	APISecret   string            `yaml:"api_secret,omitempty"`
	Endpoint    string            `yaml:"endpoint,omitempty"`
	Environment string            `yaml:"environment,omitempty"` // prod, sandbox
	Enabled     bool              `yaml:"enabled"`
	Extra       map[string]string `yaml:"extra,omitempty"` // Registrar-specific config
}

// Registry manages multiple registrar instances
type Registry struct {
	registrars map[string]Registrar
	configs    map[string]Config
}

// NewRegistry creates a new registrar registry
func NewRegistry() *Registry {
	return &Registry{
		registrars: make(map[string]Registrar),
		configs:    make(map[string]Config),
	}
}

// Register adds a registrar to the registry
func (r *Registry) Register(name string, registrar Registrar) {
	r.registrars[name] = registrar
}

// Configure sets the configuration for a registrar
func (r *Registry) Configure(name string, config Config) error {
	if registrar, exists := r.registrars[name]; exists {
		r.configs[name] = config
		return registrar.Configure(config)
	}
	// Store config even if registrar not yet registered
	r.configs[name] = config
	return nil
}

// Get retrieves a registrar by name
func (r *Registry) Get(name string) (Registrar, bool) {
	registrar, exists := r.registrars[name]
	return registrar, exists
}

// List returns all registered registrar names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.registrars))
	for name := range r.registrars {
		names = append(names, name)
	}
	return names
}

// GetEnabledRegistrars returns registrars that are configured and enabled
func (r *Registry) GetEnabledRegistrars() []Registrar {
	var enabled []Registrar
	for name, registrar := range r.registrars {
		if config, exists := r.configs[name]; exists && config.Enabled {
			enabled = append(enabled, registrar)
		}
	}
	return enabled
}

// CalculateDomainStatus determines the status based on expiry date and auto-renewal
func CalculateDomainStatus(expiryDate time.Time, autoRenewal bool) DomainStatus {
	now := time.Now()
	daysUntilExpiry := int(expiryDate.Sub(now).Hours() / 24)

	if daysUntilExpiry < 0 {
		return StatusExpired
	}

	if daysUntilExpiry < 7 {
		return StatusCritical
	}

	if daysUntilExpiry < 30 || !autoRenewal {
		return StatusWarning
	}

	return StatusHealthy
}

