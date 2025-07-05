package providers

import (
	"context"
	"fmt"

	"indietool/cli/domains"
)

// NamecheapConfig holds Namecheap-specific configuration
type NamecheapConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Username  string `yaml:"username"`
	Sandbox   bool   `yaml:"sandbox"`
	Enabled   bool   `yaml:"enabled"`
}

// IsEnabled implements ProviderConfig interface
func (n *NamecheapConfig) IsEnabled() bool {
	return n.Enabled
}

// SetEnabled implements ProviderConfig interface
func (n *NamecheapConfig) SetEnabled(enabled bool) {
	n.Enabled = enabled
}

// NamecheapProvider implements the Provider interface for Namecheap
type NamecheapProvider struct {
	config NamecheapConfig
}

// NewNamecheapProvider creates a new Namecheap provider instance
func NewNamecheapProvider() *NamecheapProvider {
	return &NamecheapProvider{}
}

// NewNamecheap creates a new Namecheap provider instance with configuration
func NewNamecheap(config NamecheapConfig) *NamecheapProvider {
	return &NamecheapProvider{
		config: config,
	}
}

// Name returns the provider name
func (n *NamecheapProvider) Name() string {
	return "namecheap"
}

// IsEnabled returns whether this provider is enabled
func (n *NamecheapProvider) IsEnabled() bool {
	return n.config.Enabled
}

// SetEnabled sets the enabled state of this provider
func (n *NamecheapProvider) SetEnabled(enabled bool) {
	n.config.Enabled = enabled
}

// Validate validates the provider configuration and connection
func (n *NamecheapProvider) Validate(ctx context.Context) error {
	// TODO: Implement validation by testing API connection
	return fmt.Errorf("validation not implemented")
}

// AsRegistrar returns the registrar interface for domain operations
func (n *NamecheapProvider) AsRegistrar() domains.Registrar {
	return n
}

// Configure sets up the Namecheap API client with credentials (for backward compatibility)
func (n *NamecheapProvider) Configure(config NamecheapConfig) error {
	n.config = config
	return nil
}

// ListDomains retrieves all domains from Namecheap
func (n *NamecheapProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	// TODO: Implement domain listing from Namecheap API
	return nil, fmt.Errorf("not implemented")
}

// GetDomain retrieves a specific domain from Namecheap
func (n *NamecheapProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// TODO: Implement get domain from Namecheap API
	return nil, fmt.Errorf("not implemented")
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (n *NamecheapProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via Namecheap API
	return fmt.Errorf("not implemented")
}

// GetRenewalInfo retrieves renewal pricing information
func (n *NamecheapProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal info retrieval from Namecheap API
	return nil, fmt.Errorf("not implemented")
}

// GetNameservers retrieves nameservers for a domain
func (n *NamecheapProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	// TODO: Implement nameserver retrieval from Namecheap API
	return nil, fmt.Errorf("not implemented")
}

// UpdateNameservers updates nameservers for a domain
func (n *NamecheapProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	// TODO: Implement nameserver update via Namecheap API
	return fmt.Errorf("not implemented")
}
