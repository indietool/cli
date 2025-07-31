package providers

import (
	"context"
	"fmt"
	"indietool/cli/dns"
	"indietool/cli/domains"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
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

	// Convert Porkbun domains to our internal domain structure concurrently
	domainList := make([]domains.ManagedDomain, 0, len(response.Domains))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, porkbunDomain := range response.Domains {
		wg.Add(1)
		go func(pbDomain porkbun.Domain) {
			defer wg.Done()

			managedDomain, err := p.convertPorkbunDomain(ctx, pbDomain)
			if err != nil {
				log.Errorf("Failed to convert Porkbun domain %s: %v", pbDomain.Domain, err)
				return // Skip this domain but continue with others
			}

			mu.Lock()
			domainList = append(domainList, managedDomain)
			mu.Unlock()
		}(porkbunDomain)
	}

	wg.Wait()

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

// ============================================================================
// DNS Provider Methods
// ============================================================================

// ListRecords retrieves all DNS records for a domain
func (p *PorkbunProvider) ListRecords(ctx context.Context, domain string) ([]dns.Record, error) {
	if p.client == nil {
		return nil, fmt.Errorf("porkbun client not configured")
	}

	// Get all DNS records for the domain
	resp, err := p.client.Dns.GetRecords(ctx, domain, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Convert Porkbun records to our DNS record format
	var dnsRecords []dns.Record
	for _, record := range resp.Records {
		dnsRecord, err := p.convertFromPorkbunRecord(record, domain)
		if err != nil {
			log.Warnf("Failed to convert Porkbun record %v: %v", record.ID, err)
			continue
		}
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	log.Debugf("Retrieved %d DNS records for domain %s", len(dnsRecords), domain)
	return dnsRecords, nil
}

// SetRecord creates or updates a DNS record
func (p *PorkbunProvider) SetRecord(ctx context.Context, domain string, record dns.Record) error {
	if p.client == nil {
		return fmt.Errorf("porkbun client not configured")
	}

	// Check if record already exists
	existingRecord, err := p.findExistingRecord(ctx, domain, record.Name, record.Type)
	if err != nil {
		return fmt.Errorf("failed to check for existing record: %w", err)
	}

	if existingRecord != nil {
		// Update existing record
		log.Debugf("Updating existing DNS record: %s %s %s", record.Name, record.Type, record.Content)
		return p.updateRecord(ctx, domain, *existingRecord.ID, record)
	} else {
		// Create new record
		log.Debugf("Creating new DNS record: %s %s %s", record.Name, record.Type, record.Content)
		return p.createRecord(ctx, domain, record)
	}
}

// DeleteRecord removes a DNS record by ID
func (p *PorkbunProvider) DeleteRecord(ctx context.Context, domain, recordID string) error {
	if p.client == nil {
		return fmt.Errorf("porkbun client not configured")
	}

	// Convert string ID to int64
	id, err := strconv.ParseInt(recordID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	_, err = p.client.Dns.DeleteRecord(ctx, domain, id)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record %s: %w", recordID, err)
	}

	log.Debugf("Deleted DNS record %s", recordID)
	return nil
}

// GetRecord retrieves a specific DNS record by name and type
func (p *PorkbunProvider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	if p.client == nil {
		return nil, fmt.Errorf("porkbun client not configured")
	}

	existingRecord, err := p.findExistingRecord(ctx, domain, name, recordType)
	if err != nil {
		return nil, fmt.Errorf("failed to find DNS record: %w", err)
	}

	if existingRecord == nil {
		return nil, fmt.Errorf("DNS record not found")
	}

	dnsRecord, err := p.convertFromPorkbunRecord(*existingRecord, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to convert record: %w", err)
	}

	return &dnsRecord, nil
}

// ============================================================================
// DNS Helper Methods
// ============================================================================

// findExistingRecord searches for an existing DNS record by name and type
func (p *PorkbunProvider) findExistingRecord(ctx context.Context, domain, name, recordType string) (*porkbun.DnsRecord, error) {
	// Normalize the subdomain for Porkbun API
	subdomain := p.normalizeSubdomain(name, domain)

	// Get records by type and subdomain
	resp, err := p.client.Dns.GetRecordsByType(ctx, domain, porkbun.DnsRecordType(recordType), &subdomain)
	if err != nil {
		return nil, err
	}

	// Return the first matching record if any exist
	if len(resp.Records) > 0 {
		return &resp.Records[0], nil
	}

	return nil, nil
}

// createRecord creates a new DNS record in Porkbun
func (p *PorkbunProvider) createRecord(ctx context.Context, domain string, record dns.Record) error {
	porkbunRecord := p.convertToPorkbunRecord(record, domain)

	_, err := p.client.Dns.CreateRecord(ctx, domain, &porkbunRecord)
	return err
}

// updateRecord updates an existing DNS record in Porkbun
func (p *PorkbunProvider) updateRecord(ctx context.Context, domain string, recordID int64, record dns.Record) error {
	porkbunRecord := p.convertToPorkbunRecord(record, domain)

	editRecord := &porkbun.EditRecord{
		Type:    porkbunRecord.Type,
		Name:    porkbunRecord.Name,
		Content: porkbunRecord.Content,
		TTL:     porkbunRecord.TTL,
		Prio:    porkbunRecord.Prio,
	}

	_, err := p.client.Dns.EditRecord(ctx, domain, recordID, editRecord)
	return err
}

// convertFromPorkbunRecord converts a Porkbun DNS record to our format
func (p *PorkbunProvider) convertFromPorkbunRecord(porkbunRecord porkbun.DnsRecord, domain string) (dns.Record, error) {
	record := dns.Record{
		Type:    string(porkbunRecord.Type),
		Content: porkbunRecord.Content,
	}

	// Convert record ID from int64 to string
	if porkbunRecord.ID != nil {
		record.ID = strconv.FormatInt(*porkbunRecord.ID, 10)
	}

	// Convert TTL from string to int
	if porkbunRecord.TTL != "" {
		ttl, err := strconv.Atoi(porkbunRecord.TTL)
		if err != nil {
			log.Warnf("Failed to parse TTL '%s': %v", porkbunRecord.TTL, err)
			ttl = 300 // Default TTL
		}
		record.TTL = ttl
	} else {
		record.TTL = 300 // Default TTL
	}

	// Handle priority for MX records
	if porkbunRecord.Prio != "" {
		priority, err := strconv.Atoi(porkbunRecord.Prio)
		if err != nil {
			log.Warnf("Failed to parse priority '%s': %v", porkbunRecord.Prio, err)
		} else {
			record.Priority = &priority
		}
	}

	// Convert subdomain name to our format
	record.Name = p.denormalizeSubdomain(porkbunRecord.Name, domain)

	return record, nil
}

// convertToPorkbunRecord converts our DNS record to Porkbun format
func (p *PorkbunProvider) convertToPorkbunRecord(record dns.Record, domain string) porkbun.DnsRecord {
	porkbunRecord := porkbun.DnsRecord{
		Type:    porkbun.DnsRecordType(record.Type),
		Name:    p.normalizeSubdomain(record.Name, domain),
		Content: record.Content,
		TTL:     strconv.Itoa(record.TTL),
	}

	// Handle priority for MX records
	if record.Priority != nil {
		porkbunRecord.Prio = strconv.Itoa(*record.Priority)
	}

	return porkbunRecord
}

// normalizeSubdomain converts record names to Porkbun subdomain format
func (p *PorkbunProvider) normalizeSubdomain(name, domain string) string {
	// Handle root domain
	if name == "@" || name == "" || name == domain {
		return ""
	}

	// If name is already just the subdomain, return as-is
	if !strings.Contains(name, ".") {
		return name
	}

	// If name is FQDN, extract subdomain
	if strings.HasSuffix(name, "."+domain) {
		return strings.TrimSuffix(name, "."+domain)
	}

	// Default: return name as-is
	return name
}

// denormalizeSubdomain converts Porkbun subdomain format to our record name format
func (p *PorkbunProvider) denormalizeSubdomain(subdomain, domain string) string {
	// Handle root domain
	if subdomain == "" {
		return "@"
	}

	// Return subdomain as-is for non-root records
	return subdomain
}
