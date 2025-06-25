/*
Copyright © 2025
*/
package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"indietool/cli/config"
)

var (
	cloudflareAPIToken string
	cloudflareEmail    string
)

// configAddRegistrarCloudflareCmd represents the config add registrar cloudflare command
var configAddRegistrarCloudflareCmd = &cobra.Command{
	Use:   "cloudflare",
	Short: "Add Cloudflare registrar configuration",
	Long: `Add Cloudflare registrar configuration to your indietool config file.

This command adds Cloudflare API credentials to your configuration file,
allowing indietool to manage domains through Cloudflare's registrar service.

You can obtain your API token from your Cloudflare dashboard.`,
	Example: `  indietool config add registrar cloudflare --api-token YOUR_API_TOKEN
  indietool config add registrar cloudflare --api-token YOUR_API_TOKEN --email you@example.com`,
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
		cloudflareConfig := &config.CloudflareConfig{
			APIToken: cloudflareAPIToken,
			Email:    cloudflareEmail,
			Enabled:  true,
		}

		// Set the Cloudflare config
		cfg.SetCloudflareConfig(cloudflareConfig)

		log.Info("Successfully added and enabled Cloudflare registrar configuration")

		return nil
	},
}

func init() {
	configAddRegistrarCmd.AddCommand(configAddRegistrarCloudflareCmd)

	// Add flags for Cloudflare configuration
	configAddRegistrarCloudflareCmd.Flags().StringVar(&cloudflareAPIToken, "api-token", "", "Cloudflare API token (required)")
	configAddRegistrarCloudflareCmd.Flags().StringVar(&cloudflareEmail, "email", "", "Cloudflare account email")

	// Mark required flags
	configAddRegistrarCloudflareCmd.MarkFlagRequired("api-token")
}