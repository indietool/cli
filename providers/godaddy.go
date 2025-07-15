package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

// GoDaddyClient minimal HTTP client for GoDaddy API
type GoDaddyClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewGoDaddyClient creates a new GoDaddy API client
func NewGoDaddyClient(apiKey, apiSecret, environment string) *GoDaddyClient {
	baseURL := "https://api.godaddy.com"
	if environment == "ote" {
		baseURL = "https://api.ote-godaddy.com"
	}

	return &GoDaddyClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest makes an authenticated HTTP request to the GoDaddy API
func (c *GoDaddyClient) makeRequest(ctx context.Context, method, endpoint string) (*http.Response, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	authHeader := fmt.Sprintf("sso-key %s:%s", c.apiKey, c.apiSecret)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// GoDaddyDomain represents a domain from GoDaddy API
type GoDaddyDomain struct {
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Expires     time.Time `json:"expires"`
	RenewAuto   bool      `json:"renewAuto"`
	NameServers []string  `json:"nameServers"`
	CreatedAt   time.Time `json:"createdAt"`
	DomainId    int64     `json:"domainId"`
	Privacy     bool      `json:"privacy"`
	Locked      bool      `json:"locked"`
}

// ListDomains retrieves all domains from GoDaddy
func (c *GoDaddyClient) ListDomains(ctx context.Context) ([]GoDaddyDomain, error) {
	resp, err := c.makeRequest(ctx, "GET", "/v1/domains")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var domains []GoDaddyDomain
	if err := json.Unmarshal(body, &domains); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return domains, nil
}

// GoDaddyProvider implements the Provider interface for GoDaddy
type GoDaddyProvider struct {
	client *GoDaddyClient
	config GoDaddyConfig
}

// NewGoDaddyProvider creates a new GoDaddy provider instance
func NewGoDaddyProvider() *GoDaddyProvider {
	return &GoDaddyProvider{}
}

// NewGoDaddy creates a new GoDaddy provider instance with configuration
func NewGoDaddy(config GoDaddyConfig) *GoDaddyProvider {
	gd := &GoDaddyProvider{
		config: config,
	}

	if config.APIKey != "" && config.APISecret != "" {
		gd.client = NewGoDaddyClient(config.APIKey, config.APISecret, config.Environment)
	}

	return gd
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
	if g.client == nil {
		return fmt.Errorf("GoDaddy client not configured")
	}

	// Test the connection by making a simple API call
	_, err := g.client.makeRequest(ctx, "GET", "/v1/domains?limit=1")
	if err != nil {
		return fmt.Errorf("failed to validate GoDaddy API connection: %w", err)
	}

	return nil
}

// AsRegistrar returns the registrar interface for domain operations
func (g *GoDaddyProvider) AsRegistrar() domains.Registrar {
	return g
}

// Configure sets up the GoDaddy API client with credentials
func (g *GoDaddyProvider) Configure(config GoDaddyConfig) error {
	g.config = config
	if config.APIKey != "" && config.APISecret != "" {
		g.client = NewGoDaddyClient(config.APIKey, config.APISecret, config.Environment)
	}
	return nil
}

// ListDomains retrieves all domains from GoDaddy
func (g *GoDaddyProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GoDaddy client not configured")
	}

	gdDomains, err := g.client.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("provider/godaddy: failed to list domains: %w", err)
	}

	domainList := make([]domains.ManagedDomain, 0, len(gdDomains))
	for _, d := range gdDomains {
		domainList = append(domainList, parseGoDaddyDomain(d))
	}

	return domainList, nil
}

// parseGoDaddyDomain converts a GoDaddy domain to ManagedDomain
func parseGoDaddyDomain(gd GoDaddyDomain) domains.ManagedDomain {
	dm := domains.ManagedDomain{
		Name:        gd.Domain,
		Provider:    "godaddy",
		ExpiryDate:  gd.Expires,
		AutoRenewal: gd.RenewAuto,
		Nameservers: gd.NameServers,
		LastUpdated: time.Now(),
	}

	// Calculate and set status
	dm.SetStatus()

	return dm
}

// GetDomain retrieves a specific domain from GoDaddy
func (g *GoDaddyProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GoDaddy client not configured")
	}

	// For now, we'll get all domains and find the specific one
	// In a production implementation, you might want to use the specific domain endpoint
	allDomains, err := g.ListDomains(ctx)
	if err != nil {
		return nil, err
	}

	for _, domain := range allDomains {
		if domain.Name == name {
			return &domain, nil
		}
	}

	return nil, fmt.Errorf("domain %s not found", name)
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (g *GoDaddyProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via GoDaddy API
	// This would require the PATCH /v1/domains/{domain} endpoint
	return fmt.Errorf("UpdateAutoRenewal not implemented yet")
}

// GetRenewalInfo retrieves renewal pricing information
func (g *GoDaddyProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal info retrieval from GoDaddy API
	// This would require checking pricing endpoints
	return nil, fmt.Errorf("GetRenewalInfo not implemented yet")
}

// GetNameservers retrieves nameservers for a domain
func (g *GoDaddyProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	domain, err := g.GetDomain(ctx, name)
	if err != nil {
		return nil, err
	}
	return domain.Nameservers, nil
}

// UpdateNameservers updates nameservers for a domain
func (g *GoDaddyProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	// TODO: Implement nameserver update via GoDaddy API
	// This would require the PUT /v1/domains/{domain}/nameServers endpoint
	return fmt.Errorf("UpdateNameservers not implemented yet")
}
