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
	client      *namecheap.Client
	config      NamecheapConfig
	recordCache map[string][]namecheap.DomainsDNSHostRecordDetailed // Cache for batch operations
	cacheMutex  sync.RWMutex                                        // Protects record cache
}

// NewNamecheapProvider creates a new Namecheap provider instance
func NewNamecheapProvider() *NamecheapProvider {
	return &NamecheapProvider{
		recordCache: make(map[string][]namecheap.DomainsDNSHostRecordDetailed),
	}
}

// NewNamecheap creates a new Namecheap provider instance with configuration
func NewNamecheap(config NamecheapConfig) *NamecheapProvider {
	nc := &NamecheapProvider{
		config:      config,
		recordCache: make(map[string][]namecheap.DomainsDNSHostRecordDetailed),
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
	var mu sync.Mutex
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

		// Convert Namecheap domains to our internal domain structure concurrently
		var wg sync.WaitGroup
		for _, ncDomain := range *response.Domains {
			wg.Add(1)
			go func(domain namecheap.Domain) {
				defer wg.Done()

				managedDomain, err := n.convertNamecheapDomain(ctx, domain)
				if err != nil {
					domainName := "unknown"
					if domain.Name != nil {
						domainName = *domain.Name
					}
					log.Errorf("Failed to convert Namecheap domain %s: %v",
						domainName, err)
					return // Skip this domain but continue with others
				}

				mu.Lock()
				allDomains = append(allDomains, managedDomain)
				mu.Unlock()
			}(ncDomain)
		}
		wg.Wait()

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

// ============================================================================
// DNS Provider Methods
// ============================================================================

// ListRecords retrieves all DNS records for a domain
func (n *NamecheapProvider) ListRecords(ctx context.Context, domain string) ([]dns.Record, error) {
	if n.client == nil {
		return nil, fmt.Errorf("namecheap client not configured")
	}

	// Get DNS host records from Namecheap
	response, err := n.client.DomainsDNS.GetHosts(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	if response == nil || response.DomainDNSGetHostsResult == nil || response.DomainDNSGetHostsResult.Hosts == nil {
		return []dns.Record{}, nil
	}

	// Convert Namecheap records to our DNS record format
	var dnsRecords []dns.Record
	for _, host := range *response.DomainDNSGetHostsResult.Hosts {
		dnsRecord, err := n.convertFromNamecheapRecord(host, domain)
		if err != nil {
			log.Warnf("Failed to convert Namecheap record %v: %v", host.HostId, err)
			continue
		}
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	// Update cache with fresh data
	n.updateRecordCache(domain, *response.DomainDNSGetHostsResult.Hosts)

	log.Debugf("Retrieved %d DNS records for domain %s", len(dnsRecords), domain)
	return dnsRecords, nil
}

// SetRecord creates or updates a DNS record
func (n *NamecheapProvider) SetRecord(ctx context.Context, domain string, record dns.Record) error {
	if n.client == nil {
		return fmt.Errorf("namecheap client not configured")
	}

	// Load current records to cache if not already cached
	if err := n.ensureRecordsLoaded(ctx, domain); err != nil {
		return fmt.Errorf("failed to load existing records: %w", err)
	}

	// Get current records from cache
	hosts := n.getCachedRecords(domain)

	// Find and update existing record or add new one
	recordFound := false
	for i, host := range hosts {
		if n.recordMatches(host, record, domain) {
			// Update existing record
			hosts[i] = n.convertToNamecheapRecord(record, domain)
			recordFound = true
			log.Debugf("Updating existing DNS record: %s %s %s", record.Name, record.Type, record.Content)
			break
		}
	}

	if !recordFound {
		// Add new record
		newHost := n.convertToNamecheapRecord(record, domain)
		hosts = append(hosts, newHost)
		log.Debugf("Adding new DNS record: %s %s %s", record.Name, record.Type, record.Content)
	}

	// Commit batch changes
	return n.commitRecordChanges(ctx, domain, hosts)
}

// DeleteRecord removes a DNS record by ID
func (n *NamecheapProvider) DeleteRecord(ctx context.Context, domain, recordID string) error {
	if n.client == nil {
		return fmt.Errorf("namecheap client not configured")
	}

	// Convert string ID to int
	hostID, err := strconv.Atoi(recordID)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Load current records to cache if not already cached
	if err := n.ensureRecordsLoaded(ctx, domain); err != nil {
		return fmt.Errorf("failed to load existing records: %w", err)
	}

	// Get current records from cache and remove the target record
	hosts := n.getCachedRecords(domain)
	var updatedHosts []namecheap.DomainsDNSHostRecordDetailed

	for _, host := range hosts {
		if host.HostId == nil || *host.HostId != hostID {
			updatedHosts = append(updatedHosts, host)
		}
	}

	if len(updatedHosts) == len(hosts) {
		return fmt.Errorf("DNS record %s not found", recordID)
	}

	log.Debugf("Deleting DNS record %s", recordID)
	return n.commitRecordChanges(ctx, domain, updatedHosts)
}

// GetRecord retrieves a specific DNS record by name and type
func (n *NamecheapProvider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	if n.client == nil {
		return nil, fmt.Errorf("namecheap client not configured")
	}

	// Load current records
	records, err := n.ListRecords(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Find matching record
	for _, record := range records {
		if record.Name == name && record.Type == recordType {
			return &record, nil
		}
	}

	return nil, fmt.Errorf("DNS record not found")
}

// ============================================================================
// Batch Operation Manager
// ============================================================================

// ensureRecordsLoaded loads records from API if not already cached
func (n *NamecheapProvider) ensureRecordsLoaded(ctx context.Context, domain string) error {
	n.cacheMutex.RLock()
	_, exists := n.recordCache[domain]
	n.cacheMutex.RUnlock()

	if !exists {
		// Load records from API
		_, err := n.ListRecords(ctx, domain)
		return err
	}
	return nil
}

// updateRecordCache updates the cache with fresh record data
func (n *NamecheapProvider) updateRecordCache(domain string, hosts []namecheap.DomainsDNSHostRecordDetailed) {
	n.cacheMutex.Lock()
	defer n.cacheMutex.Unlock()
	n.recordCache[domain] = hosts
}

// getCachedRecords retrieves cached records for a domain
func (n *NamecheapProvider) getCachedRecords(domain string) []namecheap.DomainsDNSHostRecordDetailed {
	n.cacheMutex.RLock()
	defer n.cacheMutex.RUnlock()

	hosts, exists := n.recordCache[domain]
	if !exists {
		return []namecheap.DomainsDNSHostRecordDetailed{}
	}

	// Return a copy to prevent external modification
	result := make([]namecheap.DomainsDNSHostRecordDetailed, len(hosts))
	copy(result, hosts)
	return result
}

// commitRecordChanges commits all record changes via batch SetHosts operation
func (n *NamecheapProvider) commitRecordChanges(ctx context.Context, domain string, hosts []namecheap.DomainsDNSHostRecordDetailed) error {
	// Convert detailed records to input records for SetHosts
	var inputRecords []namecheap.DomainsDNSHostRecord
	for _, host := range hosts {
		inputRecord := namecheap.DomainsDNSHostRecord{
			HostName:   host.Name,
			RecordType: host.Type,
			Address:    host.Address,
			TTL:        host.TTL,
		}
		if host.MXPref != nil {
			mxPref := uint8(*host.MXPref)
			inputRecord.MXPref = &mxPref
		}
		inputRecords = append(inputRecords, inputRecord)
	}

	// Prepare SetHosts arguments
	args := &namecheap.DomainsDNSSetHostsArgs{
		Domain:  namecheap.String(domain),
		Records: &inputRecords,
	}

	// Execute batch update
	_, err := n.client.DomainsDNS.SetHosts(args)
	if err != nil {
		return fmt.Errorf("failed to commit DNS record changes: %w", err)
	}

	// Update cache with committed changes
	n.updateRecordCache(domain, hosts)

	return nil
}

// ============================================================================
// Type Conversion Helpers
// ============================================================================

// convertFromNamecheapRecord converts a Namecheap Host record to our DNS record format
func (n *NamecheapProvider) convertFromNamecheapRecord(host namecheap.DomainsDNSHostRecordDetailed, domain string) (dns.Record, error) {
	record := dns.Record{}

	// Convert record ID from int to string
	if host.HostId != nil {
		record.ID = strconv.Itoa(*host.HostId)
	}

	// Convert record type
	if host.Type != nil {
		record.Type = *host.Type
	} else {
		return record, fmt.Errorf("record type is missing")
	}

	// Convert record name (handle root domain)
	if host.Name != nil {
		if *host.Name == "" {
			record.Name = "@" // Root domain
		} else {
			record.Name = *host.Name
		}
	} else {
		record.Name = "@" // Default to root if missing
	}

	// Convert record content
	if host.Address != nil {
		record.Content = *host.Address
	} else {
		return record, fmt.Errorf("record content is missing")
	}

	// Convert TTL with validation
	if host.TTL != nil {
		record.TTL = n.validateTTL(*host.TTL)
	} else {
		record.TTL = 1800 // Default TTL
	}

	// Handle priority for MX records
	if host.MXPref != nil {
		priority := *host.MXPref
		record.Priority = &priority
	}

	return record, nil
}

// convertToNamecheapRecord converts our DNS record to Namecheap Host format
func (n *NamecheapProvider) convertToNamecheapRecord(record dns.Record, domain string) namecheap.DomainsDNSHostRecordDetailed {
	host := namecheap.DomainsDNSHostRecordDetailed{}

	// Convert record type
	// Namecheap API only accepts uppercase record types
	hostType := strings.ToUpper(record.Type)
	host.Type = &hostType

	// Convert record name (handle root domain)
	if record.Name == "@" || record.Name == "" {
		host.Name = namecheap.String("") // Empty string for root domain in Namecheap
	} else {
		host.Name = &record.Name
	}

	// Convert record content
	host.Address = &record.Content

	// Convert TTL with validation
	validTTL := n.validateTTL(record.TTL)
	host.TTL = &validTTL

	// Handle priority for MX records
	if record.Priority != nil {
		host.MXPref = record.Priority
	}

	return host
}

// validateTTL ensures TTL is within Namecheap's acceptable range
func (n *NamecheapProvider) validateTTL(ttl int) int {
	const (
		minTTL     = 60
		maxTTL     = 60000
		defaultTTL = 1800
	)

	if ttl < minTTL {
		log.Warnf("TTL %d below minimum %d, using default %d", ttl, minTTL, defaultTTL)
		return defaultTTL
	}
	if ttl > maxTTL {
		log.Warnf("TTL %d above maximum %d, using default %d", ttl, maxTTL, defaultTTL)
		return defaultTTL
	}
	return ttl
}

// ============================================================================
// DNS Helper Methods
// ============================================================================

// recordMatches checks if a Namecheap Host record matches our DNS record
func (n *NamecheapProvider) recordMatches(host namecheap.DomainsDNSHostRecordDetailed, record dns.Record, domain string) bool {
	// Check record type
	if host.Type == nil || *host.Type != record.Type {
		return false
	}

	// Check record name (handle root domain)
	hostName := ""
	if host.Name != nil {
		hostName = *host.Name
	}

	recordName := record.Name
	if recordName == "@" {
		recordName = "" // Convert to Namecheap format for comparison
	}

	return hostName == recordName
}
