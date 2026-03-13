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

func init() {
	secretsInitCmd.Flags().String("backend", "", `Key storage backend: "keyring" or "age-ssh" (default: auto-detect)`)
	secretsInitCmd.Flags().String("ssh-public-key", "", "Path to SSH public key for age-ssh backend")
	secretsInitCmd.Flags().String("ssh-private-key", "", "Path to SSH private key for age-ssh backend")
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

	backend, _ := cmd.Flags().GetString("backend")
	if backend != "" && backend != "keyring" && backend != "age-ssh" {
		return fmt.Errorf("invalid backend %q: must be \"keyring\" or \"age-ssh\"", backend)
	}

	sshPublicKey, _ := cmd.Flags().GetString("ssh-public-key")
	sshPrivateKey, _ := cmd.Flags().GetString("ssh-private-key")

	// Get secrets config with defaults
	secretsConfig := cfg.GetSecretsConfig()

	if sshPublicKey != "" {
		secretsConfig.SSHPublicKeyPath = sshPublicKey
	}
	if sshPrivateKey != "" {
		secretsConfig.SSHPrivateKeyPath = sshPrivateKey
	}
	if backend != "" {
		secretsConfig.KeyBackend = backend
	}

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

	// If backend was explicitly set, persist it to config
	if backend != "" {
		cfg.Secrets.KeyBackend = backend
		if sshPublicKey != "" {
			cfg.Secrets.SSHPublicKeyPath = sshPublicKey
		}
		if sshPrivateKey != "" {
			cfg.Secrets.SSHPrivateKeyPath = sshPrivateKey
		}
		if err := cfg.SaveConfig(cfg.Path); err != nil {
			fmt.Printf("⚠ Failed to save backend preference to config: %v\n", err)
		} else {
			fmt.Printf("✓ Backend '%s' recorded in config\n", backend)
		}
	}

	switch backend {
	case "age-ssh":
		fmt.Printf("✓ Encryption key stored as age-ssh file (SSH key: %s)\n", secretsConfig.SSHPublicKeyPath)
	case "keyring":
		if keyPath != "" {
			fmt.Printf("✓ Encryption key loaded from '%s' for database '%s'\n", keyPath, database)
		} else {
			fmt.Printf("✓ New encryption key generated for database '%s'\n", database)
		}
	default:
		if keyPath != "" {
			fmt.Printf("✓ Encryption key loaded from '%s' for database '%s'\n", keyPath, database)
		} else {
			fmt.Printf("✓ New encryption key generated for database '%s'\n", database)
		}
	}

	return nil
}
