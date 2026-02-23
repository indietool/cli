package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"indietool/cli/dns"
	"indietool/cli/domains"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// TheLittleHostConfig holds The Little Host specific configuration
type TheLittleHostConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Enabled bool   `yaml:"enabled"`
}

// IsEnabled implements ProviderConfig interface
func (t *TheLittleHostConfig) IsEnabled() bool {
	return t.Enabled
}

// SetEnabled implements ProviderConfig interface
func (t *TheLittleHostConfig) SetEnabled(enabled bool) {
	t.Enabled = enabled
}

// ============================================================================
// API Types
// ============================================================================

// tlhZone represents a DNS zone from The Little Host API
type tlhZone struct {
	ID         int         `json:"id"`
	DomainName string      `json:"domain_name"`
	Records    []tlhRecord `json:"records,omitempty"`
}

// tlhRecord represents a DNS record from The Little Host API
type tlhRecord struct {
	ID         int    `json:"id"`
	RecordType string `json:"type"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	TTL        int    `json:"ttl"`
	Priority   *int   `json:"priority,omitempty"`
}

// tlhVersion represents a version snapshot from The Little Host API
type tlhVersion struct {
	ID        int    `json:"id"`
	ZoneFile  string `json:"zone_file,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Request body wrappers (Rails-style nested params)
type tlhZoneRequest struct {
	Zone tlhZoneParams `json:"zone"`
}

type tlhZoneParams struct {
	DomainName string `json:"domain_name"`
}

type tlhRecordRequest struct {
	Record tlhRecordParams `json:"record"`
}

type tlhRecordParams struct {
	RecordType string `json:"record_type,omitempty"`
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	TTL        int    `json:"ttl,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
}

// ============================================================================
// HTTP Client
// ============================================================================

// TheLittleHostClient is an HTTP client for The Little Host DNS API
type TheLittleHostClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewTheLittleHostClient creates a new The Little Host API client
func NewTheLittleHostClient(apiKey, baseURL string) *TheLittleHostClient {
	return &TheLittleHostClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest makes an authenticated HTTP request to The Little Host API
func (c *TheLittleHostClient) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request %s %s failed with status %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return resp, nil
}

// decodeResponse reads and JSON-decodes the response body into dest
func (c *TheLittleHostClient) decodeResponse(resp *http.Response, dest any) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// --- Zone Operations ---

// ListZones returns all DNS zones owned by the authenticated user
func (c *TheLittleHostClient) ListZones(ctx context.Context) ([]tlhZone, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/zones", nil)
	if err != nil {
		return nil, err
	}

	var zones []tlhZone
	if err := c.decodeResponse(resp, &zones); err != nil {
		return nil, fmt.Errorf("failed to decode zones response: %w", err)
	}
	return zones, nil
}

// CreateZone creates a new DNS zone
func (c *TheLittleHostClient) CreateZone(ctx context.Context, domainName string) (*tlhZone, error) {
	body := tlhZoneRequest{
		Zone: tlhZoneParams{DomainName: domainName},
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/zones", body)
	if err != nil {
		return nil, err
	}

	var zone tlhZone
	if err := c.decodeResponse(resp, &zone); err != nil {
		return nil, fmt.Errorf("failed to decode zone response: %w", err)
	}
	return &zone, nil
}

// ShowZone returns zone details including all DNS records.
// zoneRef can be a numeric ID or a domain name.
func (c *TheLittleHostClient) ShowZone(ctx context.Context, zoneRef string) (*tlhZone, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/zones/"+zoneRef, nil)
	if err != nil {
		return nil, err
	}

	var zone tlhZone
	if err := c.decodeResponse(resp, &zone); err != nil {
		return nil, fmt.Errorf("failed to decode zone response: %w", err)
	}
	return &zone, nil
}

// UpdateZone updates a zone's domain name
func (c *TheLittleHostClient) UpdateZone(ctx context.Context, zoneID int, domainName string) (*tlhZone, error) {
	body := tlhZoneRequest{
		Zone: tlhZoneParams{DomainName: domainName},
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/zones/%d", zoneID), body)
	if err != nil {
		return nil, err
	}

	var zone tlhZone
	if err := c.decodeResponse(resp, &zone); err != nil {
		return nil, fmt.Errorf("failed to decode zone response: %w", err)
	}
	return &zone, nil
}

// DeleteZone permanently deletes a zone and all its records
func (c *TheLittleHostClient) DeleteZone(ctx context.Context, zoneID int) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/zones/%d", zoneID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- Record Operations ---

// ListRecords returns all DNS records for the specified zone
func (c *TheLittleHostClient) ListRecords(ctx context.Context, zoneID int) ([]tlhRecord, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%d/records", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var records []tlhRecord
	if err := c.decodeResponse(resp, &records); err != nil {
		return nil, fmt.Errorf("failed to decode records response: %w", err)
	}
	return records, nil
}

// CreateRecord creates a new DNS record in the specified zone
func (c *TheLittleHostClient) CreateRecord(ctx context.Context, zoneID int, params tlhRecordParams) (*tlhRecord, error) {
	body := tlhRecordRequest{Record: params}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/zones/%d/records", zoneID), body)
	if err != nil {
		return nil, err
	}

	var record tlhRecord
	if err := c.decodeResponse(resp, &record); err != nil {
		return nil, fmt.Errorf("failed to decode record response: %w", err)
	}
	return &record, nil
}

// ShowRecord returns a single DNS record
func (c *TheLittleHostClient) ShowRecord(ctx context.Context, zoneID, recordID int) (*tlhRecord, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID), nil)
	if err != nil {
		return nil, err
	}

	var record tlhRecord
	if err := c.decodeResponse(resp, &record); err != nil {
		return nil, fmt.Errorf("failed to decode record response: %w", err)
	}
	return &record, nil
}

// UpdateRecord updates a DNS record (partial update)
func (c *TheLittleHostClient) UpdateRecord(ctx context.Context, zoneID, recordID int, params tlhRecordParams) (*tlhRecord, error) {
	body := tlhRecordRequest{Record: params}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID), body)
	if err != nil {
		return nil, err
	}

	var record tlhRecord
	if err := c.decodeResponse(resp, &record); err != nil {
		return nil, fmt.Errorf("failed to decode record response: %w", err)
	}
	return &record, nil
}

// DeleteRecord deletes a DNS record
func (c *TheLittleHostClient) DeleteRecord(ctx context.Context, zoneID, recordID int) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- Version Operations ---

// ListVersions returns all version snapshots for a zone, newest first
func (c *TheLittleHostClient) ListVersions(ctx context.Context, zoneID int) ([]tlhVersion, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%d/versions", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var versions []tlhVersion
	if err := c.decodeResponse(resp, &versions); err != nil {
		return nil, fmt.Errorf("failed to decode versions response: %w", err)
	}
	return versions, nil
}

// ShowVersion returns a version snapshot including the full zone file content
func (c *TheLittleHostClient) ShowVersion(ctx context.Context, zoneID, versionID int) (*tlhVersion, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%d/versions/%d", zoneID, versionID), nil)
	if err != nil {
		return nil, err
	}

	var version tlhVersion
	if err := c.decodeResponse(resp, &version); err != nil {
		return nil, fmt.Errorf("failed to decode version response: %w", err)
	}
	return &version, nil
}

// RevertToVersion reverts the zone to the state captured in a version snapshot
func (c *TheLittleHostClient) RevertToVersion(ctx context.Context, zoneID, versionID int) error {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/zones/%d/versions/%d/revert", zoneID, versionID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ============================================================================
// Provider
// ============================================================================

// TheLittleHostProvider implements the dns.Provider interface for The Little Host
type TheLittleHostProvider struct {
	client *TheLittleHostClient
	config TheLittleHostConfig
}

// NewTheLittleHostProvider creates a new The Little Host provider instance
func NewTheLittleHostProvider() *TheLittleHostProvider {
	return &TheLittleHostProvider{}
}

// NewTheLittleHost creates a new The Little Host provider instance with configuration
func NewTheLittleHost(config TheLittleHostConfig) *TheLittleHostProvider {
	tlh := &TheLittleHostProvider{
		config: config,
	}

	if tlh.config.APIKey != "" {
		baseURL := tlh.config.BaseURL
		if baseURL == "" {
			baseURL = "https://dns.thelittlehost.net/api/v1"
		}
		log.Debug("Provisioning The Little Host provider with API credentials")
		tlh.client = NewTheLittleHostClient(tlh.config.APIKey, baseURL)
	}

	return tlh
}

// Name returns the provider name
func (t *TheLittleHostProvider) Name() string {
	return "thelittlehost"
}

// IsEnabled returns whether this provider is enabled
func (t *TheLittleHostProvider) IsEnabled() bool {
	return t.config.Enabled
}

// SetEnabled sets the enabled state of this provider
func (t *TheLittleHostProvider) SetEnabled(enabled bool) {
	t.config.Enabled = enabled
}

// Validate validates the provider configuration and connection
func (t *TheLittleHostProvider) Validate(ctx context.Context) error {
	if t.client == nil {
		return fmt.Errorf("The Little Host client not configured")
	}

	// Test the connection by listing zones
	_, err := t.client.ListZones(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate The Little Host API connection: %w", err)
	}

	return nil
}

// AsRegistrar returns nil since The Little Host is a DNS-only provider
func (t *TheLittleHostProvider) AsRegistrar() domains.Registrar {
	return nil
}

// Capabilities returns the provider's capabilities
func (t *TheLittleHostProvider) Capabilities() dns.ProviderCapabilities {
	return dns.ProviderCapabilities{
		SupportsPriority: true,
		SupportsWildcard: true,
		SupportsTTLRange: true,
		MinTTL:           60,
		MaxTTL:           86400,
	}
}

// Client returns the underlying API client for direct access to
// API functionality not exposed by the dns.Provider interface
// (zone management, version snapshots).
func (t *TheLittleHostProvider) Client() *TheLittleHostClient {
	return t.client
}

// ============================================================================
// dns.Provider Implementation
// ============================================================================

// ListRecords retrieves all DNS records for a domain
func (t *TheLittleHostProvider) ListRecords(ctx context.Context, domain string) ([]dns.Record, error) {
	if t.client == nil {
		return nil, fmt.Errorf("The Little Host client not configured")
	}

	zone, err := t.client.ShowZone(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone for domain %s: %w", domain, err)
	}

	records, err := t.client.ListRecords(ctx, zone.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	var dnsRecords []dns.Record
	for _, rec := range records {
		dnsRecords = append(dnsRecords, t.convertFromTLHRecord(rec, domain))
	}

	log.Debugf("Retrieved %d DNS records for domain %s", len(dnsRecords), domain)
	return dnsRecords, nil
}

// SetRecord creates or updates a DNS record
func (t *TheLittleHostProvider) SetRecord(ctx context.Context, domain string, record dns.Record) error {
	if t.client == nil {
		return fmt.Errorf("The Little Host client not configured")
	}

	zone, err := t.client.ShowZone(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone for domain %s: %w", domain, err)
	}

	// Check if a record with the same name and type already exists
	existing, err := t.findExistingRecord(ctx, zone.ID, record.Name, record.Type, domain)
	if err != nil {
		return fmt.Errorf("failed to check for existing record: %w", err)
	}

	params := t.convertToTLHParams(record)

	if existing != nil {
		log.Debugf("Updating existing DNS record: %s %s %s", record.Name, record.Type, record.Content)
		_, err = t.client.UpdateRecord(ctx, zone.ID, existing.ID, params)
	} else {
		log.Debugf("Creating new DNS record: %s %s %s", record.Name, record.Type, record.Content)
		_, err = t.client.CreateRecord(ctx, zone.ID, params)
	}

	return err
}

// DeleteRecord removes a DNS record by ID
func (t *TheLittleHostProvider) DeleteRecord(ctx context.Context, domain, recordID string) error {
	if t.client == nil {
		return fmt.Errorf("The Little Host client not configured")
	}

	zone, err := t.client.ShowZone(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone for domain %s: %w", domain, err)
	}

	id, err := strconv.Atoi(recordID)
	if err != nil {
		return fmt.Errorf("invalid record ID %q: %w", recordID, err)
	}

	if err := t.client.DeleteRecord(ctx, zone.ID, id); err != nil {
		return fmt.Errorf("failed to delete DNS record %s: %w", recordID, err)
	}

	log.Debugf("Deleted DNS record %s", recordID)
	return nil
}

// GetRecord retrieves a specific DNS record by name and type
func (t *TheLittleHostProvider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	if t.client == nil {
		return nil, fmt.Errorf("The Little Host client not configured")
	}

	zone, err := t.client.ShowZone(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone for domain %s: %w", domain, err)
	}

	existing, err := t.findExistingRecord(ctx, zone.ID, name, recordType, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find DNS record: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("DNS record not found: %s %s", name, recordType)
	}

	rec := t.convertFromTLHRecord(*existing, domain)
	return &rec, nil
}

// ============================================================================
// Helpers
// ============================================================================

// findExistingRecord searches for a record matching name and type within a zone
func (t *TheLittleHostProvider) findExistingRecord(ctx context.Context, zoneID int, name, recordType, domain string) (*tlhRecord, error) {
	records, err := t.client.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	name = strings.ToLower(name)
	recordType = strings.ToUpper(recordType)

	for _, rec := range records {
		normalized := strings.ToLower(dns.NormalizeName(rec.Name, domain))
		if normalized == name && strings.ToUpper(rec.RecordType) == recordType {
			return &rec, nil
		}
	}

	return nil, nil
}

// convertFromTLHRecord converts a TLH API record to the indietool dns.Record format
func (t *TheLittleHostProvider) convertFromTLHRecord(rec tlhRecord, domain string) dns.Record {
	r := dns.Record{
		ID:      strconv.Itoa(rec.ID),
		Type:    rec.RecordType,
		Name:    dns.NormalizeName(rec.Name, domain),
		Content: rec.Value,
		TTL:     rec.TTL,
	}
	if rec.Priority != nil {
		p := *rec.Priority
		r.Priority = &p
	}
	return r
}

// convertToTLHParams converts an indietool dns.Record to TLH API request params
func (t *TheLittleHostProvider) convertToTLHParams(record dns.Record) tlhRecordParams {
	params := tlhRecordParams{
		RecordType: record.Type,
		Name:       record.Name,
		Value:      record.Content,
		TTL:        record.TTL,
	}
	if record.Priority != nil {
		p := *record.Priority
		params.Priority = &p
	}
	return params
}
