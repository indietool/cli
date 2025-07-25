package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsInitCmd = &cobra.Command{
	Use:   "init [key-path]",
	Short: "Initialize encryption key for secrets database",
	Long:  "Initialize encryption key for the secrets database. If no key-path is provided, a new key will be generated.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  initSecrets,
}

func initSecrets(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	var keyPath string
	if len(args) > 0 {
		keyPath = strings.TrimSpace(args[0])
	}

	// Get secrets config with defaults
	secretsConfig := cfg.GetSecretsConfig()
	database := secretsConfig.GetDefaultDatabase()

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	if err := manager.InitDatabase(database, keyPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if keyPath != "" {
		fmt.Printf("✓ Encryption key loaded from '%s' for database '%s'\n", keyPath, database)
	} else {
		fmt.Printf("✓ New encryption key generated for database '%s'\n", database)
	}

	return nil
}