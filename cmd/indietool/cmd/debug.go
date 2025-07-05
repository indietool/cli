package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	// log "github.com/sirupsen/logrus"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:    "debug",
	Short:  "Debug information about providers and configuration",
	Hidden: true, // Hide from help output - this is for development only
	Run: func(cmd *cobra.Command, args []string) {
		// Get global instances
		config := GetConfig()
		registry := GetProviderRegistry()

		fmt.Println("=== Debug Information ===")
		fmt.Println()

		// Configuration source information
		fmt.Println("Configuration Source:")
		fmt.Printf("  Config path: %s\n", configPath)
		fmt.Printf("  Default path: %s\n", defaultConfigPath)
		fmt.Printf("  Using default: %v\n", configPath == defaultConfigPath)

		// Check if default config file exists
		if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
			fmt.Printf("  Default config exists: false\n")
		} else {
			fmt.Printf("  Default config exists: true\n")
		}
		fmt.Println()

		// Config information
		fmt.Println("Configuration:")
		if config == nil {
			fmt.Println("  Config: nil")
		} else if !config.Valid() {
			fmt.Println("  Config: loaded but invalid (no path set)")
		} else {
			fmt.Printf("  Config: loaded from %s\n", config.Path)
			fmt.Printf("  Enabled providers: %v\n", config.GetEnabledProviders())

			// Show domain providers list
			if len(config.Domains.Providers) > 0 {
				fmt.Printf("  Domain providers: %v\n", config.Domains.Providers)
			} else {
				fmt.Printf("  Domain providers: [] (empty)\n")
			}
		}
		fmt.Println()

		// Provider registry information
		fmt.Println("Provider Registry:")
		if registry == nil {
			fmt.Println("  Registry: nil")
		} else {
			allProviders := registry.List()
			enabledProviders := registry.GetEnabledProviders()

			fmt.Printf("  Registered providers: %v\n", allProviders)
			fmt.Printf("  Enabled providers: %d\n", len(enabledProviders))

			// Test each registered provider
			for _, providerName := range allProviders {
				if provider, exists := registry.Get(providerName); exists {
					fmt.Printf("  %s: ✓ registered, name=%s\n", providerName, provider.Name())
				} else {
					fmt.Printf("  %s: ✗ registration failed\n", providerName)
				}
			}
		}
		fmt.Println()

		// Individual provider configs
		if config != nil && config.Valid() {
			fmt.Println("Provider Configurations:")

			if cf := config.Providers.Cloudflare; cf != nil {
				fmt.Printf("  Cloudflare: enabled=%v, has_token=%v, email=%s\n",
					cf.Enabled, cf.APIToken != "", cf.Email)
			} else {
				fmt.Printf("  Cloudflare: not configured\n")
			}

			if pb := config.Providers.Porkbun; pb != nil {
				fmt.Printf("  Porkbun: enabled=%v, has_key=%v, has_secret=%v\n",
					pb.Enabled, pb.APIKey != "", pb.APISecret != "")
			} else {
				fmt.Printf("  Porkbun: not configured\n")
			}

			if nc := config.Providers.Namecheap; nc != nil {
				fmt.Printf("  Namecheap: enabled=%v, has_key=%v, username=%s\n",
					nc.Enabled, nc.APIKey != "", nc.Username)
			} else {
				fmt.Printf("  Namecheap: not configured\n")
			}

			if gd := config.Providers.GoDaddy; gd != nil {
				fmt.Printf("  GoDaddy: enabled=%v, has_key=%v, env=%s\n",
					gd.Enabled, gd.APIKey != "", gd.Environment)
			} else {
				fmt.Printf("  GoDaddy: not configured\n")
			}
		} else {
			fmt.Println("Provider Configurations: none (config invalid)")
		}
		fmt.Println()

		// Test scenario information
		fmt.Println("Test Scenarios:")
		fmt.Printf("  To test empty config creation: rm %s && ./indietool debug\n", defaultConfigPath)
		fmt.Printf("  To test custom config: ./indietool --config test-config.yaml debug\n")
		fmt.Printf("  To test provider addition: ./indietool config add provider cloudflare --api-token test123\n")
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
}
