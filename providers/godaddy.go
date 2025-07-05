package providers

import (
	"context"
	"fmt"

	"indietool/cli/domains"
)

// GoDaddyConfig holds GoDaddy-specific configuration
type GoDaddyConfig struct {
	APIKey      string `yaml:"api_key"`
	APISecret   string `yaml:"api_secret"`
	Environment string `yaml:"environment"` // "production" or "ote" (test environment)
	Enabled     bool   `yaml:"enabled"`
}

// IsEnabled implements ProviderConfig interface
func (g *GoDaddyConfig) IsEnabled() bool {
	return g.Enabled
}

// SetEnabled implements ProviderConfig interface
func (g *GoDaddyConfig) SetEnabled(enabled bool) {
	g.Enabled = enabled
}

// GoDaddyProvider implements the Provider interface for GoDaddy
type GoDaddyProvider struct {
	config GoDaddyConfig
}

// NewGoDaddyProvider creates a new GoDaddy provider instance
func NewGoDaddyProvider() *GoDaddyProvider {
	return &GoDaddyProvider{}
}

// NewGoDaddy creates a new GoDaddy provider instance with configuration
func NewGoDaddy(config GoDaddyConfig) *GoDaddyProvider {
	return &GoDaddyProvider{
		config: config,
	}
}

// Name returns the provider name
func (g *GoDaddyProvider) Name() string {
	return "godaddy"
}

// IsEnabled returns whether this provider is enabled
func (g *GoDaddyProvider) IsEnabled() bool {
	return g.config.Enabled
}

// SetEnabled sets the enabled state of this provider
func (g *GoDaddyProvider) SetEnabled(enabled bool) {
	g.config.Enabled = enabled
}

// Validate validates the provider configuration and connection
func (g *GoDaddyProvider) Validate(ctx context.Context) error {
	// TODO: Implement validation by testing API connection
	return fmt.Errorf("validation not implemented")
}

// AsRegistrar returns the registrar interface for domain operations
func (g *GoDaddyProvider) AsRegistrar() domains.Registrar {
	return g
}

// ConfigureFromInterface sets up the GoDaddy API client with credentials from ProviderConfig interface
// func (g *GoDaddyProvider) ConfigureFromInterface(config ProviderConfig) error {
// 	godaddyConfig, ok := config.(*GoDaddyConfig)
// 	if !ok {
// 		return fmt.Errorf("invalid config type for GoDaddy provider")
// 	}
// 	g.config = *godaddyConfig
// 	return nil
// }

// Configure sets up the GoDaddy API client with credentials (for backward compatibility)
func (g *GoDaddyProvider) Configure(config GoDaddyConfig) error {
	g.config = config
	return nil
}

// ListDomains retrieves all domains from GoDaddy
func (g *GoDaddyProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	// TODO: Implement domain listing from GoDaddy API
	return nil, fmt.Errorf("not implemented")
}

// GetDomain retrieves a specific domain from GoDaddy
func (g *GoDaddyProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// TODO: Implement get domain from GoDaddy API
	return nil, fmt.Errorf("not implemented")
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (g *GoDaddyProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via GoDaddy API
	return fmt.Errorf("not implemented")
}

// GetRenewalInfo retrieves renewal pricing information
func (g *GoDaddyProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal info retrieval from GoDaddy API
	return nil, fmt.Errorf("not implemented")
}

// GetNameservers retrieves nameservers for a domain
func (g *GoDaddyProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	// TODO: Implement nameserver retrieval from GoDaddy API
	return nil, fmt.Errorf("not implemented")
}

// UpdateNameservers updates nameservers for a domain
func (g *GoDaddyProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	// TODO: Implement nameserver update via GoDaddy API
	return fmt.Errorf("not implemented")
}
