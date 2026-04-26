package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

var secretsShardCmd = &cobra.Command{
	Use:   "shard <file>",
	Short: "Split an encrypted file into recoverable shards",
	Long: `Encrypt and split a file into multiple shards using Shamir's Secret Sharing.

A minimum number of shards (threshold) are needed to recover the original file.
The file is first encrypted with a passphrase, then split into shards.

Set INDIETOOL_SECRET_PASSWORD to avoid interactive prompting.

Examples:
  indietool secret shard secret.txt --shards 5 --threshold 3
  indietool secret shard secret.txt --dir ./shards --prefix vault`,
	Args: cobra.ExactArgs(1),
	RunE: shardSecret,
}

func init() {
	secretsShardCmd.Flags().IntP("shards", "s", secrets.DefaultShards, "Total number of shards to generate")
	secretsShardCmd.Flags().IntP("threshold", "t", secrets.DefaultThreshold, "Minimum shards required to recover the secret")
	secretsShardCmd.Flags().StringP("prefix", "p", secrets.DefaultPrefix, "Prefix for shard file names (e.g., shard.0, shard.1)")
	secretsShardCmd.Flags().StringP("dir", "d", "", "Output directory for shards (default: ./<filename>.shards)")
}

func shardSecret(cmd *cobra.Command, args []string) error {
	filename := args[0]

	n, _ := cmd.Flags().GetInt("shards")
	threshold, _ := cmd.Flags().GetInt("threshold")
	prefix, _ := cmd.Flags().GetString("prefix")
	outDir, _ := cmd.Flags().GetString("dir")

	if threshold > n {
		return fmt.Errorf("threshold (%d) cannot exceed total shards (%d)", threshold, n)
	}
	if n < 2 {
		return fmt.Errorf("shards must be at least 2")
	}
	if threshold < 1 {
		return fmt.Errorf("threshold must be at least 1")
	}

	if outDir == "" {
		outDir = filename + ".shards"
	}

	passphrase := secrets.ReadPassphrase()
	if passphrase == "" {
		var err error
		passphrase, err = promptPassphrase("Enter passphrase to encrypt the secret: ", true)
		if err != nil {
			return err
		}
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	encrypted, err := secrets.EncryptWithPassphrase(data, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	shardData, err := secrets.SplitShards(encrypted, n, threshold)
	if err != nil {
		return fmt.Errorf("failed to split into shards: %w", err)
	}

	written, err := secrets.WriteShards(shardData, prefix, outDir)
	if err != nil {
		return fmt.Errorf("failed to write shards: %w", err)
	}

	if jsonOutput {
		result := map[string]interface{}{
			"status":      "success",
			"file":        filename,
			"shards":      n,
			"threshold":   threshold,
			"output_dir":  outDir,
			"shard_files": written,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Fprintf(os.Stderr, "Sharded %s into %d shards (threshold: %d)\n", filename, n, threshold)
	fmt.Fprintf(os.Stderr, "Shards written to: %s/\n", outDir)
	for _, name := range written {
		fmt.Fprintf(os.Stderr, "  - %s\n", name)
	}
	return nil
}
