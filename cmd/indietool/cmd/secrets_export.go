package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"indietool/cli/indietool/secrets"
)

var secretsExportCmd = &cobra.Command{
	Use:   "export <@db | secret[@db]> [<@db | secret[@db]> ...]",
	Short: "Export secrets to a portable JSON file",
	Long: `Export individual secrets or entire databases to a JSON file.

By default the export is passphrase-encrypted for safe transport.
Use -P / --no-passphrase for plaintext output (e.g. when piping to another tool).

Specify what to export using one or more arguments:
  @<db>           export all secrets from a database
  <name>[@<db>]   export a single secret (omit @db to target the default database)

Examples:
  # Export entire databases
  indietool secret export @default @production --out backup.json

  # Export individual secrets
  indietool secret export stripe-key@production openai-key --out keys.json

  # Mix: entire db plus one secret from another db
  indietool secret export @production mysecret@staging --out backup.json

  # Plaintext output (pipe-friendly)
  indietool secret export @default --no-passphrase | jq .`,
	Args: cobra.MinimumNArgs(1),
	RunE: exportSecrets,
}

func init() {
	secretsExportCmd.Flags().StringP("out", "o", "", "Output file (default: stdout)")
	secretsExportCmd.Flags().BoolP("no-passphrase", "P", false, "Write plaintext JSON instead of passphrase-encrypting the output")
}

func exportSecrets(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}

	noPassphrase, _ := cmd.Flags().GetBool("no-passphrase")
	outFile, _ := cmd.Flags().GetString("out")

	secretsConfig := cfg.GetSecretsConfig()
	defaultDB := secretsConfig.GetDefaultDatabase()

	// Parse args into export spec, deduplicating as we go.
	// spec[db] = nil   → export the whole database
	// spec[db] = names → export only those named secrets
	spec := make(map[string][]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			// @db → whole database
			db := arg[1:]
			if db == "" {
				db = defaultDB
			}
			spec[db] = nil // nil marks "all secrets"
		} else {
			// name[@db] → individual secret
			name, db := secrets.ParseSecretIdentifier(arg)
			if db == "" {
				db = defaultDB
			}
			existing, present := spec[db]
			if present && existing == nil {
				// Already exporting the whole db — this secret is covered
				continue
			}
			// Avoid duplicates within the per-db list
			alreadyListed := false
			for _, n := range existing {
				if n == name {
					alreadyListed = true
					break
				}
			}
			if !alreadyListed {
				spec[db] = append(spec[db], name)
			}
		}
	}

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	data, err := manager.ExportSecrets(spec)
	if err != nil {
		return fmt.Errorf("failed to export secrets: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	var payload []byte
	if noPassphrase {
		payload = jsonData
	} else {
		passphrase, err := promptPassphrase("Enter export passphrase: ", true)
		if err != nil {
			return err
		}
		recipient, err := age.NewScryptRecipient(passphrase)
		if err != nil {
			return fmt.Errorf("failed to create passphrase recipient: %w", err)
		}
		var buf bytes.Buffer
		w, err := age.Encrypt(&buf, recipient)
		if err != nil {
			return fmt.Errorf("failed to initialize encryption: %w", err)
		}
		if _, err := w.Write(jsonData); err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to finalize encryption: %w", err)
		}
		payload = buf.Bytes()
	}

	var out io.Writer = os.Stdout
	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	// Summary always goes to stderr so it doesn't pollute stdout when piping
	printExportSummary(data.Databases)
	return nil
}

func printExportSummary(databases map[string][]*secrets.Secret) {
	fmt.Fprintln(os.Stderr, "Exported")
	for _, db := range sortedKeys(databases) {
		n := len(databases[db])
		fmt.Fprintf(os.Stderr, "  - @%s: %d %s\n", db, n, plural("secret", n))
	}
}

// promptPassphrase reads a passphrase from the terminal without echo.
// When confirm is true it prompts a second time and requires both to match.
func promptPassphrase(prompt string, confirm bool) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	passphrase := string(passBytes)
	if passphrase == "" {
		return "", fmt.Errorf("passphrase cannot be empty")
	}
	if confirm {
		fmt.Fprint(os.Stderr, "Confirm passphrase: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("failed to read passphrase confirmation: %w", err)
		}
		if string(confirmBytes) != passphrase {
			return "", fmt.Errorf("passphrases do not match")
		}
	}
	return passphrase, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func plural(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
