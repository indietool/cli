package indietool

import (
	"indietool/cli/providers"
	"os"

	"github.com/goccy/go-yaml"
)

// Config represents the entire configuration structure for the indietool CLI
type Config struct {
	Domains   DomainsConfig   `yaml:"domains"`
	Providers ProvidersConfig `yaml:"providers"`
	Path      string          `yaml:"-"` // Path where config was successfully loaded from
}

// DomainsConfig holds all domain-related configuration
type DomainsConfig struct {
	Providers  []string         `yaml:"providers"` // List of provider names to use for domain management
	Management ManagementConfig `yaml:"management"`
}

// ProvidersConfig holds configuration for all supported providers
type ProvidersConfig struct {
	Cloudflare *providers.CloudflareConfig `yaml:"cloudflare,omitempty,omitzero"`
	Namecheap  *providers.NamecheapConfig  `yaml:"namecheap,omitempty,omitzero"`
	Porkbun    *providers.PorkbunConfig    `yaml:"porkbun,omitempty,omitzero"`
	GoDaddy    *providers.GoDaddyConfig    `yaml:"godaddy,omitempty,omitzero"`
}

// ManagementConfig holds domain management settings
type ManagementConfig struct {
	ExpiryWarningDays []int `yaml:"expiry_warning_days"`
}

// LoadFromPath loads configuration from the specified file path
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Set the loaded path on successful parse
	cfg.Path = path

	return cfg, nil
}

// Valid returns true if the configuration was successfully loaded from a file
func (c *Config) Valid() bool {
	return c != nil && c.Path != ""
}

// SaveConfig saves the configuration to the specified file path
func (c *Config) SaveConfig(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// ValidateConfig validates the configuration and returns any validation errors
func (c *Config) ValidateConfig() []string {
	var errors []string

	// Validate Cloudflare config if present
	if cf := c.Providers.Cloudflare; cf != nil {
		if cf.APIToken == "" && (cf.APIKey == "" || cf.Email == "") {
			errors = append(errors, "Cloudflare: either api_token or both api_key and email must be provided")
		}
	}

	// Validate Porkbun config if present
	if pb := c.Providers.Porkbun; pb != nil {
		if pb.APIKey == "" {
			errors = append(errors, "Porkbun: api_key is required")
		}
		if pb.APISecret == "" {
			errors = append(errors, "Porkbun: api_secret is required")
		}
	}

	// Validate Namecheap config if present
	if nc := c.Providers.Namecheap; nc != nil {
		if nc.APIKey == "" || nc.APISecret == "" || nc.Username == "" {
			errors = append(errors, "Namecheap: api_key, api_secret, and username are all required")
		}
	}

	// Validate GoDaddy config if present
	if gd := c.Providers.GoDaddy; gd != nil {
		if gd.APIKey == "" {
			errors = append(errors, "GoDaddy: api_key is required")
		}
		if gd.APISecret == "" {
			errors = append(errors, "GoDaddy: api_secret is required")
		}
	}

	return errors
}

// GetEnabledProviders returns a list of provider names that are configured and enabled
func (c *Config) GetEnabledProviders() []string {
	var enabled []string
	
	if c.Providers.Cloudflare != nil && c.Providers.Cloudflare.Enabled {
		enabled = append(enabled, "cloudflare")
	}
	if c.Providers.Porkbun != nil && c.Providers.Porkbun.Enabled {
		enabled = append(enabled, "porkbun")
	}
	if c.Providers.Namecheap != nil && c.Providers.Namecheap.Enabled {
		enabled = append(enabled, "namecheap")
	}
	if c.Providers.GoDaddy != nil && c.Providers.GoDaddy.Enabled {
		enabled = append(enabled, "godaddy")
	}
	
	return enabled
}
