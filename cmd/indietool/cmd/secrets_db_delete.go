package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsDbDeleteCmd = &cobra.Command{
	Use:   "delete <database>",
	Short: "Delete a secrets database",
	Long:  "Delete a secrets database and all its secrets. This action is irreversible.",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteDatabase,
}

func init() {
	secretsDbDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")
}

func deleteDatabase(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	database := strings.TrimSpace(args[0])
	if database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	force, _ := cmd.Flags().GetBool("force")

	secretsConfig := cfg.GetSecretsConfig()
	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	// Check if database exists
	databases, err := manager.ListDatabases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	found := false
	for _, db := range databases {
		if db == database {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("database '%s' does not exist", database)
	}

	// Show warning and ask for confirmation unless --force is used
	if !force {
		fmt.Printf("⚠️  WARNING: This will permanently delete the database '%s' and ALL its secrets.\n", database)
		fmt.Println("   This action cannot be undone.")
		fmt.Print("   Type 'yes' to confirm deletion: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" {
			fmt.Println("Database deletion cancelled.")
			return nil
		}
	}

	// Delete the database
	if err := manager.DeleteDatabase(database); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	fmt.Printf("✓ Database '%s' deleted successfully\n", database)
	return nil
}
