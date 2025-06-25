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
	porkbunAPIKey    string
	porkbunAPISecret string
)

// configAddRegistrarPorkbunCmd represents the config add registrar porkbun command
var configAddRegistrarPorkbunCmd = &cobra.Command{
	Use:   "porkbun",
	Short: "Add Porkbun registrar configuration",
	Long: `Add Porkbun registrar configuration to your indietool config file.

This command adds Porkbun API credentials to your configuration file,
allowing indietool to manage domains through Porkbun's registrar service.

You can obtain your API key and secret from your Porkbun account dashboard.`,
	Example: `  indietool config add registrar porkbun --api-key YOUR_API_KEY --api-secret YOUR_API_SECRET`,
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
		porkbunConfig := &config.PorkbunConfig{
			APIKey:    porkbunAPIKey,
			APISecret: porkbunAPISecret,
			Enabled:   true,
		}

		// Set the Porkbun config
		cfg.SetPorkbunConfig(porkbunConfig)

		log.Info("Successfully added and enabled Porkbun registrar configuration")

		return nil
	},
}

func init() {
	configAddRegistrarCmd.AddCommand(configAddRegistrarPorkbunCmd)

	// Add flags for Porkbun configuration
	configAddRegistrarPorkbunCmd.Flags().StringVar(&porkbunAPIKey, "api-key", "", "Porkbun API key (required)")
	configAddRegistrarPorkbunCmd.Flags().StringVar(&porkbunAPISecret, "api-secret", "", "Porkbun API secret (required)")

	// Mark required flags
	configAddRegistrarPorkbunCmd.MarkFlagRequired("api-key")
	configAddRegistrarPorkbunCmd.MarkFlagRequired("api-secret")
}