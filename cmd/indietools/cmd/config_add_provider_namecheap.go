package cmd

import (
	"fmt"
	"indietools/cli/providers"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	namecheapAPIKey    string
	namecheapAPISecret string
	namecheapUsername  string
	namecheapClientIP  string
	namecheapSandbox   bool
)

// configAddProviderNamecheapCmd represents the config add provider namecheap command
var configAddProviderNamecheapCmd = &cobra.Command{
	Use:   "namecheap",
	Short: "Add Namecheap provider configuration",
	Long: `Add Namecheap provider configuration to your indietool config file.

This command adds Namecheap API credentials to your configuration file,
allowing indietool to manage domains and other services through Namecheap.

You can obtain your API key from your Namecheap account dashboard
under Tools > Business & Dev Tools > API access.

The client IP should be the public IP address that will be making API requests.
You can find your current IP by visiting https://whatismyipaddress.com/
If not specified, Namecheap will try to auto-detect it from the request.

Note: API access requires a minimum account balance and may not be available
for all account types. You must also whitelist your IP address in the Namecheap
API settings.`,
	Example: `  indietool config add provider namecheap --api-key YOUR_API_KEY --username YOUR_USERNAME
  indietool config add provider namecheap --api-key YOUR_KEY --username YOUR_USERNAME --client-ip 203.0.113.1
  indietool config add provider namecheap --api-key YOUR_KEY --username YOUR_USERNAME --sandbox --client-ip 203.0.113.1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if namecheapAPIKey == "" {
			return fmt.Errorf("--api-key is required")
		}
		if namecheapUsername == "" {
			return fmt.Errorf("--username is required")
		}

		// Use the global config instance
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not initialized")
		}

		// Create Namecheap config (enabled by default)
		namecheapConfig := &providers.NamecheapConfig{
			APIKey:   namecheapAPIKey,
			Username: namecheapUsername,
			ClientIP: namecheapClientIP,
			Sandbox:  namecheapSandbox,
			Enabled:  true,
		}

		// Set the Namecheap config
		cfg.Providers.Namecheap = namecheapConfig

		environment := "production"
		if namecheapSandbox {
			environment = "sandbox"
		}

		clientInfo := ""
		if namecheapClientIP != "" {
			clientInfo = fmt.Sprintf(" (client IP: %s)", namecheapClientIP)
		}

		log.Infof("Successfully added and enabled Namecheap provider configuration (environment: %s)%s", environment, clientInfo)

		return nil
	},
}

func init() {
	configAddProviderCmd.AddCommand(configAddProviderNamecheapCmd)

	// Add flags for Namecheap configuration
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapAPIKey, "api-key", "", "Namecheap API key (required)")
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapUsername, "username", "", "Namecheap username (required)")
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapClientIP, "client-ip", "", "Client IP address for API requests (optional - auto-detected if not specified)")
	configAddProviderNamecheapCmd.Flags().BoolVar(&namecheapSandbox, "sandbox", false, "Use Namecheap sandbox environment (default: false)")

	// Mark required flags
	configAddProviderNamecheapCmd.MarkFlagRequired("api-key")
	configAddProviderNamecheapCmd.MarkFlagRequired("username")
	configAddProviderNamecheapCmd.MarkFlagRequired("client-ip")
}
