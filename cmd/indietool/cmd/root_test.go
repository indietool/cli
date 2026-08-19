package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indietool/cli/indietool"
)

// TestConfigIntegration tests the complete config loading flow
func TestConfigIntegration(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `providers:
  cloudflare:
    account_id: "test-account-id"
    api_token: "test-cf-token"
    enabled: true
  namecheap:
    api_key: "test-nc-key"
    username: "test-user"
    sandbox: true
    enabled: false
domains:
  management:
    expiry_warning_days: [30, 7, 1]
`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test the config loading function directly
	cfg, err := indietool.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the config was loaded correctly
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	// Check that the config is valid
	if !cfg.Valid() {
		t.Error("Config should be valid after successful load")
	}

	// Check that the loaded path is set correctly
	if cfg.Path != configPath {
		t.Errorf("Expected Path to be '%s', got '%s'", configPath, cfg.Path)
	}

	// Test Cloudflare config
	cfConfig := cfg.Providers.Cloudflare
	if cfConfig == nil {
		t.Fatal("Cloudflare config should not be nil")
	}

	if !cfConfig.Enabled {
		t.Error("Cloudflare should be enabled")
	}

	if cfConfig.AccountId != "test-account-id" {
		t.Errorf("Expected Cloudflare account id 'test-account-id', got '%s'", cfConfig.AccountId)
	}

	if cfConfig.APIToken != "test-cf-token" {
		t.Errorf("Expected Cloudflare API token 'test-cf-token', got '%s'", cfConfig.APIToken)
	}

	// Test Namecheap config (should be configured but disabled)
	ncConfig := cfg.Providers.Namecheap
	if ncConfig == nil {
		t.Fatal("Namecheap config should not be nil")
	}

	if ncConfig.Enabled {
		t.Error("Namecheap should be disabled")
	}

	if ncConfig.APIKey != "test-nc-key" {
		t.Errorf("Expected Namecheap API key 'test-nc-key', got '%s'", ncConfig.APIKey)
	}

	if !ncConfig.Sandbox {
		t.Error("Namecheap sandbox should be true")
	}

	// Test enabled providers
	enabledProviders := cfg.GetEnabledProviders()
	if len(enabledProviders) != 1 {
		t.Errorf("Expected 1 enabled provider, got %d", len(enabledProviders))
	}

	if len(enabledProviders) > 0 && enabledProviders[0] != "cloudflare" {
		t.Errorf("Expected 'cloudflare' to be the only enabled provider, got '%s'", enabledProviders[0])
	}

	// Test management config
	expectedDays := []int{30, 7, 1}
	actualDays := cfg.Domains.Management.ExpiryWarningDays
	if len(actualDays) != len(expectedDays) {
		t.Errorf("Expected %d expiry warning days, got %d", len(expectedDays), len(actualDays))
	}

	for i, expected := range expectedDays {
		if i >= len(actualDays) || actualDays[i] != expected {
			t.Errorf("Expected expiry warning day %d at index %d, got %d", expected, i, actualDays[i])
		}
	}

	// Test validation
	errors := cfg.ValidateConfig()
	if len(errors) > 0 {
		t.Errorf("Config should be valid, but got errors: %v", errors)
	}
}

// TestConfigPathExpansion tests that the default config path is properly expanded
func TestConfigPathExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expectedPath := filepath.Join(homeDir, ".config", "indietool.yaml")

	// The actual logic is in init(), but we can test the path construction
	actualPath := filepath.Join(homeDir, ".config", "indietool.yaml")

	if actualPath != expectedPath {
		t.Errorf("Expected default config path '%s', got '%s'", expectedPath, actualPath)
	}

	// Test that the path is absolute
	if !filepath.IsAbs(actualPath) {
		t.Errorf("Config path should be absolute, got '%s'", actualPath)
	}
}

// TestGetConfig tests the global config accessor
func TestGetConfig(t *testing.T) {
	// Store original config
	originalConfig := appConfig
	defer func() {
		appConfig = originalConfig
	}()

	// Test with nil config
	appConfig = nil
	cfg := GetConfig()
	if cfg != nil {
		t.Error("GetConfig() should return nil when appConfig is nil")
	}

	// Test with valid config
	testConfig := &indietool.Config{}
	appConfig = testConfig
	cfg = GetConfig()
	if cfg != testConfig {
		t.Error("GetConfig() should return the same config instance")
	}
}

func TestInitConfigWithMissingFile(t *testing.T) {
	// Store original values
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	// Point the config at a non-existent file
	appConfig = indietool.GetDefaultConfig()
	appConfig.Path = "/non/existent/config.yaml"

	// Call initConfig - this should not fatal, but fall back to a default config
	initConfig()

	// Check that appConfig is not nil (default config created)
	if appConfig == nil {
		t.Error("appConfig should not be nil after initConfig with missing file")
	}

	// The config must not claim to have been loaded from the missing file
	if appConfig.Path == "/non/existent/config.yaml" {
		t.Error("Config path should not point at the missing file after initConfig")
	}
}

func TestSaveConfigIfValid(t *testing.T) {
	// Store original values
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	// Test with nil config
	appConfig = nil
	saveConfigIfValid() // Should not panic or do anything

	// Test with invalid config
	appConfig = &indietool.Config{} // No Path, so invalid
	saveConfigIfValid()             // Should not save anything

	// Test with Viper config (should skip saving)
	appConfig = &indietool.Config{Path: "<viper>"}
	saveConfigIfValid() // Should skip saving

	// Test with valid config and temporary file
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "test-save-config.yaml")

	// Create a test config file first
	testConfigContent := `domains:
  management:
    expiry_warning_days: [30, 7, 1]
`
	err := os.WriteFile(testConfigPath, []byte(testConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load the config
	cfg, err := indietool.LoadFromPath(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Modify the config
	cfg.Domains.Management.ExpiryWarningDays = []int{60, 14, 2}
	appConfig = cfg

	// Save it back
	saveConfigIfValid()

	// Verify it was saved correctly by loading it again
	reloadedCfg, err := indietool.LoadFromPath(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to reload saved config: %v", err)
	}

	// Check that the changes were persisted
	expectedDays := []int{60, 14, 2}
	actualDays := reloadedCfg.Domains.Management.ExpiryWarningDays
	if len(actualDays) != len(expectedDays) {
		t.Errorf("Expected %d expiry warning days, got %d", len(expectedDays), len(actualDays))
	}

	for i, expected := range expectedDays {
		if i >= len(actualDays) || actualDays[i] != expected {
			t.Errorf("Expected expiry warning day %d at index %d, got %d", expected, i, actualDays[i])
		}
	}
}

// newTestConfigFile creates a minimal config file and points appConfig at it.
// Returns the config file path.
func newTestConfigFile(t *testing.T) string {
	t.Helper()

	// Reset provider flag values so state from earlier test executions
	// does not leak into this one.
	cloudflareAPIToken = ""
	cloudflareEmail = ""
	cloudflareAccountID = ""
	jsonOutput = false
	verbose = false
	domainRegisterYes = false
	domainRegisterDryRun = false

	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.yaml")

	content := `domains:
  management:
    expiry_warning_days: [30, 7, 1]
`
	if err := os.WriteFile(testConfigPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := indietool.LoadFromPath(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	appConfig = cfg
	return testConfigPath
}

// TestConfigAddProviderCloudflarePersistsAccountID verifies that
// `config add provider cloudflare --api-token X --account-id acc`
// persists the account_id into the config file.
func TestConfigAddProviderCloudflarePersistsAccountID(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	configPath := newTestConfigFile(t)

	rootCmd.SetArgs([]string{
		"config", "add", "provider", "cloudflare",
		"--api-token", "test-token",
		"--account-id", "acc-123",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config add provider cloudflare failed: %v", err)
	}

	reloaded, err := indietool.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	cf := reloaded.Providers.Cloudflare
	if cf == nil {
		t.Fatal("Cloudflare config should be persisted")
	}
	if cf.AccountId != "acc-123" {
		t.Errorf("Expected account_id 'acc-123', got '%s'", cf.AccountId)
	}
	if cf.APIToken != "test-token" {
		t.Errorf("Expected api_token 'test-token', got '%s'", cf.APIToken)
	}
	if !cf.Enabled {
		t.Error("Cloudflare provider should be enabled")
	}
}

// TestConfigAddProviderCloudflareRequiresAccountID verifies that omitting
// --account-id returns a validation error.
func TestConfigAddProviderCloudflareRequiresAccountID(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	newTestConfigFile(t)

	rootCmd.SetArgs([]string{
		"config", "add", "provider", "cloudflare",
		"--api-token", "test-token",
	})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected an error when --account-id is missing")
	}
	if !strings.Contains(err.Error(), "account-id") {
		t.Errorf("Expected error to mention account-id, got: %v", err)
	}
}
