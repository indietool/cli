package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsDbRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a secrets database",
	Long:  "Rename a secrets database by copying all its secrets to the new name and removing the old one.",
	Args:  cobra.ExactArgs(2),
	RunE:  renameDatabase,
}

func init() {
	secretsDbRenameCmd.Flags().BoolP("force", "f", false, "Overwrite existing secrets in the target database")
}

func renameDatabase(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	oldName := strings.TrimSpace(args[0])
	newName := strings.TrimSpace(args[1])

	if oldName == "" || newName == "" {
		return fmt.Errorf("database names cannot be empty")
	}
	if oldName == newName {
		return fmt.Errorf("old and new database names are the same")
	}

	force, _ := cmd.Flags().GetBool("force")

	secretsConfig := cfg.GetSecretsConfig()
	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	// Verify source exists
	databases, err := manager.ListDatabases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}
	found := false
	for _, db := range databases {
		if db == oldName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("database %q does not exist", oldName)
	}

	if err := manager.RenameDatabase(oldName, newName, force); err != nil {
		return err
	}

	fmt.Printf("✓ Renamed @%s to @%s\n", oldName, newName)
	return nil
}
