package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsJoinCmd = &cobra.Command{
	Use:   "join <shard-directory>",
	Short: "Recover a secret from shards",
	Long: `Recover an encrypted file from a directory of shards.

You need at least the threshold number of shards used during sharding.
The passphrase used during encryption will be required to decrypt.

Set INDIETOOL_SECRET_PASSWORD to avoid interactive prompting.

Examples:
  indietool secret join ./shards
  indietool secret join ./shards --out recovered.txt`,
	Args: cobra.ExactArgs(1),
	RunE: joinSecret,
}

func init() {
	secretsJoinCmd.Flags().StringP("out", "o", "", "Output filename (default: recovered)")
}

func joinSecret(cmd *cobra.Command, args []string) error {
	shardDir := args[0]
	outFile, _ := cmd.Flags().GetString("out")

	if outFile == "" {
		outFile = filepath.Join(shardDir, "recovered")
	} else if !filepath.IsAbs(outFile) {
		outFile = filepath.Join(shardDir, outFile)
	}

	passphrase := secrets.ReadPassphrase()
	if passphrase == "" {
		var err error
		passphrase, err = promptPassphrase("Enter passphrase to decrypt the secret: ", false)
		if err != nil {
			return err
		}
	}

	parts, err := secrets.ReadShards(shardDir)
	if err != nil {
		return fmt.Errorf("failed to read shards: %w", err)
	}

	combined, err := secrets.CombineShards(parts)
	if err != nil {
		return fmt.Errorf("failed to combine shards (need at least the threshold number): %w", err)
	}

	decrypted, err := secrets.DecryptWithPassphrase(combined, passphrase)
	if err != nil {
		return fmt.Errorf("failed to decrypt (wrong passphrase or insufficient shards): %w", err)
	}

	if err := os.WriteFile(outFile, decrypted, 0600); err != nil {
		return fmt.Errorf("failed to write recovered file: %w", err)
	}

	if jsonOutput {
		result := map[string]interface{}{
			"status":      "success",
			"shards_used": len(parts),
			"output_file": outFile,
			"size_bytes":  len(decrypted),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Fprintf(os.Stderr, "Recovered secret from %d shard(s)\n", len(parts))
	fmt.Fprintf(os.Stderr, "Written to: %s\n", outFile)
	return nil
}
