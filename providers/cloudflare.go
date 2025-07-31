package providers

import (
	"context"
	"fmt"
	"indietool/cli/dns"
	"indietool/cli/domains"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/cloudflare/cloudflare-go/v4"
	cfDNS "github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/registrar"
	"github.com/cloudflare/cloudflare-go/v4/zones"
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

// ============================================================================
// DNS Provider Methods
// ============================================================================

// ListRecords retrieves all DNS records for a domain
func (c *CloudflareProvider) ListRecords(ctx context.Context, domain string) ([]dns.Record, error) {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID for domain %s: %w", domain, err)
	}

	// List DNS records for the zone
	resp, err := c.client.DNS.Records.List(ctx, cfDNS.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Convert Cloudflare records to our DNS record format
	var dnsRecords []dns.Record
	for _, record := range resp.Result {
		dnsRecord := c.convertFromCloudflareRecord(record, domain)
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	log.Debugf("Retrieved %d DNS records for domain %s", len(dnsRecords), domain)
	return dnsRecords, nil
}

// SetRecord creates or updates a DNS record
func (c *CloudflareProvider) SetRecord(ctx context.Context, domain string, record dns.Record) error {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for domain %s: %w", domain, err)
	}

	// Check if record already exists
	existingRecord, err := c.findExistingRecord(ctx, zoneID, record.Name, record.Type)
	if err != nil {
		return fmt.Errorf("failed to check for existing record: %w", err)
	}

	if existingRecord != nil {
		// Update existing record
		log.Debugf("Updating existing DNS record: %s %s %s", record.Name, record.Type, record.Content)
		return c.updateRecord(ctx, zoneID, existingRecord.ID, record)
	} else {
		// Create new record
		log.Debugf("Creating new DNS record: %s %s %s", record.Name, record.Type, record.Content)
		return c.createRecord(ctx, zoneID, record)
	}
}

// DeleteRecord removes a DNS record by ID
func (c *CloudflareProvider) DeleteRecord(ctx context.Context, domain, recordID string) error {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for domain %s: %w", domain, err)
	}

	_, err = c.client.DNS.Records.Delete(ctx, recordID, cfDNS.RecordDeleteParams{
		ZoneID: cloudflare.F(zoneID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete DNS record %s: %w", recordID, err)
	}

	log.Debugf("Deleted DNS record %s", recordID)
	return nil
}

// GetRecord retrieves a specific DNS record by name and type
func (c *CloudflareProvider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID for domain %s: %w", domain, err)
	}

	existingRecord, err := c.findExistingRecord(ctx, zoneID, name, recordType)
	if err != nil {
		return nil, fmt.Errorf("failed to find DNS record: %w", err)
	}

	if existingRecord == nil {
		return nil, fmt.Errorf("DNS record not found")
	}

	dnsRecord := c.convertFromCloudflareRecord(*existingRecord, domain)
	return &dnsRecord, nil
}

// ============================================================================
// DNS Helper Methods
// ============================================================================

// getZoneID retrieves the Cloudflare zone ID for a domain
func (c *CloudflareProvider) getZoneID(ctx context.Context, domain string) (string, error) {
	// Search for the zone by name
	resp, err := c.client.Zones.List(ctx, zones.ZoneListParams{
		Name: cloudflare.F(domain),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	if len(resp.Result) == 0 {
		return "", fmt.Errorf("zone not found for domain %s", domain)
	}

	zoneID := resp.Result[0].ID
	log.Debugf("Found zone ID %s for domain %s", zoneID, domain)
	return zoneID, nil
}

// findExistingRecord searches for an existing DNS record by name and type
func (c *CloudflareProvider) findExistingRecord(ctx context.Context, zoneID, name, recordType string) (*cfDNS.RecordResponse, error) {
	resp, err := c.client.DNS.Records.List(ctx, cfDNS.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
		Type:   cloudflare.F(cfDNS.RecordListParamsType(recordType)),
	})
	if err != nil {
		return nil, err
	}

	// Filter by name manually since the Name parameter seems to have type issues
	for _, record := range resp.Result {
		if record.Name == name {
			return &record, nil
		}
	}

	return nil, nil
}

// createRecord creates a new DNS record in Cloudflare
func (c *CloudflareProvider) createRecord(ctx context.Context, zoneID string, record dns.Record) error {
	params := c.buildRecordParams(zoneID, record)

	_, err := c.client.DNS.Records.New(ctx, cfDNS.RecordNewParams{
		ZoneID: cloudflare.F(zoneID),
		Body:   params,
	})

	return err
}

// updateRecord updates an existing DNS record in Cloudflare
func (c *CloudflareProvider) updateRecord(ctx context.Context, zoneID, recordID string, record dns.Record) error {
	newParams := c.buildRecordParams(zoneID, record)

	// Cast the NewParams to UpdateParams - they're the same concrete types
	var updateParams cfDNS.RecordUpdateParamsBodyUnion
	updateParams = newParams.(cfDNS.RecordUpdateParamsBodyUnion)

	_, err := c.client.DNS.Records.Update(ctx, recordID, cfDNS.RecordUpdateParams{
		ZoneID: cloudflare.F(zoneID),
		Body:   updateParams,
	})

	return err
}

// buildRecordParams builds Cloudflare API parameters from our DNS record
func (c *CloudflareProvider) buildRecordParams(zoneID string, record dns.Record) cfDNS.RecordNewParamsBodyUnion {
	// Handle different record types
	switch strings.ToUpper(record.Type) {
	case "A":
		return cfDNS.ARecordParam{
			Content: cloudflare.F(record.Content),
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:    cloudflare.F(cfDNS.ARecordTypeA),
		}
	case "AAAA":
		return cfDNS.AAAARecordParam{
			Content: cloudflare.F(record.Content),
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:    cloudflare.F(cfDNS.AAAARecordTypeAAAA),
		}
	case "CNAME":
		return cfDNS.CNAMERecordParam{
			Content: cloudflare.F(record.Content),
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:    cloudflare.F(cfDNS.CNAMERecordTypeCNAME),
		}
	case "MX":
		priority := 10 // Default priority
		if record.Priority != nil {
			priority = *record.Priority
		}
		return cfDNS.MXRecordParam{
			Content:  cloudflare.F(record.Content),
			Name:     cloudflare.F(record.Name),
			Priority: cloudflare.F(float64(priority)),
			TTL:      cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:     cloudflare.F(cfDNS.MXRecordTypeMX),
		}
	case "TXT":
		return cfDNS.TXTRecordParam{
			Content: cloudflare.F(record.Content),
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:    cloudflare.F(cfDNS.TXTRecordTypeTXT),
		}
	default:
		// Fallback to A record for unsupported types
		return cfDNS.ARecordParam{
			Content: cloudflare.F(record.Content),
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(cfDNS.TTL(record.TTL)),
			Type:    cloudflare.F(cfDNS.ARecordTypeA),
		}
	}
}

// convertFromCloudflareRecord converts a Cloudflare DNS record to our format
func (c *CloudflareProvider) convertFromCloudflareRecord(cfRecord cfDNS.RecordResponse, domain string) dns.Record {
	record := dns.Record{
		ID:      cfRecord.ID,
		Type:    string(cfRecord.Type),
		Content: cfRecord.Content,
		TTL:     int(cfRecord.TTL),
	}

	// Handle the record name - convert full domain back to relative name
	if cfRecord.Name == domain {
		record.Name = "@"
	} else if strings.HasSuffix(cfRecord.Name, "."+domain) {
		record.Name = strings.TrimSuffix(cfRecord.Name, "."+domain)
	} else {
		record.Name = cfRecord.Name
	}

	// Handle MX priority - we'll need to parse it from the content for now
	// The Cloudflare SDK v4 has a different structure, so this is simplified for MVP
	if cfRecord.Type == cfDNS.RecordResponseTypeMX {
		// For MX records, priority might be in the record data
		// This is a simplified approach for the MVP
		priority := 10 // Default priority
		record.Priority = &priority
	}

	// Handle Cloudflare proxy status - cfRecord.Proxied is bool, record.Proxied is *bool
	proxied := cfRecord.Proxied
	record.Proxied = &proxied

	return record
}
