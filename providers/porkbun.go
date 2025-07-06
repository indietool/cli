package providers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"indietool/cli/domains"

	log "github.com/sirupsen/logrus"
	"github.com/tuzzmaniandevil/porkbun-go"
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
	client *porkbun.Client
	config PorkbunConfig
}

// NewPorkbunProvider creates a new Porkbun provider instance
func NewPorkbunProvider() *PorkbunProvider {
	return &PorkbunProvider{}
}

// NewPorkbun creates a new Porkbun provider instance with configuration
func NewPorkbun(config PorkbunConfig) *PorkbunProvider {
	pb := &PorkbunProvider{
		config: config,
	}

	// Initialize Porkbun client if we have credentials
	if pb.config.APIKey != "" && pb.config.APISecret != "" {
		log.Debug("Provisioning Porkbun provider with API credentials")
		pb.client = porkbun.NewClient(&porkbun.Options{
			ApiKey:       pb.config.APIKey,
			SecretApiKey: pb.config.APISecret,
		})
	}

	return pb
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
	if p.client == nil {
		return fmt.Errorf("porkbun client not configured")
	}

	// Test API connection using ping
	_, err := p.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Porkbun API connection: %w", err)
	}

	return nil
}

// AsRegistrar returns the registrar interface for domain operations
func (p *PorkbunProvider) AsRegistrar() domains.Registrar {
	return p
}

// Configure sets up the Porkbun API client with credentials (for backward compatibility)
func (p *PorkbunProvider) Configure(config PorkbunConfig) error {
	p.config = config
	
	if p.config.APIKey != "" && p.config.APISecret != "" {
		p.client = porkbun.NewClient(&porkbun.Options{
			ApiKey:       p.config.APIKey,
			SecretApiKey: p.config.APISecret,
		})
	}
	
	return nil
}

// ListDomains retrieves all domains from Porkbun
func (p *PorkbunProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	if p.client == nil {
		return nil, fmt.Errorf("porkbun client not configured")
	}

	// Get list of domains from Porkbun API
	response, err := p.client.Domains.ListDomains(ctx, &porkbun.DomainListOptions{})
	if err != nil {
		return nil, fmt.Errorf("provider/porkbun: failed to list domains: %w", err)
	}

	// Convert Porkbun domains to our internal domain structure
	domainList := make([]domains.ManagedDomain, 0, len(response.Domains))
	for _, porkbunDomain := range response.Domains {
		managedDomain, err := p.convertPorkbunDomain(ctx, porkbunDomain)
		if err != nil {
			log.Errorf("Failed to convert Porkbun domain %s: %v", porkbunDomain.Domain, err)
			continue // Skip this domain but continue with others
		}
		domainList = append(domainList, managedDomain)
	}

	return domainList, nil
}

// convertPorkbunDomain converts a Porkbun Domain to our internal ManagedDomain
func (p *PorkbunProvider) convertPorkbunDomain(ctx context.Context, porkbunDomain porkbun.Domain) (domains.ManagedDomain, error) {
	// Get nameservers for this domain
	nameservers, err := p.GetNameservers(ctx, porkbunDomain.Domain)
	if err != nil {
		log.Warnf("Failed to get nameservers for domain %s: %v", porkbunDomain.Domain, err)
		nameservers = []string{} // Continue with empty nameservers rather than failing
	}

	managedDomain := domains.ManagedDomain{
		Name:        porkbunDomain.Domain,
		Provider:    "porkbun",
		ExpiryDate:  porkbunDomain.ExpireDate,
		AutoRenewal: bool(porkbunDomain.AutoRenew),
		Nameservers: nameservers,
		LastUpdated: time.Now(),
	}

	// Calculate and set the domain status
	managedDomain.SetStatus()

	return managedDomain, nil
}

// GetDomain retrieves a specific domain from Porkbun
func (p *PorkbunProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// Porkbun API doesn't have a single domain endpoint, so we list all and filter
	domainList, err := p.ListDomains(ctx)
	if err != nil {
		return nil, err
	}

	for _, domain := range domainList {
		if domain.Name == name {
			return &domain, nil
		}
	}

	return nil, fmt.Errorf("domain %s not found", name)
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (p *PorkbunProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via Porkbun API
	// Note: Porkbun API doesn't currently provide an endpoint to update auto-renewal settings
	return fmt.Errorf("auto-renewal update not supported by Porkbun API")
}

// GetRenewalInfo retrieves renewal pricing information
func (p *PorkbunProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// Get pricing information from Porkbun
	if p.client == nil {
		return nil, fmt.Errorf("porkbun client not configured")
	}

	pricingResponse, err := p.client.Pricing.ListPricing(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing information: %w", err)
	}

	// Extract TLD from domain name
	// Simple extraction - in production you might want more robust TLD parsing
	tld := extractTLD(name)
	
	if pricing, exists := pricingResponse.Pricing[tld]; exists {
		// Parse renewal price (Porkbun returns prices as strings)
		// Note: You might need more robust price parsing here
		return &domains.DomainCost{
			Currency:     "USD", // Porkbun prices are typically in USD
			RenewalPrice: parsePrice(pricing.Renewal),
		}, nil
	}

	return nil, fmt.Errorf("pricing information not available for TLD: %s", tld)
}

// GetNameservers retrieves nameservers for a domain
func (p *PorkbunProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	if p.client == nil {
		return nil, fmt.Errorf("porkbun client not configured")
	}

	response, err := p.client.Domains.GetNameServers(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get nameservers for domain %s: %w", name, err)
	}

	return []string(response.NS), nil
}

// UpdateNameservers updates nameservers for a domain
func (p *PorkbunProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	if p.client == nil {
		return fmt.Errorf("porkbun client not configured")
	}

	nsPointer := (*porkbun.NameServers)(&nameservers)
	_, err := p.client.Domains.UpdateNameServers(ctx, name, nsPointer)
	if err != nil {
		return fmt.Errorf("failed to update nameservers for domain %s: %w", name, err)
	}

	return nil
}

// Helper functions

// extractTLD extracts the TLD from a domain name
// Simple implementation - you might want more robust TLD parsing
func extractTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}
	return parts[len(parts)-1]
}

// parsePrice parses a price string to float64
// Simple implementation - you might want more robust price parsing
func parsePrice(priceStr string) float64 {
	// Remove any currency symbols and parse
	cleanPrice := strings.TrimPrefix(priceStr, "$")
	price, err := strconv.ParseFloat(cleanPrice, 64)
	if err != nil {
		log.Warnf("Failed to parse price '%s': %v", priceStr, err)
		return 0.0
	}
	return price
}
