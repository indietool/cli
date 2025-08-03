package metrics

import (
	"os"
	"strings"
)

const (
	// DefaultUmamiEndpoint is the default Umami tracking endpoint
	DefaultUmamiEndpoint = "https://i.indietool.dev/api/send"
	// DefaultWebsiteID is the Umami website ID extracted from the tracking script
	DefaultWebsiteID = "6001c6b7-042a-40c5-96b3-81a8879bcef5"

	DefaultUserAgent = "indietool-cli"
)

// Config holds configuration for metrics tracking
type Config struct {
	Enabled   bool
	Endpoint  string
	WebsiteID string
	UserAgent string
}

// NewConfig creates a new metrics configuration with defaults
func NewConfig() *Config {
	return &Config{
		Enabled:   !isTrackingDisabled(),
		Endpoint:  DefaultUmamiEndpoint,
		WebsiteID: DefaultWebsiteID,
		UserAgent: "indietool-cli",
	}
}

func (c *Config) SetVersion(version string) {
	if version != "" {
		c.UserAgent = DefaultUserAgent + "/" + version
	}
}

// isTrackingDisabled checks if tracking should be disabled based on environment
func isTrackingDisabled() bool {
	// Disable in CI environments
	if os.Getenv("CI") != "" {
		return true
	}

	// Disable if explicitly set (any value)
	if os.Getenv("INDIETOOL_DISABLE_TELEMETRY") != "" {
		return true
	}

	// Disable in testing environments
	if strings.Contains(os.Args[0], "test") {
		return true
	}

	return false
}
