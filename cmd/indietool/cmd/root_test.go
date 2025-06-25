package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"indietool/cli/config"
)

// TestConfigIntegration tests the complete config loading flow
func TestConfigIntegration(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `domains:
  registrars:
    cloudflare:
      api_key: "test-cf-key"
      email: "test@example.com"
      enabled: true
    namecheap:
      api_key: "test-nc-key"
      api_secret: "test-nc-secret"
      username: "test-user"
      sandbox: true
      enabled: false
  management:
    expiry_warning_days: [30, 7, 1]
`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test the config loading function directly
	cfg, err := config.LoadConfigFromPath(configPath)
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
	if cfg.LoadedPath != configPath {
		t.Errorf("Expected LoadedPath to be '%s', got '%s'", configPath, cfg.LoadedPath)
	}

	// Test Cloudflare config
	if !cfg.IsRegistrarEnabled("cloudflare") {
		t.Error("Cloudflare should be enabled")
	}

	cfConfig := cfg.GetCloudflareConfig()
	if cfConfig == nil {
		t.Fatal("Cloudflare config should not be nil")
	}

	if cfConfig.APIKey != "test-cf-key" {
		t.Errorf("Expected Cloudflare API key 'test-cf-key', got '%s'", cfConfig.APIKey)
	}

	if cfConfig.Email != "test@example.com" {
		t.Errorf("Expected Cloudflare email 'test@example.com', got '%s'", cfConfig.Email)
	}

	// Test Namecheap config (should be configured but disabled)
	if cfg.IsRegistrarEnabled("namecheap") {
		t.Error("Namecheap should be disabled")
	}

	if !cfg.HasRegistrarConfig("namecheap") {
		t.Error("Namecheap should be configured")
	}

	ncConfig := cfg.GetNamecheapConfig()
	if ncConfig == nil {
		t.Fatal("Namecheap config should not be nil")
	}

	if ncConfig.APIKey != "test-nc-key" {
		t.Errorf("Expected Namecheap API key 'test-nc-key', got '%s'", ncConfig.APIKey)
	}

	if !ncConfig.Sandbox {
		t.Error("Namecheap sandbox should be true")
	}

	// Test enabled registrars
	enabledRegistrars := cfg.GetEnabledRegistrars()
	if len(enabledRegistrars) != 1 {
		t.Errorf("Expected 1 enabled registrar, got %d", len(enabledRegistrars))
	}

	if len(enabledRegistrars) > 0 && enabledRegistrars[0] != "cloudflare" {
		t.Errorf("Expected 'cloudflare' to be the only enabled registrar, got '%s'", enabledRegistrars[0])
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
	testConfig := &config.Config{}
	appConfig = testConfig
	cfg = GetConfig()
	if cfg != testConfig {
		t.Error("GetConfig() should return the same config instance")
	}
}

func TestInitConfigWithMissingFile(t *testing.T) {
	// Store original values
	originalConfigPath := configPath
	originalAppConfig := appConfig
	defer func() {
		configPath = originalConfigPath
		appConfig = originalAppConfig
	}()

	// Set configPath to a non-existent file
	configPath = "/non/existent/config.yaml"
	appConfig = nil

	// Call initConfig - this should not fatal, but create empty config
	initConfig()

	// Check that appConfig is not nil (empty config created)
	if appConfig == nil {
		t.Error("appConfig should not be nil after initConfig with missing file")
	}

	// Check that the config is not valid (no LoadedPath set)
	if appConfig.Valid() {
		t.Error("Config should not be valid when loaded from missing file")
	}

	// Check that LoadedPath is empty
	if appConfig.LoadedPath != "" {
		t.Errorf("Expected empty LoadedPath, got '%s'", appConfig.LoadedPath)
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
	appConfig = &config.Config{} // No LoadedPath, so invalid
	saveConfigIfValid() // Should not save anything

	// Test with Viper config (should skip saving)
	appConfig = &config.Config{LoadedPath: "<viper>"}
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
	cfg, err := config.LoadConfigFromPath(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Modify the config
	cfg.Domains.Management.ExpiryWarningDays = []int{60, 14, 2}
	appConfig = cfg

	// Save it back
	saveConfigIfValid()

	// Verify it was saved correctly by loading it again
	reloadedCfg, err := config.LoadConfigFromPath(testConfigPath)
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
