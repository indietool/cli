package cmd

import (
	"fmt"
	"indietool/cli/providers"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	thelittlehostAPIKey  string
	thelittlehostBaseURL string
)

// configAddProviderTheLittleHostCmd represents the config add provider thelittlehost command
var configAddProviderTheLittleHostCmd = &cobra.Command{
	Use:   "thelittlehost",
	Short: "Add The Little Host provider configuration",
	Long: `Add The Little Host provider configuration to your indietool config file.

This command adds The Little Host API credentials to your configuration file,
allowing indietool to manage DNS zones and records through The Little Host.

You can create an API key from the API Keys page in your The Little Host
account dashboard. Keys are prefixed with tlh_.`,
	Example: `  indietool config add provider thelittlehost --api-key tlh_YOUR_API_KEY
  indietool config add provider thelittlehost --api-key tlh_YOUR_API_KEY --base-url https://custom.host/api/v1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if thelittlehostAPIKey == "" {
			return fmt.Errorf("--api-key is required")
		}

		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not initialized")
		}

		tlhConfig := &providers.TheLittleHostConfig{
			APIKey:  thelittlehostAPIKey,
			BaseURL: thelittlehostBaseURL,
			Enabled: true,
		}

		cfg.Providers.TheLittleHost = tlhConfig

		log.Info("Successfully added and enabled The Little Host provider configuration")

		return nil
	},
}

func init() {
	configAddProviderCmd.AddCommand(configAddProviderTheLittleHostCmd)

	configAddProviderTheLittleHostCmd.Flags().StringVar(&thelittlehostAPIKey, "api-key", "", "The Little Host API key (required, prefix: tlh_)")
	configAddProviderTheLittleHostCmd.Flags().StringVar(&thelittlehostBaseURL, "base-url", "", "Custom API base URL (optional)")

	configAddProviderTheLittleHostCmd.MarkFlagRequired("api-key")
}
