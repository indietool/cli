package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsDbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets databases",
	Long:  "List all existing secrets databases with their names.",
	RunE:  listDatabases,
}

func listDatabases(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	secretsConfig := cfg.GetSecretsConfig()
	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	databases, err := manager.ListDatabases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		fmt.Println("No secrets databases found.")
		return nil
	}

	defaultDb := secretsConfig.GetDefaultDatabase()
	fmt.Println("Available secrets databases:")
	for _, db := range databases {
		if db == defaultDb {
			fmt.Printf("  %s (default)\n", db)
		} else {
			fmt.Printf("  %s\n", db)
		}
	}

	return nil
}
