package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"indietool/cli/indietool/secrets"
)

// ageMagic is the header written by filippo.io/age for binary-format encrypted files.
const ageMagic = "age-encryption.org/v1"

var secretsImportCmd = &cobra.Command{
	Use:   "import <input-file>",
	Short: "Import secrets from an exported JSON file",
	Long: `Import secrets from a previously exported JSON file into the local instance.

Encrypted exports are detected automatically and will prompt for a passphrase.
Use --force to overwrite secrets that already exist locally.

Use --remap to redirect a source database to a different name on import:
  --remap old:new          rename one database
  --remap a:x --remap b:x  merge two databases into one`,
	Args: cobra.ExactArgs(1),
	RunE: importSecrets,
}

func init() {
	secretsImportCmd.Flags().BoolP("force", "f", false, "Overwrite existing secrets")
	secretsImportCmd.Flags().StringArrayP("remap", "r", nil, "Rename a database on import: old:new (repeatable)")
}

func importSecrets(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	force, _ := cmd.Flags().GetBool("force")
	remapArgs, _ := cmd.Flags().GetStringArray("remap")

	// Parse --remap old:new pairs
	remap, err := parseRemap(remapArgs)
	if err != nil {
		return err
	}

	fileData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	jsonData := fileData
	if bytes.HasPrefix(fileData, []byte(ageMagic)) {
		passphrase, err := promptPassphrase("Enter import passphrase: ", false)
		if err != nil {
			return err
		}
		identity, err := age.NewScryptIdentity(passphrase)
		if err != nil {
			return fmt.Errorf("failed to create passphrase identity: %w", err)
		}
		r, err := age.Decrypt(bytes.NewReader(fileData), identity)
		if err != nil {
			return fmt.Errorf("failed to decrypt import file (wrong passphrase?): %w", err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			return fmt.Errorf("failed to read decrypted data: %w", err)
		}
		jsonData = buf.Bytes()
	}

	var data secrets.ExportData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	// Apply remaps: merge remapped entries, then replace the databases map
	if len(remap) > 0 {
		remapped := make(map[string][]*secrets.Secret, len(data.Databases))
		for src, secretsList := range data.Databases {
			dst := src
			if target, ok := remap[src]; ok {
				dst = target
			}
			remapped[dst] = append(remapped[dst], secretsList...)
		}
		data.Databases = remapped
	}

	secretsConfig := cfg.GetSecretsConfig()
	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	imported, conflicts, err := manager.ImportSecrets(&data, force)
	if err != nil {
		return err
	}

	if len(conflicts) > 0 {
		fmt.Fprintf(os.Stderr, "⚠  Overwrote %d existing %s\n", len(conflicts), plural("secret", len(conflicts)))
	}

	printImportSummary(data.Databases, imported)
	return nil
}

// parseRemap validates and parses --remap old:new flags into a lookup map.
func parseRemap(args []string) (map[string]string, error) {
	remap := make(map[string]string, len(args))
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --remap value %q: expected old:new", arg)
		}
		remap[parts[0]] = parts[1]
	}
	return remap, nil
}

func printImportSummary(databases map[string][]*secrets.Secret, total int) {
	fmt.Println("Imported")
	for _, db := range sortedKeys(databases) {
		n := len(databases[db])
		fmt.Printf("  - @%s: %d %s\n", db, n, plural("secret", n))
	}
}
