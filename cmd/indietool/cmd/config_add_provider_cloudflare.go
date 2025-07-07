package cmd

import (
	"fmt"
	"indietool/cli/providers"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	cloudflareAPIToken string
	cloudflareEmail    string
)

// configAddProviderCloudflareCmd represents the config add provider cloudflare command
var configAddProviderCloudflareCmd = &cobra.Command{
	Use:   "cloudflare",
	Short: "Add Cloudflare provider configuration",
	Long: `Add Cloudflare provider configuration to your indietool config file.

This command adds Cloudflare API credentials to your configuration file,
allowing indietool to manage domains and other services through Cloudflare.

You can obtain your API token from your Cloudflare dashboard.`,
	Example: `  indietool config add provider cloudflare --api-token YOUR_API_TOKEN
  indietool config add provider cloudflare --api-token YOUR_API_TOKEN --email you@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if cloudflareAPIToken == "" {
			return fmt.Errorf("--api-token is required")
		}

		// Use the global config instance
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not initialized")
		}

		// Create Cloudflare config (enabled by default)
		cloudflareConfig := &providers.CloudflareConfig{
			APIToken: cloudflareAPIToken,
			Email:    cloudflareEmail,
			Enabled:  true,
		}

		// Set the Cloudflare config
		cfg.Providers.Cloudflare = cloudflareConfig

		log.Info("Successfully added and enabled Cloudflare provider configuration")

		return nil
	},
}

func init() {
	configAddProviderCmd.AddCommand(configAddProviderCloudflareCmd)

	// Add flags for Cloudflare configuration
	configAddProviderCloudflareCmd.Flags().StringVar(&cloudflareAPIToken, "api-token", "", "Cloudflare API token (required)")
	configAddProviderCloudflareCmd.Flags().StringVar(&cloudflareEmail, "email", "", "Cloudflare account email")

	// Mark required flags
	configAddProviderCloudflareCmd.MarkFlagRequired("api-token")
}
