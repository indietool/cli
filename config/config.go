package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/spf13/viper"
)

// Config represents the entire configuration structure for the indietool CLI
type Config struct {
	Domains DomainsConfig `yaml:"domains"`
	Path    string        `yaml:"-"` // Path where config was successfully loaded from
}

// DomainsConfig holds all domain-related configuration
type DomainsConfig struct {
	Registrars RegistrarsConfig `yaml:"registrars"`
	Management ManagementConfig `yaml:"management"`
}

// RegistrarsConfig holds configuration for all supported registrars
type RegistrarsConfig struct {
	Cloudflare *CloudflareConfig `yaml:"cloudflare,omitempty,omitzero"`
	Namecheap  *NamecheapConfig  `yaml:"namecheap,omitempty,omitzero"`
	Porkbun    *PorkbunConfig    `yaml:"porkbun,omitempty,omitzero"`
	GoDaddy    *GoDaddyConfig    `yaml:"godaddy,omitempty,omitzero"`
}

// CloudflareConfig holds Cloudflare-specific configuration
type CloudflareConfig struct {
	APIToken string `yaml:"api_token"`
	Email    string `yaml:"email"`
	Enabled  bool   `yaml:"enabled"`
}

// NamecheapConfig holds Namecheap-specific configuration
type NamecheapConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Username  string `yaml:"username"`
	Sandbox   bool   `yaml:"sandbox"`
	Enabled   bool   `yaml:"enabled"`
}

// PorkbunConfig holds Porkbun-specific configuration
type PorkbunConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Enabled   bool   `yaml:"enabled"`
}

// GoDaddyConfig holds GoDaddy-specific configuration
type GoDaddyConfig struct {
	APIKey      string `yaml:"api_key"`
	APISecret   string `yaml:"api_secret"`
	Environment string `yaml:"environment"` // "production" or "ote" (test environment)
	Enabled     bool   `yaml:"enabled"`
}

// ManagementConfig holds domain management settings
type ManagementConfig struct {
	ExpiryWarningDays []int `yaml:"expiry_warning_days"`
}

// LoadConfigFromPath loads the configuration from the specified file path
func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set the loaded path on successful parse
	config.Path = path

	return &config, nil
}

// LoadConfig searches for and loads the configuration file from standard locations.
// Searches in order:
//  1. ~/.indietool.yaml
//  2. ~/.config/indietool.yaml
//
// Returns the first config file found, or an error if none are found.
func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Define search paths in order of preference
	searchPaths := []string{
		filepath.Join(homeDir, ".indietool.yaml"),
		filepath.Join(homeDir, ".config", "indietool.yaml"),
	}

	// var lastErr error
	for _, path := range searchPaths {
		config, err := LoadConfigFromPath(path)
		if err == nil {
			return config, nil
		}
		// Only store the error if it's not a "file not found" error
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
		}
		// lastErr = err
	}

	// If we get here, no config file was found in any location
	return nil, fmt.Errorf("no config file found in any of the search paths: %v", searchPaths)
}

// LoadConfigWithDefaults searches for and loads configuration from standard locations,
// applying sensible defaults if certain values are not set.
func LoadConfigWithDefaults() (*Config, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Apply defaults if not set
	if len(config.Domains.Management.ExpiryWarningDays) == 0 {
		config.Domains.Management.ExpiryWarningDays = []int{30, 7, 1}
	}

	return config, nil
}

// Valid returns true if the configuration was successfully loaded from a file
func (c *Config) Valid() bool {
	return c != nil && c.Path != ""
}

// GetCloudflareConfig safely returns the Cloudflare configuration
func (c *Config) GetCloudflareConfig() *CloudflareConfig {
	if c.Domains.Registrars.Cloudflare == nil {
		return nil
	}
	return c.Domains.Registrars.Cloudflare
}

// GetNamecheapConfig safely returns the Namecheap configuration
func (c *Config) GetNamecheapConfig() *NamecheapConfig {
	if c.Domains.Registrars.Namecheap == nil {
		return nil
	}
	return c.Domains.Registrars.Namecheap
}

// GetPorkbunConfig safely returns the Porkbun configuration
func (c *Config) GetPorkbunConfig() *PorkbunConfig {
	if c.Domains.Registrars.Porkbun == nil {
		return nil
	}
	return c.Domains.Registrars.Porkbun
}

// GetGoDaddyConfig safely returns the GoDaddy configuration
func (c *Config) GetGoDaddyConfig() *GoDaddyConfig {
	if c.Domains.Registrars.GoDaddy == nil {
		return nil
	}
	return c.Domains.Registrars.GoDaddy
}

// SetCloudflareConfig sets the Cloudflare configuration
func (c *Config) SetCloudflareConfig(config *CloudflareConfig) {
	c.Domains.Registrars.Cloudflare = config
}

// SetNamecheapConfig sets the Namecheap configuration
func (c *Config) SetNamecheapConfig(config *NamecheapConfig) {
	c.Domains.Registrars.Namecheap = config
}

// SetPorkbunConfig sets the Porkbun configuration
func (c *Config) SetPorkbunConfig(config *PorkbunConfig) {
	c.Domains.Registrars.Porkbun = config
}

// SetGoDaddyConfig sets the GoDaddy configuration
func (c *Config) SetGoDaddyConfig(config *GoDaddyConfig) {
	c.Domains.Registrars.GoDaddy = config
}

// HasRegistrarConfig checks if a registrar configuration exists (regardless of enabled status)
func (c *Config) HasRegistrarConfig(registrar string) bool {
	switch registrar {
	case "cloudflare":
		return c.Domains.Registrars.Cloudflare != nil
	case "namecheap":
		return c.Domains.Registrars.Namecheap != nil
	case "porkbun":
		return c.Domains.Registrars.Porkbun != nil
	case "godaddy":
		return c.Domains.Registrars.GoDaddy != nil
	default:
		return false
	}
}

// LoadConfigFromHome loads the configuration from the default location (~/.indietool.yaml)
func LoadConfigFromHome() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".indietool.yaml")
	return LoadConfigFromPath(configPath)
}

// LoadConfigFromCurrentDir loads the configuration from the current directory (config.yaml)
func LoadConfigFromCurrentDir() (*Config, error) {
	return LoadConfigFromPath("config.yaml")
}

// SaveConfig saves the configuration to the specified file path
func (c *Config) SaveConfig(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetEnabledRegistrars returns a list of enabled registrar names
func (c *Config) GetEnabledRegistrars() []string {
	var enabled []string

	if c.Domains.Registrars.Cloudflare != nil && c.Domains.Registrars.Cloudflare.Enabled {
		enabled = append(enabled, "cloudflare")
	}
	if c.Domains.Registrars.Namecheap != nil && c.Domains.Registrars.Namecheap.Enabled {
		enabled = append(enabled, "namecheap")
	}
	if c.Domains.Registrars.Porkbun != nil && c.Domains.Registrars.Porkbun.Enabled {
		enabled = append(enabled, "porkbun")
	}
	if c.Domains.Registrars.GoDaddy != nil && c.Domains.Registrars.GoDaddy.Enabled {
		enabled = append(enabled, "godaddy")
	}

	return enabled
}

// IsRegistrarEnabled checks if a specific registrar is enabled
func (c *Config) IsRegistrarEnabled(registrar string) bool {
	switch registrar {
	case "cloudflare":
		return c.Domains.Registrars.Cloudflare != nil && c.Domains.Registrars.Cloudflare.Enabled
	case "namecheap":
		return c.Domains.Registrars.Namecheap != nil && c.Domains.Registrars.Namecheap.Enabled
	case "porkbun":
		return c.Domains.Registrars.Porkbun != nil && c.Domains.Registrars.Porkbun.Enabled
	case "godaddy":
		return c.Domains.Registrars.GoDaddy != nil && c.Domains.Registrars.GoDaddy.Enabled
	default:
		return false
	}
}

// GetRegistrarConfig returns the configuration for a specific registrar as a map
func (c *Config) GetRegistrarConfig(registrar string) map[string]interface{} {
	switch registrar {
	case "cloudflare":
		if c.Domains.Registrars.Cloudflare == nil {
			return nil
		}
		return map[string]interface{}{
			"api_token": c.Domains.Registrars.Cloudflare.APIToken,
			"email":     c.Domains.Registrars.Cloudflare.Email,
			"enabled":   c.Domains.Registrars.Cloudflare.Enabled,
		}
	case "namecheap":
		if c.Domains.Registrars.Namecheap == nil {
			return nil
		}
		return map[string]interface{}{
			"api_key":    c.Domains.Registrars.Namecheap.APIKey,
			"api_secret": c.Domains.Registrars.Namecheap.APISecret,
			"username":   c.Domains.Registrars.Namecheap.Username,
			"sandbox":    c.Domains.Registrars.Namecheap.Sandbox,
			"enabled":    c.Domains.Registrars.Namecheap.Enabled,
		}
	case "porkbun":
		if c.Domains.Registrars.Porkbun == nil {
			return nil
		}
		return map[string]interface{}{
			"api_key":    c.Domains.Registrars.Porkbun.APIKey,
			"api_secret": c.Domains.Registrars.Porkbun.APISecret,
			"enabled":    c.Domains.Registrars.Porkbun.Enabled,
		}
	case "godaddy":
		if c.Domains.Registrars.GoDaddy == nil {
			return nil
		}
		return map[string]interface{}{
			"api_key":     c.Domains.Registrars.GoDaddy.APIKey,
			"api_secret":  c.Domains.Registrars.GoDaddy.APISecret,
			"environment": c.Domains.Registrars.GoDaddy.Environment,
			"enabled":     c.Domains.Registrars.GoDaddy.Enabled,
		}
	default:
		return nil
	}
}

// ValidateConfig performs basic validation on the configuration
func (c *Config) ValidateConfig() []string {
	var errors []string

	// Validate Cloudflare config if enabled
	if c.Domains.Registrars.Cloudflare != nil && c.Domains.Registrars.Cloudflare.Enabled {
		if c.Domains.Registrars.Cloudflare.APIToken == "" {
			errors = append(errors, "Cloudflare API token is required when enabled")
		}
	}

	// Validate Namecheap config if enabled
	if c.Domains.Registrars.Namecheap != nil && c.Domains.Registrars.Namecheap.Enabled {
		if c.Domains.Registrars.Namecheap.APIKey == "" {
			errors = append(errors, "Namecheap API key is required when enabled")
		}
		if c.Domains.Registrars.Namecheap.APISecret == "" {
			errors = append(errors, "Namecheap API secret is required when enabled")
		}
		if c.Domains.Registrars.Namecheap.Username == "" {
			errors = append(errors, "Namecheap username is required when enabled")
		}
	}

	// Validate Porkbun config if enabled
	if c.Domains.Registrars.Porkbun != nil && c.Domains.Registrars.Porkbun.Enabled {
		if c.Domains.Registrars.Porkbun.APIKey == "" {
			errors = append(errors, "Porkbun API key is required when enabled")
		}
		if c.Domains.Registrars.Porkbun.APISecret == "" {
			errors = append(errors, "Porkbun API secret is required when enabled")
		}
	}

	// Validate GoDaddy config if enabled
	if c.Domains.Registrars.GoDaddy != nil && c.Domains.Registrars.GoDaddy.Enabled {
		if c.Domains.Registrars.GoDaddy.APIKey == "" {
			errors = append(errors, "GoDaddy API key is required when enabled")
		}
		if c.Domains.Registrars.GoDaddy.APISecret == "" {
			errors = append(errors, "GoDaddy API secret is required when enabled")
		}
		if c.Domains.Registrars.GoDaddy.Environment != "production" && c.Domains.Registrars.GoDaddy.Environment != "ote" {
			errors = append(errors, "GoDaddy environment must be 'production' or 'ote'")
		}
	}

	// Validate expiry warning days
	for _, days := range c.Domains.Management.ExpiryWarningDays {
		if days < 0 {
			errors = append(errors, "Expiry warning days must be positive")
			break
		}
	}

	return errors
}

// LoadFromViper loads configuration from the already initialized Viper instance
// This integrates with the existing Viper setup in cmd/root.go
func LoadFromViper() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from viper: %w", err)
	}

	// Set a special path indicator for Viper-loaded configs
	config.Path = "<viper>"

	return &config, nil
}

// LoadFromViperWithDefaults loads configuration from Viper and applies sensible defaults
func LoadFromViperWithDefaults() (*Config, error) {
	config, err := LoadFromViper()
	if err != nil {
		return nil, err
	}

	// Apply defaults if not set
	if len(config.Domains.Management.ExpiryWarningDays) == 0 {
		config.Domains.Management.ExpiryWarningDays = []int{30, 7, 1}
	}

	return config, nil
}

// Example usage:
//
// In your command files, you can use this config like:
//
//   // Load config from standard locations (preferred method)
//   config, err := config.LoadConfigWithDefaults()
//   if err != nil {
//       return err
//   }
//
//   // Or load from specific path
//   config, err := config.LoadConfigFromPath("/path/to/config.yaml")
//   if err != nil {
//       return err
//   }
//
//   // Or use with existing Viper setup
//   config, err := config.LoadFromViper()
//   if err != nil {
//       return err
//   }
//
//   // Type-safe access to registrar configs
//   if cfConfig := config.GetCloudflareConfig(); cfConfig != nil && cfConfig.Enabled {
//       // Use cfConfig.APIKey, cfConfig.Email
//   }
//
//   // Check if registrar is both configured and enabled
//   if config.IsRegistrarEnabled("cloudflare") {
//       cfConfig := config.GetCloudflareConfig()
//       // cfConfig is guaranteed to be non-nil here
//   }
//
//   // Loop through all enabled registrars
//   enabledRegistrars := config.GetEnabledRegistrars()
//   for _, registrar := range enabledRegistrars {
//       regConfig := config.GetRegistrarConfig(registrar)
//       // Process each enabled registrar
//   }
//
//   // Check if a registrar is configured (regardless of enabled status)
//   if config.HasRegistrarConfig("namecheap") {
//       ncConfig := config.GetNamecheapConfig()
//       // ncConfig might be disabled but is configured
//   }
//
