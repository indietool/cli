/*
Copyright © 2025
*/
package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	// "indietool/cli/config"
	"indietool/cli/providers"
)

var (
	porkbunAPIKey    string
	porkbunAPISecret string
)

// configAddProviderPorkbunCmd represents the config add provider porkbun command
var configAddProviderPorkbunCmd = &cobra.Command{
	Use:   "porkbun",
	Short: "Add Porkbun provider configuration",
	Long: `Add Porkbun provider configuration to your indietool config file.

This command adds Porkbun API credentials to your configuration file,
allowing indietool to manage domains and other services through Porkbun.

You can obtain your API key and secret from your Porkbun account dashboard.`,
	Example: `  indietool config add provider porkbun --api-key YOUR_API_KEY --api-secret YOUR_API_SECRET`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if porkbunAPIKey == "" {
			return fmt.Errorf("--api-key is required")
		}
		if porkbunAPISecret == "" {
			return fmt.Errorf("--api-secret is required")
		}

		// Use the global config instance
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not initialized")
		}

		// Create Porkbun config (enabled by default)
		porkbunConfig := &providers.PorkbunConfig{
			APIKey:    porkbunAPIKey,
			APISecret: porkbunAPISecret,
			Enabled:   true,
		}

		// Set the Porkbun config
		cfg.Providers.Porkbun = porkbunConfig

		log.Info("Successfully added and enabled Porkbun provider configuration")

		return nil
	},
}

func init() {
	configAddProviderCmd.AddCommand(configAddProviderPorkbunCmd)

	// Add flags for Porkbun configuration
	configAddProviderPorkbunCmd.Flags().StringVar(&porkbunAPIKey, "api-key", "", "Porkbun API key (required)")
	configAddProviderPorkbunCmd.Flags().StringVar(&porkbunAPISecret, "api-secret", "", "Porkbun API secret (required)")

	// Mark required flags
	configAddProviderPorkbunCmd.MarkFlagRequired("api-key")
	configAddProviderPorkbunCmd.MarkFlagRequired("api-secret")
}
