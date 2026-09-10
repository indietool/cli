package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indietool/cli/indietool"
	"github.com/indietool/cli/indietool/secrets"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// writeIsolatedSecretsConfig writes a temp config yaml with an isolated secrets
// storage dir and a throwaway age-ssh keypair, so tests never touch the real
// keyring, the developer's ~/.config/indietool, or the @hermes database.
func writeIsolatedSecretsConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	storageDir := filepath.Join(dir, "secrets")

	pubPath := filepath.Join(dir, "test_ed25519.pub")
	privPath := filepath.Join(dir, "test_ed25519")
	keygen := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-q",
		"-f", privPath, "-C", "indietool-test")
	if out, err := keygen.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, out)
	}

	conf := fmt.Sprintf(`secrets:
  default_database: testdb
  storage_dir: %s
  key_backend: age-ssh
  ssh_public_key_path: %s
  ssh_private_key_path: %s
`, storageDir, pubPath, privPath)

	configPath := filepath.Join(dir, "indietool.yaml")
	if err := os.WriteFile(configPath, []byte(conf), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return configPath
}

// useTestConfig points the global appConfig at an isolated config (as
// initConfig would after loading configPath) and restores the original.
func useTestConfig(t *testing.T, configPath string) {
	t.Helper()
	original := appConfig
	t.Cleanup(func() { appConfig = original })

	cfg, err := indietool.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}
	appConfig = cfg
}

// newTestSecretsManager builds a Manager against the current appConfig.
func newTestSecretsManager(t *testing.T) *secrets.Manager {
	t.Helper()
	cfg := GetConfig()
	if cfg == nil {
		t.Fatal("no config loaded")
	}
	mgr, err := secrets.NewManager(cfg.GetSecretsConfig())
	if err != nil {
		t.Fatalf("failed to create secrets manager: %v", err)
	}
	return mgr
}

// testDefaultDB is the default_database written by writeIsolatedSecretsConfig.
const testDefaultDB = "testdb"

// resetLocalFlags clears any locally-set flags on cmd so tests running against
// the shared singleton cobra commands don't leak boolean flags into each other.
func resetLocalFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	})
}

// runSecretsCmd runs the CLI with args and the given stdin, returning captured
// cobra stdout+stderr (error-path messages included) and the Execute error.
func runSecretsCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetIn(nil)
		rootCmd.PersistentFlags().Set("json", "false")
		jsonOutput = false
		resetLocalFlags(secretsSetCmd)
		resetLocalFlags(secretsExecCmd)
	})
	resetLocalFlags(secretsSetCmd)
	resetLocalFlags(secretsExecCmd)

	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()
	return buf.String(), execErr
}

// captureChildStdout redirects os.Stdout (where exec'd children inherit) and
// returns a func that restores it and yields everything written so far.
func captureChildStdout(t *testing.T) func() string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })
	return func() string {
		_ = w.Close()
		out, _ := io.ReadAll(r)
		return string(out)
	}
}

// mustSetSecret stores a secret in the test database via the Manager.
func mustSetSecret(t *testing.T, mgr *secrets.Manager, name, value string) {
	t.Helper()
	if err := mgr.SetSecret(name, value, testDefaultDB, "", nil); err != nil {
		t.Fatalf("SetSecret(%s) failed: %v", name, err)
	}
}
