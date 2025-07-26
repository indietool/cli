package cmd

import (
	"fmt"
	"indietool/cli/indietool/secrets"
	"strings"

	"github.com/spf13/cobra"
)

var secretsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Retrieve a secret value",
	Long:  "Retrieve and display a secret value. By default, the value is masked for security.",
	Args:  cobra.ExactArgs(1),
	RunE:  getSecret,
}

func init() {
	secretsGetCmd.Flags().BoolP("show", "s", false, "Show the actual secret value (WARNING: will be visible in terminal)")
}

func getSecret(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	show, _ := cmd.Flags().GetBool("show")

	// Get secrets config
	secretsConfig := cfg.GetSecretsConfig()
	database := secretsConfig.GetDefaultDatabase()

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	secret, err := manager.GetSecret(name, database)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Check if secret is expired
	if secret.IsExpired() {
		fmt.Printf("⚠️  WARNING: Secret '%s' has expired!\n", name)
	}

	// Display secret information
	fmt.Printf("Name: %s\n", secret.Name)

	if show {
		fmt.Printf("⚠️  WARNING: Secret value will be displayed in plaintext!\n")
		fmt.Printf("Value: %s\n", secret.Value)
	} else {
		fmt.Printf("Value: ***MASKED*** (use --show to show)\n")
	}

	if secret.Note != "" {
		fmt.Printf("Note: %s\n", secret.Note)
	}

	fmt.Printf("Created: %s\n", secret.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", secret.UpdatedAt.Format("2006-01-02 15:04:05"))

	if secret.ExpiresAt != nil {
		fmt.Printf("Expires: %s", secret.ExpiresAt.Format("2006-01-02 15:04:05"))
		if secret.IsExpired() {
			fmt.Printf(" (EXPIRED)")
		}
		fmt.Println()
	}

	return nil
}
