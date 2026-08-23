package cmd

import (
	"fmt"
	"github.com/indietool/cli/providers"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	cloudflareAPIToken  string
	cloudflareEmail     string
	cloudflareAccountID string
	cloudflareSandbox   bool
)

// configAddProviderCloudflareCmd represents the config add provider cloudflare command
var configAddProviderCloudflareCmd = &cobra.Command{
	Use:   "cloudflare",
	Short: "Add Cloudflare provider configuration",
	Long: `Add Cloudflare provider configuration to your indietool config file.

This command adds Cloudflare API credentials to your configuration file,
allowing indietool to manage domains and other services through Cloudflare.

You can obtain your API token from your Cloudflare dashboard. The account ID
is shown on the dashboard Overview page (or in the URL) and is required for
registrar operations such as listing, buying, and renewing domains.`,
	Example: `  indietool config add provider cloudflare --api-token YOUR_API_TOKEN --account-id YOUR_ACCOUNT_ID
  indietool config add provider cloudflare --api-token YOUR_API_TOKEN --account-id YOUR_ACCOUNT_ID --email you@example.com
  indietool config add provider cloudflare --api-token YOUR_API_TOKEN --account-id YOUR_ACCOUNT_ID --sandbox   # Registrar Sandbox test environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if cloudflareAPIToken == "" {
			return fmt.Errorf("--api-token is required")
		}
		if cloudflareAccountID == "" {
			return fmt.Errorf("--account-id is required for Cloudflare registrar operations (find it on the dashboard Overview page)")
		}

		// Use the global config instance
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not initialized")
		}

		// Create Cloudflare config (enabled by default)
		cloudflareConfig := &providers.CloudflareConfig{
			AccountId: cloudflareAccountID,
			APIToken:  cloudflareAPIToken,
			Email:     cloudflareEmail,
			Sandbox:   cloudflareSandbox,
			Enabled:   true,
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
	configAddProviderCloudflareCmd.Flags().StringVar(&cloudflareAccountID, "account-id", "", "Cloudflare account ID (required for registrar operations)")
	configAddProviderCloudflareCmd.Flags().StringVar(&cloudflareEmail, "email", "", "Cloudflare account email")
	configAddProviderCloudflareCmd.Flags().BoolVar(&cloudflareSandbox, "sandbox", false, "Use Cloudflare Registrar Sandbox API (test environment, no billing)")

	// Mark required flags
	configAddProviderCloudflareCmd.MarkFlagRequired("api-token")
}
