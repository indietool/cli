package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Long:  "List all secrets in the database. Values are never displayed for security.",
	RunE:  listSecrets,
}

func init() {
	secretsListCmd.Flags().Bool("show-notes", false, "Include notes in the output")
}

func listSecrets(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	showNotes, _ := cmd.Flags().GetBool("show-notes")

	// Get secrets config
	secretsConfig := cfg.GetSecretsConfig()
	database := secretsConfig.GetDefaultDatabase()

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	secretsList, err := manager.ListSecrets(database)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(secretsList) == 0 {
		fmt.Printf("No secrets found in database '%s'\n", database)
		fmt.Println("Use 'indietool secrets set <name> <value>' to add a secret")
		return nil
	}

	// Sort secrets by name
	sort.Slice(secretsList, func(i, j int) bool {
		return secretsList[i].Name < secretsList[j].Name
	})

	// Display results
	fmt.Printf("Secrets in database '%s':\n\n", database)

	// Calculate column widths for better formatting
	maxNameWidth := 4 // "NAME"
	maxNoteWidth := 4 // "NOTE"
	for _, secret := range secretsList {
		if len(secret.Name) > maxNameWidth {
			maxNameWidth = len(secret.Name)
		}
		if showNotes && len(secret.Note) > maxNoteWidth {
			maxNoteWidth = len(secret.Note)
		}
	}

	// Print header
	nameHeader := padRight("NAME", maxNameWidth)
	fmt.Printf("%s  CREATED             UPDATED", nameHeader)
	if showNotes {
		noteHeader := padRight("NOTE", maxNoteWidth)
		fmt.Printf("             %s", noteHeader)
	}
	fmt.Printf("  STATUS\n")

	// Print separator
	fmt.Printf("%s  %s  %s", strings.Repeat("-", maxNameWidth), strings.Repeat("-", 19), strings.Repeat("-", 19))
	if showNotes {
		fmt.Printf("  %s", strings.Repeat("-", maxNoteWidth))
	}
	fmt.Printf("  %s\n", strings.Repeat("-", 6))

	// Print secrets
	for _, secret := range secretsList {
		name := padRight(secret.Name, maxNameWidth)
		created := secret.CreatedAt.Format("2006-01-02 15:04:05")
		updated := secret.UpdatedAt.Format("2006-01-02 15:04:05")

		status := "OK"
		if secret.Expired {
			status = "EXPIRED"
		} else if secret.ExpiresAt != nil {
			status = "EXPIRES"
		}

		fmt.Printf("%s  %s  %s", name, created, updated)
		
		if showNotes {
			note := padRight(secret.Note, maxNoteWidth)
			fmt.Printf("  %s", note)
		}
		
		fmt.Printf("  %s\n", status)
	}

	fmt.Printf("\nTotal: %d secret(s)\n", len(secretsList))

	return nil
}

// padRight pads a string to the specified width with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}