package providers

import (
	"context"
	"fmt"
	"indietools/cli/domains"
	"time"

	"github.com/charmbracelet/log"
	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

// NamecheapConfig holds Namecheap-specific configuration
type NamecheapConfig struct {
	APIKey   string `yaml:"api_key"`
	Username string `yaml:"username"`
	Sandbox  bool   `yaml:"sandbox"`
	Enabled  bool   `yaml:"enabled"`
	ClientIP string `yaml:"client_ip"`
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
	client *namecheap.Client
	config NamecheapConfig
}

// NewNamecheapProvider creates a new Namecheap provider instance
func NewNamecheapProvider() *NamecheapProvider {
	return &NamecheapProvider{}
}

// NewNamecheap creates a new Namecheap provider instance with configuration
func NewNamecheap(config NamecheapConfig) *NamecheapProvider {
	nc := &NamecheapProvider{
		config: config,
	}

	// Initialize Namecheap client if we have credentials
	if nc.config.APIKey != "" && nc.config.Username != "" {
		log.Debug("Provisioning Namecheap provider with API credentials")
		nc.initializeClient()
	}

	return nc
}

// initializeClient initializes the Namecheap API client
func (n *NamecheapProvider) initializeClient() {
	baseURL := "https://api.namecheap.com/xml.response"
	if n.config.Sandbox {
		baseURL = "https://api.sandbox.namecheap.com/xml.response"
	}

	n.client = namecheap.NewClient(&namecheap.ClientOptions{
		UserName:   n.config.Username,
		ApiUser:    n.config.Username,
		ApiKey:     n.config.APIKey,
		ClientIp:   n.config.ClientIP,
		UseSandbox: n.config.Sandbox,
	})

	// Set the base URL explicitly if needed
	if n.client != nil {
		n.client.BaseURL = baseURL
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
	if n.client == nil {
		return fmt.Errorf("namecheap client not configured")
	}

	// Test API connection by attempting to list domains with minimal parameters
	_, err := n.client.Domains.GetList(&namecheap.DomainsGetListArgs{
		ListType: namecheap.String("ALL"),
		Page:     namecheap.Int(1),
		PageSize: namecheap.Int(10),
	})

	return err
}

// AsRegistrar returns the registrar interface for domain operations
func (n *NamecheapProvider) AsRegistrar() domains.Registrar {
	return n
}

// Configure sets up the Namecheap API client with credentials (for backward compatibility)
func (n *NamecheapProvider) Configure(config NamecheapConfig) error {
	n.config = config

	if n.config.APIKey != "" && n.config.Username != "" {
		n.initializeClient()
	}

	return nil
}

// ListDomains retrieves all domains from Namecheap
func (n *NamecheapProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	if n.client == nil {
		return nil, fmt.Errorf("namecheap client not configured")
	}

	var allDomains []domains.ManagedDomain
	page := 1
	pageSize := 100 // Maximum allowed by Namecheap API

	for {
		// Get list of domains from Namecheap API
		response, err := n.client.Domains.GetList(&namecheap.DomainsGetListArgs{
			ListType: namecheap.String("ALL"),
			Page:     namecheap.Int(page),
			PageSize: namecheap.Int(pageSize),
		})
		if err != nil {
			return nil, fmt.Errorf("provider/namecheap: failed to list domains (page %d): %w", page, err)
		}

		if response == nil || response.Domains == nil || len(*response.Domains) == 0 {
			break // No more domains
		}

		// Convert Namecheap domains to our internal domain structure
		for _, ncDomain := range *response.Domains {
			managedDomain, err := n.convertNamecheapDomain(ctx, ncDomain)
			if err != nil {
				domainName := "unknown"
				if ncDomain.Name != nil {
					domainName = *ncDomain.Name
				}
				log.Errorf("Failed to convert Namecheap domain %s: %v",
					domainName, err)
				continue // Skip this domain but continue with others
			}
			allDomains = append(allDomains, managedDomain)
		}

		// Check if we have more pages
		if response.Paging == nil || len(*response.Domains) < pageSize {
			break // No more pages
		}
		page++
	}

	return allDomains, nil
}

// convertNamecheapDomain converts a Namecheap Domain to our internal ManagedDomain
func (n *NamecheapProvider) convertNamecheapDomain(ctx context.Context, ncDomain namecheap.Domain) (domains.ManagedDomain, error) {
	domainName := ""
	if ncDomain.Name != nil {
		domainName = *ncDomain.Name
	}

	// Get nameservers for this domain
	nameservers, err := n.GetNameservers(ctx, domainName)
	if err != nil {
		log.Warnf("Failed to get nameservers for domain %s: %v", domainName, err)
		nameservers = []string{} // Continue with empty nameservers rather than failing
	}

	// Convert Namecheap DateTime to Go time.Time
	var expiryDate time.Time
	if ncDomain.Expires != nil {
		expiryDate = time.Time(*&ncDomain.Expires.Time)
	}

	// Convert auto-renewal flag
	autoRenewal := false
	if ncDomain.AutoRenew != nil {
		autoRenewal = *ncDomain.AutoRenew
	}

	managedDomain := domains.ManagedDomain{
		Name:        domainName,
		Provider:    "namecheap",
		ExpiryDate:  expiryDate,
		AutoRenewal: autoRenewal,
		Nameservers: nameservers,
		LastUpdated: time.Now(),
	}

	// Calculate and set the domain status
	managedDomain.SetStatus()

	return managedDomain, nil
}

// GetDomain retrieves a specific domain from Namecheap
func (n *NamecheapProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// Namecheap API doesn't have a single domain endpoint, so we list all and filter
	domainList, err := n.ListDomains(ctx)
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
func (n *NamecheapProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via Namecheap API
	// Note: This requires the domains.renew command which may not be available in all Namecheap accounts
	return fmt.Errorf("auto-renewal update not yet implemented for Namecheap")
}

// GetRenewalInfo retrieves renewal pricing information
func (n *NamecheapProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal pricing via Namecheap API
	// This would require integration with their pricing API
	return nil, fmt.Errorf("renewal pricing information not yet implemented for Namecheap")
}

// GetNameservers retrieves nameservers for a domain
func (n *NamecheapProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	if n.client == nil {
		return nil, fmt.Errorf("namecheap client not configured")
	}

	response, err := n.client.DomainsDNS.GetList(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get nameservers for domain %s: %w", name, err)
	}

	if response == nil || response.DomainDNSGetListResult == nil {
		return nil, fmt.Errorf("invalid response from Namecheap API")
	}

	nameservers := []string{}
	if response.DomainDNSGetListResult.Nameservers != nil {
		for _, ns := range *response.DomainDNSGetListResult.Nameservers {
			if ns != "" {
				nameservers = append(nameservers, ns)
			}
		}
	}

	return nameservers, nil
}

// UpdateNameservers updates nameservers for a domain
func (n *NamecheapProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	if n.client == nil {
		return fmt.Errorf("namecheap client not configured")
	}

	_, err := n.client.DomainsDNS.SetCustom(name, nameservers)
	if err != nil {
		return fmt.Errorf("failed to update nameservers for domain %s: %w", name, err)
	}

	return nil
}
