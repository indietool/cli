package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromPath(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `domains:
  registrars:
    cloudflare:
      api_key: "test-key"
      email: "test@example.com"
      enabled: true
  management:
    expiry_warning_days: [30, 7, 1]
`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading from the specific path
	config, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load config from path: %v", err)
	}

	// Verify config was loaded correctly
	if config == nil {
		t.Fatal("Config is nil")
	}

	// Check that the config is valid and has the loaded path set
	if !config.Valid() {
		t.Error("Config should be valid after successful load")
	}

	if config.Path
		t.Errorf("Expected Path to be '%s', got '%s'", configPath, config.Path
	}

	if !config.IsRegistrarEnabled("cloudflare") {
		t.Error("Cloudflare should be enabled")
	}

	cfConfig := config.GetCloudflareConfig()
	if cfConfig == nil {
		t.Fatal("Cloudflare config should not be nil")
	}

	if cfConfig.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", cfConfig.APIKey)
	}

	if cfConfig.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", cfConfig.Email)
	}
}

func TestLoadConfigFromPathNotFound(t *testing.T) {
	// Test loading from a non-existent path
	_, err := LoadConfigFromPath("/non/existent/path.yaml")
	if err == nil {
		t.Error("Expected error when loading from non-existent path")
	}

	if !os.IsNotExist(err) {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

func TestLoadConfigSearchLocations(t *testing.T) {
	// Create a temporary home directory
	tempHome := t.TempDir()

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set temporary HOME
	os.Setenv("HOME", tempHome)

	// Create config in the first search location
	configContent := `domains:
  registrars:
    cloudflare:
      api_key: "from-home"
      email: "home@example.com"
      enabled: true
`

	configPath := filepath.Join(tempHome, ".indietool.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test that LoadConfig finds and loads the config
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cfConfig := config.GetCloudflareConfig()
	if cfConfig == nil {
		t.Fatal("Cloudflare config should not be nil")
	}

	if cfConfig.APIKey != "from-home" {
		t.Errorf("Expected API key 'from-home', got '%s'", cfConfig.APIKey)
	}

	// Check that the config is valid and has the loaded path set
	if !config.Valid() {
		t.Error("Config should be valid after successful load")
	}

	if config.Path
		t.Errorf("Expected Path to be '%s', got '%s'", configPath, config.Path
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Create a temporary home directory
	tempHome := t.TempDir()

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set temporary HOME
	os.Setenv("HOME", tempHome)

	// Create minimal config without expiry warning days
	configContent := `domains:
  registrars:
    cloudflare:
      api_key: "test-key"
      email: "test@example.com"
      enabled: true
`

	configPath := filepath.Join(tempHome, ".indietool.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading with defaults
	config, err := LoadConfigWithDefaults()
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Check that defaults were applied
	if len(config.Domains.Management.ExpiryWarningDays) == 0 {
		t.Error("Expected default expiry warning days to be applied")
	}

	expectedDays := []int{30, 7, 1}
	for i, expected := range expectedDays {
		if i >= len(config.Domains.Management.ExpiryWarningDays) {
			t.Errorf("Missing expected day %d", expected)
			continue
		}
		if config.Domains.Management.ExpiryWarningDays[i] != expected {
			t.Errorf("Expected expiry warning day %d, got %d", expected, config.Domains.Management.ExpiryWarningDays[i])
		}
	}
}

func TestLoadConfigNoFileFound(t *testing.T) {
	// Create a temporary directory with no config files
	tempHome := t.TempDir()

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set temporary HOME with no config files
	os.Setenv("HOME", tempHome)

	// Test that LoadConfig returns an appropriate error
	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when no config file is found")
	}

	// Check that the error message mentions the search paths
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}

	// The error should mention that no config file was found
	if len(errorMsg) == 0 {
		t.Error("Expected non-empty error message")
	}
}

func TestConfigValid(t *testing.T) {
	// Test nil config
	var nilConfig *Config
	if nilConfig.Valid() {
		t.Error("Nil config should not be valid")
	}

	// Test empty config (no Path)
	emptyConfig := &Config{}
	if emptyConfig.Valid() {
		t.Error("Empty config should not be valid")
	}

	// Test config with empty Path
	emptyPathConfig := &Config{Path
	if emptyPathConfig.Valid() {
		t.Error("Config with empty Path should not be valid")
	}

	// Test valid config
	validConfig := &Config{Path"}
	if !validConfig.Valid() {
		t.Error("Config with Path should be valid")
	}

	// Test config loaded from Viper
	viperConfig := &Config{Path
	if !viperConfig.Valid() {
		t.Error("Config loaded from Viper should be valid")
	}
}
