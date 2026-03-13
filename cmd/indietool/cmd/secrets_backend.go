package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"indietool/cli/indietool/secrets"
)

// resolveKeyBackend handles an ErrKeyringUnavailable by printing an explanation
// of the age-ssh backend and, if necessary, prompting the user to choose an SSH
// public key. On success secretsConfig is updated with the chosen key paths and
// KeyBackend is set to "age-ssh" so the caller can retry without hitting the
// keyring again.
func resolveKeyBackend(secretsConfig *secrets.Config, e *secrets.ErrKeyringUnavailable) error {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "The system keyring is not available in this session.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "indietool recommends the age-ssh backend instead.")
	fmt.Fprintln(os.Stderr, "Your database key is encrypted with your SSH public key; only your")
	fmt.Fprintln(os.Stderr, "SSH private key can decrypt it.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Requirements:")
	fmt.Fprintln(os.Stderr, "  • SSH public key must be present on this host")
	fmt.Fprintln(os.Stderr, "  • SSH private key must be accessible (ssh-agent or local key file)")
	fmt.Fprintln(os.Stderr, "  • Remote hosts: connect with agent forwarding  (ssh -A  or  ForwardAgent yes)")
	fmt.Fprintln(os.Stderr)

	selectedPub, err := selectSSHPublicKey(e)
	if err != nil {
		return err
	}

	secretsConfig.SSHPublicKeyPath = selectedPub
	secretsConfig.SSHPrivateKeyPath = strings.TrimSuffix(selectedPub, ".pub")
	secretsConfig.KeyBackend = "age-ssh"
	return nil
}

// selectSSHPublicKey resolves which SSH public key to use.
// If the configured default exists it is returned immediately.
// Otherwise the user is shown a numbered list of keys found in ~/.ssh/ and
// prompted to pick one or enter a custom path.
func selectSSHPublicKey(e *secrets.ErrKeyringUnavailable) (string, error) {
	if e.DefaultExists {
		fmt.Fprintf(os.Stderr, "Using SSH public key: %s\n\n", e.DefaultPubKey)
		return e.DefaultPubKey, nil
	}

	reader := bufio.NewReader(os.Stdin)

	if len(e.AvailableKeys) == 0 {
		fmt.Fprintf(os.Stderr, "No SSH public key found at %s.\n", e.DefaultPubKey)
		fmt.Fprint(os.Stderr, "Enter path to SSH public key: ")
		line, _ := reader.ReadString('\n')
		selected := strings.TrimSpace(line)
		if selected == "" {
			return "", fmt.Errorf("no SSH public key provided")
		}
		return selected, nil
	}

	fmt.Fprintf(os.Stderr, "No SSH public key found at %s.\n", e.DefaultPubKey)
	fmt.Fprintln(os.Stderr, "Available SSH keys:")
	for i, k := range e.AvailableKeys {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, k)
	}
	fmt.Fprintf(os.Stderr, "  [%d] Enter a custom path\n", len(e.AvailableKeys)+1)
	fmt.Fprint(os.Stderr, "Select [1]: ")

	line, _ := reader.ReadString('\n')
	input := strings.TrimSpace(line)

	choice := 1
	if input != "" {
		fmt.Sscanf(input, "%d", &choice) //nolint:errcheck
	}

	if choice >= 1 && choice <= len(e.AvailableKeys) {
		return e.AvailableKeys[choice-1], nil
	}

	// Custom path
	fmt.Fprint(os.Stderr, "Enter path to SSH public key: ")
	line, _ = reader.ReadString('\n')
	selected := strings.TrimSpace(line)
	if selected == "" {
		return "", fmt.Errorf("no SSH public key provided")
	}
	return selected, nil
}
