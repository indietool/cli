package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsSetCmd = &cobra.Command{
	Use:   "set <name[@database]> <value>",
	Short: "Store an encrypted secret",
	Long:  "Store an encrypted secret with an optional note. The secret will be encrypted and stored securely. Use name@database to specify a custom database.",
	Args:  cobra.ExactArgs(2),
	RunE:  setSecret,
}

func init() {
	secretsSetCmd.Flags().String("note", "", "Add a note to describe the secret")
	secretsSetCmd.Flags().String("expires", "", "Set expiration date (RFC3339 format: 2025-12-31T23:59:59Z)")
}

func setSecret(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	identifier := strings.TrimSpace(args[0])
	value := args[1] // Don't trim value as it might contain intentional whitespace

	if identifier == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	// Parse name@database format
	name, database := secrets.ParseSecretIdentifier(identifier)

	// Get flags
	note, _ := cmd.Flags().GetString("note")
	expiresStr, _ := cmd.Flags().GetString("expires")

	var expiresAt *time.Time
	if expiresStr != "" {
		expires, err := time.Parse(time.RFC3339, expiresStr)
		if err != nil {
			return fmt.Errorf("invalid expiration date format (use RFC3339: 2025-12-31T23:59:59Z): %w", err)
		}
		expiresAt = &expires
	}

	// Get secrets config
	secretsConfig := cfg.GetSecretsConfig()

	// Use parsed database or fall back to default
	if database == "" {
		database = secretsConfig.GetDefaultDatabase()
	}

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	if err := manager.SetSecret(name, value, database, note, expiresAt); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' stored successfully", name)
	if note != "" {
		fmt.Printf(" with note: %s", note)
	}
	if expiresAt != nil {
		fmt.Printf(" (expires: %s)", expiresAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	return nil
}
