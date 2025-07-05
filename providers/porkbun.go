package providers

import (
	"context"
	"fmt"

	"indietool/cli/domains"
)

// PorkbunConfig holds Porkbun-specific configuration
type PorkbunConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Enabled   bool   `yaml:"enabled"`
}

// IsEnabled implements ProviderConfig interface
func (p *PorkbunConfig) IsEnabled() bool {
	return p.Enabled
}

// SetEnabled implements ProviderConfig interface
func (p *PorkbunConfig) SetEnabled(enabled bool) {
	p.Enabled = enabled
}

// PorkbunProvider implements the Provider interface for Porkbun
type PorkbunProvider struct {
	config PorkbunConfig
}

// NewPorkbunProvider creates a new Porkbun provider instance
func NewPorkbunProvider() *PorkbunProvider {
	return &PorkbunProvider{}
}

// NewPorkbun creates a new Porkbun provider instance with configuration
func NewPorkbun(config PorkbunConfig) *PorkbunProvider {
	return &PorkbunProvider{
		config: config,
	}
}

// Name returns the provider name
func (p *PorkbunProvider) Name() string {
	return "porkbun"
}

// IsEnabled returns whether this provider is enabled
func (p *PorkbunProvider) IsEnabled() bool {
	return p.config.Enabled
}

// SetEnabled sets the enabled state of this provider
func (p *PorkbunProvider) SetEnabled(enabled bool) {
	p.config.Enabled = enabled
}

// Validate validates the provider configuration and connection
func (p *PorkbunProvider) Validate(ctx context.Context) error {
	// TODO: Implement validation by testing API connection
	return fmt.Errorf("validation not implemented")
}

// AsRegistrar returns the registrar interface for domain operations
func (p *PorkbunProvider) AsRegistrar() domains.Registrar {
	return p
}

// Configure sets up the Porkbun API client with credentials (for backward compatibility)
func (p *PorkbunProvider) Configure(config PorkbunConfig) error {
	p.config = config
	return nil
}

// ListDomains retrieves all domains from Porkbun
func (p *PorkbunProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	// TODO: Implement domain listing from Porkbun API
	return nil, fmt.Errorf("not implemented")
}

// GetDomain retrieves a specific domain from Porkbun
func (p *PorkbunProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// TODO: Implement get domain from Porkbun API
	return nil, fmt.Errorf("not implemented")
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (p *PorkbunProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via Porkbun API
	return fmt.Errorf("not implemented")
}

// GetRenewalInfo retrieves renewal pricing information
func (p *PorkbunProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal info retrieval from Porkbun API
	return nil, fmt.Errorf("not implemented")
}

// GetNameservers retrieves nameservers for a domain
func (p *PorkbunProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	// TODO: Implement nameserver retrieval from Porkbun API
	return nil, fmt.Errorf("not implemented")
}

// UpdateNameservers updates nameservers for a domain
func (p *PorkbunProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	// TODO: Implement nameserver update via Porkbun API
	return fmt.Errorf("not implemented")
}
