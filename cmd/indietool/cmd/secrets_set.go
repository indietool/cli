package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/indietool/cli/indietool/secrets"
	"github.com/spf13/cobra"
)

var secretsSetCmd = &cobra.Command{
	Use:   "set <name[@database]> [value | --stdin]",
	Short: "Store an encrypted secret",
	Long: `Store an encrypted secret with an optional note. The secret will be encrypted and stored securely. Use name@database to specify a custom database.

Pass the value as an argument, or read it from stdin with --stdin (recommended
for scripts and CI: keeps the value out of shell history and process listings).

Stdin is ingested as raw bytes: nothing is trimmed unless --trim is passed, so
the trailing newline from "echo foo |" is preserved as part of the value — the
classic silent-corruption trap. Pipe with "printf %s" for byte-exact values,
or pass --trim to strip surrounding whitespace.

Examples:
  indietool secret set api-key@hermes "sk-..."
  echo "$API_KEY" | indietool secret set api-key@hermes --stdin
  echo "$TOKEN"   | indietool secret set token@hermes --stdin --trim`,
	Args: cobra.RangeArgs(1, 2),
	RunE: setSecret,
}

type secretSetJSON struct {
	Status    string     `json:"status"`
	Name      string     `json:"name"`
	Database  string     `json:"database"`
	Note      string     `json:"note,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Message   string     `json:"message"`
}

func init() {
	secretsSetCmd.Flags().String("note", "", "Add a note to describe the secret")
	secretsSetCmd.Flags().String("expires", "", "Set expiration date (RFC3339 format: 2025-12-31T23:59:59Z)")
	secretsSetCmd.Flags().Bool("stdin", false, "Read the secret value from stdin instead of an argument (raw bytes; no trailing-newline stripping)")
	secretsSetCmd.Flags().Bool("trim", false, "Trim surrounding whitespace from the value (stdin and argument input)")
}

// resolveSecretValue reads the value from the positional argument or stdin
// (--stdin), enforcing mutual exclusivity and the no-trim-by-default contract.
func resolveSecretValue(cmd *cobra.Command, args []string) (string, error) {
	readStdin, _ := cmd.Flags().GetBool("stdin")
	trim, _ := cmd.Flags().GetBool("trim")

	switch {
	case readStdin && len(args) == 2:
		return "", fmt.Errorf("pass the value either as an argument or via --stdin, not both")
	case readStdin:
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("failed to read secret from stdin: %w", err)
		}
		value := string(raw)
		if trim {
			value = strings.TrimSpace(value)
		}
		if len(value) == 0 {
			return "", fmt.Errorf("no secret value received on stdin")
		}
		return value, nil
	case len(args) == 2:
		value := args[1] // Don't trim value as it might contain intentional whitespace
		if trim {
			value = strings.TrimSpace(value)
		}
		return value, nil
	default:
		return "", fmt.Errorf("no value given: pass the value as an argument or pipe it via --stdin")
	}
}

func setSecret(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	identifier := strings.TrimSpace(args[0])
	if identifier == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	value, err := resolveSecretValue(cmd, args)
	if err != nil {
		return err
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
		var keyringErr *secrets.ErrKeyringUnavailable
		if errors.As(err, &keyringErr) {
			if resolveErr := resolveKeyBackend(secretsConfig, keyringErr); resolveErr != nil {
				return resolveErr
			}
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, "⚠  Using age-ssh for this session. Run 'indietool secrets init --backend age-ssh' to make this permanent.")
				fmt.Fprintln(os.Stderr)
			}
			err = manager.SetSecret(name, value, database, note, expiresAt)
		}
		if err != nil {
			return fmt.Errorf("failed to store secret: %w", err)
		}
	}

	msg := fmt.Sprintf("Secret '%s' stored successfully", name)
	if jsonOutput {
		return printJSON(secretSetJSON{
			Status:    "success",
			Name:      name,
			Database:  database,
			Note:      note,
			ExpiresAt: expiresAt,
			Message:   msg,
		})
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
