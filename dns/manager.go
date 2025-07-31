package dns

import (
	"context"
	"fmt"
)

// Manager handles DNS operations across multiple providers
type Manager struct {
	providers []Provider
}

// NewManager creates a new DNS manager with the given DNS providers
func NewManager(providers []Provider) *Manager {
	return &Manager{
		providers: providers,
	}
}

// ListRecords lists DNS records for a domain, auto-detecting or using specified provider
func (m *Manager) ListRecords(ctx context.Context, domain, providerName string) ([]Record, *DetectorResult, error) {
	var provider Provider
	var detectionResult *DetectorResult

	// If no provider specified, attempt auto-detection
	if providerName == "" {
		result, err := DetectProvider(domain)
		detectionResult = result

		if err != nil || result.Provider == "" {
			return nil, result, fmt.Errorf("could not detect DNS provider for %s: %w. Use --provider flag to specify manually", domain, err)
		}

		providerName = result.Provider
	}

	// Get the DNS provider from available providers
	dnsProvider := m.findProvider(providerName)
	if dnsProvider == nil {
		return nil, detectionResult, fmt.Errorf("DNS provider %s not found or not available", providerName)
	}

	provider = dnsProvider

	// List records from the provider
	records, err := provider.ListRecords(ctx, domain)
	if err != nil {
		return nil, detectionResult, fmt.Errorf("failed to list DNS records from %s: %w", providerName, err)
	}

	return records, detectionResult, nil
}

// SetRecord sets a DNS record, auto-detecting or using specified provider
func (m *Manager) SetRecord(ctx context.Context, domain, providerName string, record Record) (*DetectorResult, error) {
	var provider Provider
	var detectionResult *DetectorResult

	// If no provider specified, attempt auto-detection
	if providerName == "" {
		result, err := DetectProvider(domain)
		detectionResult = result

		if err != nil || result.Provider == "" {
			return result, fmt.Errorf("could not detect DNS provider for %s: %w. Use --provider flag to specify manually", domain, err)
		}

		providerName = result.Provider
	}

	// Validate record type
	if err := ValidateRecordType(record.Type); err != nil {
		return detectionResult, err
	}

	// Normalize record name
	record.Name = NormalizeName(record.Name, domain)

	// Get the DNS provider from available providers
	dnsProvider := m.findProvider(providerName)
	if dnsProvider == nil {
		return detectionResult, fmt.Errorf("DNS provider %s not found or not available", providerName)
	}

	provider = dnsProvider

	// Set the record
	if err := provider.SetRecord(ctx, domain, record); err != nil {
		return detectionResult, fmt.Errorf("failed to set DNS record via %s: %w", providerName, err)
	}

	return detectionResult, nil
}

// DeleteRecord deletes a DNS record by ID
func (m *Manager) DeleteRecord(ctx context.Context, domain, providerName, recordID string) error {
	// If no provider specified, attempt auto-detection
	if providerName == "" {
		result, err := DetectProvider(domain)
		if err != nil || result.Provider == "" {
			return fmt.Errorf("could not detect DNS provider for %s: %w. Use --provider flag to specify manually", domain, err)
		}
		providerName = result.Provider
	}

	// Get the DNS provider from available providers
	dnsProvider := m.findProvider(providerName)
	if dnsProvider == nil {
		return fmt.Errorf("DNS provider %s not found or not available", providerName)
	}

	// Delete the record
	if err := dnsProvider.DeleteRecord(ctx, domain, recordID); err != nil {
		return fmt.Errorf("failed to delete DNS record via %s: %w", providerName, err)
	}

	return nil
}

// findProvider finds a DNS provider by name from the available providers
func (m *Manager) findProvider(providerName string) Provider {
	for _, provider := range m.providers {
		if provider.Name() == providerName {
			return provider
		}
	}
	return nil
}

// GetAvailableProviders returns a list of DNS providers available
func (m *Manager) GetAvailableProviders() []string {
	var names []string
	for _, provider := range m.providers {
		names = append(names, provider.Name())
	}
	return names
}
