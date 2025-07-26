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

	// Check if key already exists
	if manager.HasDatabaseKey(database) {
		fmt.Printf("⚠️  WARNING: An encryption key already exists for database '%s'\n", database)
		fmt.Println("   Reinitializing will replace the existing key and make current secrets inaccessible.")
		fmt.Println("   If you have existing secrets, they will become permanently unreadable.")
		fmt.Println("   To proceed anyway, first delete the existing key or use a different database name.")
		return fmt.Errorf("refusing to overwrite existing encryption key")
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