package cmd

import (
	"fmt"
	"indietool/cli/providers"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

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

The client IP must be the public IP address that will be making API requests.
Use 'auto' to automatically detect your public IP address via https://ipinfo.io/ip (default).
Visiting https://ipinfo.io/ip also shows you your IP.

Note: API access requires a minimum account balance and may not be available
for all account types. You must also whitelist your IP address in the Namecheap
API settings.`,
	Example: `  indietool config add provider namecheap --api-key YOUR_API_KEY --username YOUR_USERNAME
  indietool config add provider namecheap --api-key YOUR_KEY --username YOUR_USERNAME --client-ip auto
  indietool config add provider namecheap --api-key YOUR_KEY --username YOUR_USERNAME --client-ip 1.2.3.4 --sandbox`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if namecheapAPIKey == "" {
			return fmt.Errorf("--api-key is required")
		}
		if namecheapUsername == "" {
			return fmt.Errorf("--username is required")
		}

		// Handle automatic IP detection
		clientIP := namecheapClientIP
		if clientIP == "auto" {
			log.Info("Detecting public IP address...")
			detectedIP, err := detectPublicIP()
			if err != nil {
				return fmt.Errorf("failed to detect public IP: %w", err)
			}
			clientIP = detectedIP
			log.Infof("Detected public IP: %s", clientIP)
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
			ClientIP: clientIP,
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
		if clientIP != "" {
			clientInfo = fmt.Sprintf(" (client IP: %s)", clientIP)
		}

		log.Infof("Successfully added and enabled Namecheap provider configuration (environment: %s)%s", environment, clientInfo)

		return nil
	},
}

// detectPublicIP queries https://ipinfo.io/ip to get the user's public IP address
func detectPublicIP() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://ipinfo.io/ip")
	if err != nil {
		return "", fmt.Errorf("failed to query IP detection service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IP detection service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read IP detection response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty IP address returned from detection service")
	}

	// Basic IP format validation
	if !isValidIP(ip) {
		return "", fmt.Errorf("invalid IP address format: %s", ip)
	}

	return ip, nil
}

// isValidIP performs basic IP address format validation using regexp
func isValidIP(ip string) bool {
	// Basic IPv4 validation pattern
	ipv4Pattern := `^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`
	matched, _ := regexp.MatchString(ipv4Pattern, ip)
	return matched
}

func init() {
	configAddProviderCmd.AddCommand(configAddProviderNamecheapCmd)

	// Add flags for Namecheap configuration
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapAPIKey, "api-key", "", "Namecheap API key (required)")
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapUsername, "username", "", "Namecheap username (required)")
	configAddProviderNamecheapCmd.Flags().StringVar(&namecheapClientIP, "client-ip", "auto", "Client IP address for API requests ('auto' to detect automatically, or specify an IP address)")
	configAddProviderNamecheapCmd.Flags().BoolVar(&namecheapSandbox, "sandbox", false, "Use Namecheap sandbox environment (default: false)")

	// Mark required flags
	configAddProviderNamecheapCmd.MarkFlagRequired("api-key")
	configAddProviderNamecheapCmd.MarkFlagRequired("username")
	// client-ip is not required since it has a default value of "auto"
}
