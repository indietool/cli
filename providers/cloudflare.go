package providers

import (
	"context"
	"fmt"
	"indietool/cli/domains"

	"github.com/charmbracelet/log"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/registrar"
	"github.com/tidwall/gjson"
)

// CloudflareConfig holds Cloudflare-specific configuration
type CloudflareConfig struct {
	AccountId string `yaml:"account_id"`
	APIToken  string `yaml:"api_token"`
	APIKey    string `yaml:"api_key"`
	Email     string `yaml:"email"`
	Enabled   bool   `yaml:"enabled"`
}

// IsEnabled implements ProviderConfig interface
func (c *CloudflareConfig) IsEnabled() bool {
	return c.Enabled
}

// SetEnabled implements ProviderConfig interface
func (c *CloudflareConfig) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

// CloudflareProvider implements the Provider interface for Cloudflare
type CloudflareProvider struct {
	client *cloudflare.Client
	config CloudflareConfig
}

// NewCloudflareProvider creates a new Cloudflare provider instance
func NewCloudflareProvider() *CloudflareProvider {
	return &CloudflareProvider{}
}

// NewCloudflare creates a new Cloudflare provider instance with configuration
func NewCloudflare(config CloudflareConfig) *CloudflareProvider {
	cf := &CloudflareProvider{
		config: config,
	}

	if cf.config.APIKey != "" && cf.config.Email != "" {
		log.Debug("Provisioning Cloudflare provider with API key and email")
		cf.client = cloudflare.NewClient(
			option.WithAPIEmail(cf.config.Email),
			option.WithAPIKey(cf.config.APIKey),
		)
	} else if cf.config.APIToken != "" {
		log.Debug("Provisioning Cloudflare provider with API token")
		cf.client = cloudflare.NewClient(
			option.WithAPIToken(cf.config.APIToken),
		)
	}

	return cf
}

// Name returns the provider name
func (c *CloudflareProvider) Name() string {
	return "cloudflare"
}

// IsEnabled returns whether this provider is enabled
func (c *CloudflareProvider) IsEnabled() bool {
	return c.config.Enabled
}

// SetEnabled sets the enabled state of this provider
func (c *CloudflareProvider) SetEnabled(enabled bool) {
	c.config.Enabled = enabled
}

// Validate validates the provider configuration and connection
func (c *CloudflareProvider) Validate(ctx context.Context) error {
	// TODO: Implement validation by testing API connection
	return fmt.Errorf("validation not implemented")
}

// AsRegistrar returns the registrar interface for domain operations
func (c *CloudflareProvider) AsRegistrar() domains.Registrar {
	return c
}

// ListDomains retrieves all domains from Cloudflare
func (c *CloudflareProvider) ListDomains(ctx context.Context) ([]domains.ManagedDomain, error) {
	cfdomains, err := c.client.Registrar.Domains.List(
		ctx,
		registrar.DomainListParams{
			AccountID: cloudflare.F(c.config.AccountId),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("provider/cloudflare: failed to list domains: %w", err)
	}

	// log.Infof("cf domains: %+v", cfdomains)

	domainList := make([]domains.ManagedDomain, 0, len(cfdomains.Result))
	for _, d := range cfdomains.Result {
		domainList = append(
			domainList,
			parseDomain(d),
		)
	}
	return domainList, nil
}

func parseDomain(rd registrar.Domain) domains.ManagedDomain {
	data := gjson.Parse(rd.JSON.RawJSON())

	autorenew := data.Get("auto_renew").Bool()
	name := data.Get("name").Str
	nameservers := []string{}
	data.Get("name_servers").ForEach(func(key, value gjson.Result) bool {
		nameservers = append(
			nameservers,
			value.String(),
		)
		return true
	})

	dm := domains.ManagedDomain{
		Name:        name,
		ExpiryDate:  rd.ExpiresAt,
		Provider:    "cloudflare",
		AutoRenewal: autorenew,
		Nameservers: nameservers,
	}
	dm.SetStatus()
	return dm
}

// GetDomain retrieves a specific domain from Cloudflare
func (c *CloudflareProvider) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	// TODO: Implement get domain from Cloudflare API
	return nil, fmt.Errorf("not implemented")
}

// UpdateAutoRenewal updates the auto-renewal setting for a domain
func (c *CloudflareProvider) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	// TODO: Implement auto-renewal update via Cloudflare API
	return fmt.Errorf("not implemented")
}

// GetRenewalInfo retrieves renewal pricing information
func (c *CloudflareProvider) GetRenewalInfo(ctx context.Context, name string) (*domains.DomainCost, error) {
	// TODO: Implement renewal info retrieval from Cloudflare API
	return nil, fmt.Errorf("not implemented")
}

// GetNameservers retrieves nameservers for a domain
func (c *CloudflareProvider) GetNameservers(ctx context.Context, name string) ([]string, error) {
	// TODO: Implement nameserver retrieval from Cloudflare API
	return nil, fmt.Errorf("not implemented")
}

// UpdateNameservers updates nameservers for a domain
func (c *CloudflareProvider) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	// TODO: Implement nameserver update via Cloudflare API
	return fmt.Errorf("not implemented")
}
